package main

// Generator for cross-language crypto test vectors.
// Produces a JSON array of objects with fields:
// alg, message_hex, public_hex, signature_hex, valid, (optional) note.
// Algorithms: Ed25519, ECDSA-P256 (canonical low-S + negative variants), BLS12-381.
//
// Determinism:
// - Ed25519: deterministic via seed-derived private key.
// - ECDSA: deterministic private key & deterministic nonce source
//   (pseudo-random HMAC-SHA256 expansion).
// - BLS: uses CSPRNG (library lacks documented deterministic setter here) ->
//   non-deterministic (acceptable for fixture snapshot; rerun will differ).
//
// Usage:
//   go run ./cmd/gen-crypto-vectors > internal/crypto/fixtures/crypto_vectors.json
//
// Re-run when adding algorithms or changing canonical encoding rules.

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	bls "github.com/herumi/bls-eth-go-binary/bls"
)

type vector struct {
	Alg          string `json:"alg"`
	MessageHex   string `json:"message_hex"`
	PublicHex    string `json:"public_hex"`
	SignatureHex string `json:"signature_hex"`
	Valid        bool   `json:"valid"`
	Note         string `json:"note,omitempty"`
}

// deterministicECDSAPriv derives a deterministic P-256 private key from a seed string.
func deterministicECDSAPriv(seed string) *ecdsa.PrivateKey {
	h := sha256.Sum256([]byte(seed))
	d := new(big.Int).SetBytes(h[:])
	curve := elliptic.P256()
	n := curve.Params().N
	d.Mod(d, n)
	if d.Sign() == 0 {
		d.Add(d, big.NewInt(1))
	}
	x, y := curve.ScalarBaseMult(d.Bytes())
	return &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y}, D: d}
}

// deterministicReader generates an infinite deterministic stream via HMAC-SHA256(seed, counter).
type deterministicReader struct {
	seed []byte
	buf  []byte
	ctr  uint64
}

func newDetReader(seed []byte) *deterministicReader { return &deterministicReader{seed: seed} }

func (r *deterministicReader) refill() {
	mac := hmac.New(sha256.New, r.seed)
	ctrBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		ctrBytes[i] = byte(r.ctr >> (56 - 8*i))
	}
	mac.Write(ctrBytes)
	r.buf = mac.Sum(nil)
	r.ctr++
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	total := 0
	for total < len(p) {
		if len(r.buf) == 0 {
			r.refill()
		}
		n := copy(p[total:], r.buf)
		r.buf = r.buf[n:]
		total += n
	}
	return total, nil
}

// encodeDERSignature replicates internal canonical encoding (SEQUENCE of r,s)
// with leading zero for negative avoidance.
func encodeDERSignature(r, s *big.Int) []byte {
	rb := r.Bytes()
	sb := s.Bytes()
	if len(rb) > 0 && rb[0]&0x80 != 0 {
		rb = append([]byte{0x00}, rb...)
	}
	if len(sb) > 0 && sb[0]&0x80 != 0 {
		sb = append([]byte{0x00}, sb...)
	}
	total := 2 + len(rb) + 2 + len(sb)
	out := make([]byte, 0, 2+total)
	out = append(out, 0x30, byte(total))
	out = append(out, 0x02, byte(len(rb)))
	out = append(out, rb...)
	out = append(out, 0x02, byte(len(sb)))
	out = append(out, sb...)
	return out
}

func normalizeLowS(s, n *big.Int) *big.Int {
	half := new(big.Int).Rsh(n, 1)
	if s.Cmp(half) == 1 {
		return new(big.Int).Sub(n, s)
	}
	return s
}

func hexOf(b []byte) string { return hex.EncodeToString(b) }

func main() {
	vectors := []vector{}

	// Ed25519 deterministic
	edSeed := sha256.Sum256([]byte("agentauth-ed25519-fixture-1"))
	edPriv := ed25519.NewKeyFromSeed(edSeed[:32])
	edPub := edPriv.Public().(ed25519.PublicKey)
	edMsg := []byte("agentauth-ed25519-fixture-msg")
	edSig := ed25519.Sign(edPriv, edMsg)
	vectors = append(vectors, vector{
		Alg:          "Ed25519",
		MessageHex:   hexOf(edMsg),
		PublicHex:    hexOf(edPub),
		SignatureHex: hexOf(edSig),
		Valid:        true,
	})

	badEd := append([]byte(nil), edSig...)
	if len(badEd) > 0 {
		badEd[0] ^= 0xFF
	}
	vectors = append(vectors, vector{
		Alg:          "Ed25519",
		MessageHex:   hexOf(edMsg),
		PublicHex:    hexOf(edPub),
		SignatureHex: hexOf(badEd),
		Valid:        false,
		Note:         "mutated first byte",
	})

	// ECDSA-P256 deterministic key + nonce
	ePriv := deterministicECDSAPriv("agentauth-ecdsa-fixture-1")
	eMsg := []byte("agentauth-ecdsa-fixture-msg")
	digest := sha256.Sum256(eMsg)
	detRand := newDetReader([]byte("agentauth-ecdsa-nonce-seed-1"))
	r, s, err := ecdsa.Sign(detRand, ePriv, digest[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ecdsa sign error: %v\n", err)
		os.Exit(1)
	}

	sLow := normalizeLowS(s, ePriv.Params().N)
	der := encodeDERSignature(r, sLow)

	// Uncompressed form: 0x04 || X || Y (32 bytes each for P-256)
	byteLen := (ePriv.PublicKey.Curve.Params().BitSize + 7) / 8
	pubBytes := make([]byte, 1+2*byteLen)
	pubBytes[0] = 0x04
	ePriv.PublicKey.X.FillBytes(pubBytes[1 : 1+byteLen])
	ePriv.PublicKey.Y.FillBytes(pubBytes[1+byteLen:])

	vectors = append(vectors, vector{
		Alg:          "ECDSA-P256",
		MessageHex:   hexOf(eMsg),
		PublicHex:    hexOf(pubBytes),
		SignatureHex: hexOf(der),
		Valid:        true,
	})

	// High-S variant
	highS := new(big.Int).Sub(ePriv.Params().N, sLow)
	derHigh := encodeDERSignature(r, highS)
	vectors = append(vectors, vector{
		Alg:          "ECDSA-P256",
		MessageHex:   hexOf(eMsg),
		PublicHex:    hexOf(pubBytes),
		SignatureHex: hexOf(derHigh),
		Valid:        false,
		Note:         "high-S malleable variant",
	})

	// Truncated variant
	trunc := der[:len(der)/2]
	vectors = append(vectors, vector{
		Alg:          "ECDSA-P256",
		MessageHex:   hexOf(eMsg),
		PublicHex:    hexOf(pubBytes),
		SignatureHex: hexOf(trunc),
		Valid:        false,
		Note:         "truncated signature",
	})

	// BLS12-381 (non-deterministic key) - acceptable for snapshot; mark note.
	if initErr := bls.Init(bls.BLS12_381); initErr == nil {
		var sk bls.SecretKey
		sk.SetByCSPRNG()
		pk := sk.GetPublicKey()
		bMsg := []byte("agentauth-bls-fixture-msg")
		sig := sk.SignByte(bMsg)

		vectors = append(vectors, vector{
			Alg:          "BLS12-381",
			MessageHex:   hexOf(bMsg),
			PublicHex:    hexOf(pk.Serialize()),
			SignatureHex: hexOf(sig.Serialize()),
			Valid:        true,
			Note:         "non-deterministic key (library CSPRNG)",
		})

		bad := append([]byte(nil), sig.Serialize()...)
		if len(bad) > 0 {
			bad[0] ^= 0xAA
		}
		vectors = append(vectors, vector{
			Alg:          "BLS12-381",
			MessageHex:   hexOf(bMsg),
			PublicHex:    hexOf(pk.Serialize()),
			SignatureHex: hexOf(bad),
			Valid:        false,
			Note:         "corrupted first byte",
		})
	} else {
		fmt.Fprintf(os.Stderr, "Skipping BLS vectors: init error: %v\n", initErr)
	}

	enc, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}
	// Note: output is pure JSON (no metadata header).
	_, _ = os.Stdout.Write(enc)

	// Emit warning to stderr if non-deterministic vectors present.
	fmt.Fprintf(os.Stderr, "Generated %d crypto vectors at %s\n", len(vectors), time.Now().UTC().Format(time.RFC3339))
}
