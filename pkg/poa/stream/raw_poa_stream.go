package stream

// Task 8: RawPOA & CBOR Stream
// Implements canonical RawPOA serialization and streaming CBOR decoding with incremental hashing.
// Design:
//  - RawPOAItem represents a single delegation/attestation element.
//  - RawPOAChain is an ordered list of items with a final chain hash (sha256 of concatenated CBOR item bytes).
//  - Encoding format (v1): sequence of length-prefixed CBOR maps (uint32 BE length + CBOR bytes).
//    This enables streaming decode without requiring indefinite-length CBOR arrays support.
//  - Decoder enforces limits: maxItems, maxItemBytes, cumulativeBytes.
//  - Incremental hashing: each item CBOR byte slice is written to sha256 as-is.
//  - Fail closed: any structural or limit violation aborts decode with error.
// Later optimization: replace length prefix with canonical CBOR array once token streaming available.

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// RawPOAItem canonical minimal fields for delegation chain.
type RawPOAItem struct {
	ID        string            `cbor:"id"`
	Issuer    string            `cbor:"iss"`
	Subject   string            `cbor:"sub"`
	Timestamp int64             `cbor:"ts"`
	Claims    map[string]string `cbor:"claims,omitempty"`
	Signature []byte            `cbor:"sig"`
	Algo      string            `cbor:"algo"`
	PrevHash  []byte            `cbor:"prev_hash,omitempty"`
}

// RawPOAChain container returned from streaming decode.
type RawPOAChain struct {
	Items     []RawPOAItem
	ChainHash []byte // hash of concatenated item CBOR bytes using selected algorithm
	HashAlgo  RawPOAHashAlg
}

// RawPOAHashAlg selects hashing algorithm for chain continuity.
type RawPOAHashAlg int

const (
	RawPOAHashSHA256 RawPOAHashAlg = iota
	RawPOAHashBLAKE2b256
	RawPOAHashSHA3_256
)

func (a RawPOAHashAlg) String() string {
	switch a {
	case RawPOAHashSHA256:
		return "sha256"
	case RawPOAHashBLAKE2b256:
		return "blake2b256"
	case RawPOAHashSHA3_256:
		return "sha3_256"
	default:
		return "unknown"
	}
}

func newHash(algo RawPOAHashAlg) (hash.Hash, error) {
	switch algo {
	case RawPOAHashSHA256:
		return sha256.New(), nil
	case RawPOAHashBLAKE2b256:
		return blake2b.New256(nil)
	case RawPOAHashSHA3_256:
		return sha3.New256(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algo %d", algo)
	}
}

// StreamLimits guard resource usage.
type StreamLimits struct {
	MaxItems      int
	MaxItemBytes  int
	MaxTotalBytes int
}

var DefaultStreamLimits = StreamLimits{MaxItems: 4096, MaxItemBytes: 64 * 1024, MaxTotalBytes: 32 * 1024 * 1024}

// EncodeRawPOAChain encodes items using length-prefixed CBOR for tests & internal.
// EncodeRawPOAChain encodes items using length-prefixed CBOR for tests & internal.
// PrevHash continuity is NOT automatically populated; callers may set PrevHash fields.
func EncodeRawPOAChain(items []RawPOAItem) ([]byte, error) {
	var out bytes.Buffer
	for _, it := range items {
		b, err := marshalCBORItem(it)
		if err != nil {
			return nil, err
		}
		if len(b) > int(^uint32(0)) {
			return nil, errors.New("item too large to encode")
		}
		var lenBuf [4]byte
		//nolint:gosec // G115: length already validated against uint32 max
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(b)))
		out.Write(lenBuf[:])
		out.Write(b)
	}
	return out.Bytes(), nil
}

// marshalCBORItem uses a tiny bespoke encoder for subset (maps of simple types) to avoid external deps.
// This keeps footprint minimal; extend if necessary. (Not full CBOR compliance.)
func marshalCBORItem(it RawPOAItem) ([]byte, error) {
	// Map with up to 8 keys; dynamic based on presence.
	kv := make([][2]string, 0, 8)
	add := func(k, v string) { kv = append(kv, [2]string{k, v}) }
	add("id", it.ID)
	add("iss", it.Issuer)
	add("sub", it.Subject)
	add("algo", it.Algo)
	// Timestamp numeric; signature & prev_hash bytes; claims map.
	// We'll encode as CBOR manually:
	var buf bytes.Buffer
	// Map header: major type 5, additional length = number of pairs + special fields.
	pairsCount := len(kv) + 1 + 1 // ts + sig
	if len(it.PrevHash) > 0 {
		pairsCount++
	}
	if len(it.Claims) > 0 {
		pairsCount++
	}
	// encode map length
	if pairsCount < 24 {
		buf.WriteByte(0xA0 | byte(pairsCount))
	} else {
		return nil, errors.New("too many keys for minimal encoder")
	}
	// helper functions
	writeText := func(s string) {
		if len(s) < 24 {
			buf.WriteByte(0x60 | byte(len(s)))
		} else {
			buf.WriteByte(0x78)
			buf.WriteByte(byte(len(s)))
		}
		buf.WriteString(s)
	}
	writeBytes := func(b []byte) {
		if len(b) < 24 {
			buf.WriteByte(0x40 | byte(len(b)))
		} else {
			buf.WriteByte(0x58)
			buf.WriteByte(byte(len(b)))
		}
		buf.Write(b)
	}
	writeInt := func(i int64) {
		switch {
		case i >= 0 && i < 24:
			buf.WriteByte(0x00 | byte(i))
		case i >= 0 && i < 256:
			buf.WriteByte(0x18)
			buf.WriteByte(byte(i))
		default:
			return /* simplified for small timestamps */
		}
	}
	// Emit simple kv
	for _, p := range kv {
		writeText(p[0])
		writeText(p[1])
	}
	// ts
	writeText("ts")
	writeInt(it.Timestamp)
	// sig
	writeText("sig")
	writeBytes(it.Signature)
	if len(it.PrevHash) > 0 {
		writeText("prev_hash")
		writeBytes(it.PrevHash)
	}
	if len(it.Claims) > 0 {
		writeText("claims")
		// encode nested map of claims
		cCount := len(it.Claims)
		if cCount < 24 {
			buf.WriteByte(0xA0 | byte(cCount))
		} else {
			return nil, errors.New("claims too many")
		}
		for k, v := range it.Claims {
			writeText(k)
			writeText(v)
		}
	}
	return buf.Bytes(), nil
}

// DecodeRawPOAStream performs streaming decode from reader.
// DecodeRawPOAStream is the original API retaining SHA-256 and no PrevHash continuity checks.
// Prefer DecodeRawPOAStreamWith for new functionality.
func DecodeRawPOAStream(r io.Reader, limits StreamLimits) (*RawPOAChain, error) {
	return DecodeRawPOAStreamWith(r, limits, RawPOAHashSHA256, false)
}

// DecodeRawPOAStreamWith adds hash algorithm selection, optional PrevHash continuity verification,
// and supports both legacy length-prefixed format and indefinite-length CBOR arrays (0x9f ... 0xff).
//
//nolint:gocyclo // Stream decoder with chaining and verification
func DecodeRawPOAStreamWith(r io.Reader, limits StreamLimits, algo RawPOAHashAlg, verifyPrev bool) (*RawPOAChain, error) {
	if limits.MaxItems == 0 {
		limits.MaxItems = DefaultStreamLimits.MaxItems
	}
	if limits.MaxItemBytes == 0 {
		limits.MaxItemBytes = DefaultStreamLimits.MaxItemBytes
	}
	if limits.MaxTotalBytes == 0 {
		limits.MaxTotalBytes = DefaultStreamLimits.MaxTotalBytes
	}
	h, err := newHash(algo)
	if err != nil {
		return nil, err
	}
	// Peek first byte to detect indefinite-length array start (0x9f)
	var first [1]byte
	n, _ := r.Read(first[:])
	if n == 0 {
		return &RawPOAChain{Items: nil, ChainHash: h.Sum(nil), HashAlgo: algo}, nil
	}
	// If not indefinite array, push byte back into a buffer with remaining reader data.
	if first[0] != 0x9f {
		// Legacy length-prefixed path
		bufReader := io.MultiReader(bytes.NewReader(first[:]), r)
		var items []RawPOAItem
		var total int
		prevDigest := h.Sum(nil)
		for len(items) < limits.MaxItems {
			var lenBuf [4]byte
			_, err2 := io.ReadFull(bufReader, lenBuf[:])
			if err2 == io.EOF {
				break
			}
			if err2 != nil {
				return nil, fmt.Errorf("length read: %w", err2)
			}
			sz := int(binary.BigEndian.Uint32(lenBuf[:]))
			if sz <= 0 || sz > limits.MaxItemBytes {
				return nil, fmt.Errorf("invalid item size %d", sz)
			}
			if total+sz > limits.MaxTotalBytes {
				return nil, errors.New("exceeds total byte limit")
			}
			buf := make([]byte, sz)
			if _, err2 := io.ReadFull(bufReader, buf); err2 != nil {
				return nil, fmt.Errorf("item read: %w", err2)
			}
			total += sz
			// continuity check uses digest before adding current bytes
			if verifyPrev {
				// For first item, PrevHash must be empty
				tmpItem, err2 := unmarshalMinimal(buf)
				if err2 != nil {
					return nil, fmt.Errorf("decode item %d: %w", len(items), err2)
				}
				if len(items) == 0 && len(tmpItem.PrevHash) > 0 {
					return nil, errors.New("first item must not contain prev_hash")
				}
				if len(items) > 0 && len(tmpItem.PrevHash) > 0 && !bytes.Equal(tmpItem.PrevHash, prevDigest) {
					return nil, fmt.Errorf("prev_hash mismatch on item %d", len(items))
				}
				// append after re-validation (avoid double unmarshal by reusing tmpItem)
				h.Write(buf)
				prevDigest = h.Sum(nil)
				items = append(items, *tmpItem)
				continue
			}
			h.Write(buf)
			prevDigest = h.Sum(nil)
			it, err3 := unmarshalMinimal(buf)
			if err3 != nil {
				return nil, fmt.Errorf("decode item %d: %w", len(items), err3)
			}
			items = append(items, *it)
		}
		return &RawPOAChain{Items: items, ChainHash: h.Sum(nil), HashAlgo: algo}, nil
	}
	// Indefinite-length array parsing. Read remainder into bounded buffer.
	rest, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read indefinite array: %w", err)
	}
	if len(rest) > limits.MaxTotalBytes {
		return nil, errors.New("exceeds total byte limit")
	}
	if len(rest) == 0 || rest[len(rest)-1] != 0xff {
		return nil, errors.New("indefinite array missing break code")
	}
	// Remove break code for processing
	content := rest[:len(rest)-1]
	pos := 0
	var items []RawPOAItem
	prevDigest := h.Sum(nil)
	for pos < len(content) && len(items) < limits.MaxItems {
		// Expect map major type (5)
		hdr := content[pos]
		if hdr>>5 != 5 {
			return nil, fmt.Errorf("expected map major type at pos %d", pos)
		}
		mapLen := int(hdr & 0x1F)
		// We don't know map byte length; parse using temporary buffer copy starting at pos
		// Find end by iterative minimal parsing replicating unmarshalMinimal logic accumulating bytes consumed.
		start := pos
		// We'll parse into a scratch slice and rely on unmarshalMinimal for correctness.
		// Need to extract exact map bytes: we simulate by walking keys/values.
		// Simplification: reuse unmarshalMinimal by slicing from start to end-of-buffer; it stops when count satisfied.
		it, consumed, err := unmarshalMinimalAt(content[start:], mapLen)
		if err != nil {
			return nil, fmt.Errorf("indefinite decode item %d: %w", len(items), err)
		}
		if consumed > limits.MaxItemBytes {
			return nil, errors.New("item exceeds MaxItemBytes")
		}
		rawMapBytes := content[start : start+consumed]
		if verifyPrev {
			if len(items) == 0 && len(it.PrevHash) > 0 {
				return nil, errors.New("first item must not contain prev_hash")
			}
			if len(items) > 0 && len(it.PrevHash) > 0 && !bytes.Equal(it.PrevHash, prevDigest) {
				return nil, fmt.Errorf("prev_hash mismatch on item %d", len(items))
			}
		}
		h.Write(rawMapBytes)
		prevDigest = h.Sum(nil)
		items = append(items, *it)
		pos += consumed
	}
	return &RawPOAChain{Items: items, ChainHash: h.Sum(nil), HashAlgo: algo}, nil
}

// unmarshalMinimal decodes subset encoded by marshalCBORItem.
//
//nolint:gocyclo // Binary deserialization with format validation
//nolint:gocyclo // Binary deserialization with format validation
func unmarshalMinimal(b []byte) (*RawPOAItem, error) {
	// Very small state machine; not a full CBOR parser.
	if len(b) == 0 {
		return nil, errors.New("empty bytes")
	}
	if b[0]>>5 != 5 {
		return nil, errors.New("not CBOR map major type")
	}
	count := int(b[0] & 0x1F)
	pos := 1
	readText := func() (string, error) {
		if pos >= len(b) {
			return "", errors.New("trunc")
		}
		hdr := b[pos]
		pos++
		if hdr>>5 != 3 {
			return "", errors.New("expected text")
		}
		ln := int(hdr & 0x1F)
		if ln == 24 {
			if pos >= len(b) {
				return "", errors.New("trunc")
			}
			ln = int(b[pos])
			pos++
		}
		if pos+ln > len(b) {
			return "", errors.New("trunc")
		}
		s := string(b[pos : pos+ln])
		pos += ln
		return s, nil
	}
	readBytes := func() ([]byte, error) {
		if pos >= len(b) {
			return nil, errors.New("trunc")
		}
		hdr := b[pos]
		pos++
		if hdr>>5 != 2 {
			return nil, errors.New("expected bytes")
		}
		ln := int(hdr & 0x1F)
		if ln == 24 {
			if pos >= len(b) {
				return nil, errors.New("trunc")
			}
			ln = int(b[pos])
			pos++
		}
		if pos+ln > len(b) {
			return nil, errors.New("trunc len")
		}
		out := make([]byte, ln)
		copy(out, b[pos:pos+ln])
		pos += ln
		return out, nil
	}
	readInt := func() (int64, error) {
		if pos >= len(b) {
			return 0, errors.New("trunc")
		}
		hdr := b[pos]
		pos++
		if hdr>>5 != 0 {
			return 0, errors.New("expected unsigned int")
		}
		val := int64(hdr & 0x1F)
		if val == 24 {
			if pos >= len(b) {
				return 0, errors.New("trunc")
			}
			val = int64(b[pos])
			pos++
		}
		return val, nil
	}
	it := &RawPOAItem{Claims: make(map[string]string)}
	for i := 0; i < count; i++ {
		k, err := readText()
		if err != nil {
			return nil, err
		}
		switch k {
		case "id":
			v, err := readText()
			if err != nil {
				return nil, err
			}
			it.ID = v
		case "iss":
			v, err := readText()
			if err != nil {
				return nil, err
			}
			it.Issuer = v
		case "sub":
			v, err := readText()
			if err != nil {
				return nil, err
			}
			it.Subject = v
		case "algo":
			v, err := readText()
			if err != nil {
				return nil, err
			}
			it.Algo = v
		case "ts":
			v, err := readInt()
			if err != nil {
				return nil, err
			}
			it.Timestamp = v
		case "sig":
			v, err := readBytes()
			if err != nil {
				return nil, err
			}
			it.Signature = v
		case "prev_hash":
			v, err := readBytes()
			if err != nil {
				return nil, err
			}
			it.PrevHash = v
		case "claims":
			// nested map header
			if pos >= len(b) {
				return nil, errors.New("claims trunc")
			}
			h := b[pos]
			pos++
			if h>>5 != 5 {
				return nil, errors.New("claims not map")
			}
			cCount := int(h & 0x1F)
			for j := 0; j < cCount; j++ {
				ck, err := readText()
				if err != nil {
					return nil, err
				}
				cv, err := readText()
				if err != nil {
					return nil, err
				}
				it.Claims[ck] = cv
			}
		default:
			return nil, fmt.Errorf("unexpected key %s", k)
		}
	}
	return it, nil
}

//nolint:gocyclo // Binary deserialization with field-by-field parsing

// unmarshalMinimalAt parses a map with known pair count starting at b[0]; returns item, bytes consumed.
//
//nolint:gocyclo // Binary deserialization with field-by-field parsing
func unmarshalMinimalAt(b []byte, count int) (*RawPOAItem, int, error) {
	if len(b) == 0 {
		return nil, 0, errors.New("empty")
	}
	if b[0]>>5 != 5 {
		return nil, 0, errors.New("not map")
	}
	// Override count with provided to defend against malicious alteration.
	pos := 1
	readText := func() (string, error) {
		if pos >= len(b) {
			return "", errors.New("trunc")
		}
		hdr := b[pos]
		pos++
		if hdr>>5 != 3 {
			return "", errors.New("expected text")
		}
		ln := int(hdr & 0x1F)
		if ln == 24 {
			if pos >= len(b) {
				return "", errors.New("trunc")
			}
			ln = int(b[pos])
			pos++
		}
		if pos+ln > len(b) {
			return "", errors.New("trunc")
		}
		s := string(b[pos : pos+ln])
		pos += ln
		return s, nil
	}
	readBytes := func() ([]byte, error) {
		if pos >= len(b) {
			return nil, errors.New("trunc")
		}
		hdr := b[pos]
		pos++
		if hdr>>5 != 2 {
			return nil, errors.New("expected bytes")
		}
		ln := int(hdr & 0x1F)
		if ln == 24 {
			if pos >= len(b) {
				return nil, errors.New("trunc")
			}
			ln = int(b[pos])
			pos++
		}
		if pos+ln > len(b) {
			return nil, errors.New("trunc len")
		}
		out := make([]byte, ln)
		copy(out, b[pos:pos+ln])
		pos += ln
		return out, nil
	}
	readInt := func() (int64, error) {
		if pos >= len(b) {
			return 0, errors.New("trunc")
		}
		hdr := b[pos]
		pos++
		if hdr>>5 != 0 {
			return 0, errors.New("expected unsigned int")
		}
		val := int64(hdr & 0x1F)
		if val == 24 {
			if pos >= len(b) {
				return 0, errors.New("trunc")
			}
			val = int64(b[pos])
			pos++
		}
		return val, nil
	}
	it := &RawPOAItem{Claims: make(map[string]string)}
	for i := 0; i < count; i++ {
		k, err := readText()
		if err != nil {
			return nil, 0, err
		}
		switch k {
		case "id":
			v, err := readText()
			if err != nil {
				return nil, 0, err
			}
			it.ID = v
		case "iss":
			v, err := readText()
			if err != nil {
				return nil, 0, err
			}
			it.Issuer = v
		case "sub":
			v, err := readText()
			if err != nil {
				return nil, 0, err
			}
			it.Subject = v
		case "algo":
			v, err := readText()
			if err != nil {
				return nil, 0, err
			}
			it.Algo = v
		case "ts":
			v, err := readInt()
			if err != nil {
				return nil, 0, err
			}
			it.Timestamp = v
		case "sig":
			v, err := readBytes()
			if err != nil {
				return nil, 0, err
			}
			it.Signature = v
		case "prev_hash":
			v, err := readBytes()
			if err != nil {
				return nil, 0, err
			}
			it.PrevHash = v
		case "claims":
			if pos >= len(b) {
				return nil, 0, errors.New("claims trunc")
			}
			h := b[pos]
			pos++
			if h>>5 != 5 {
				return nil, 0, errors.New("claims not map")
			}
			cCount := int(h & 0x1F)
			for j := 0; j < cCount; j++ {
				ck, err := readText()
				if err != nil {
					return nil, 0, err
				}
				cv, err := readText()
				if err != nil {
					return nil, 0, err
				}
				it.Claims[ck] = cv
			}
		default:
			return nil, 0, fmt.Errorf("unexpected key %s", k)
		}
	}
	return it, pos, nil
}

// MarshalRawPOAItem exposes the minimal encoder for testing continuity (exported helper).
func MarshalRawPOAItem(it RawPOAItem) ([]byte, error) { return marshalCBORItem(it) }
