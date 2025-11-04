package test

import (
	"bytes"
	"crypto/sha256"
	"testing"

	poa "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

func TestRawPOAStreamBasic(t *testing.T) {
	items := []poa.RawPOAItem{
		{ID: "a1", Issuer: "org1", Subject: "alice", Timestamp: 1, Algo: "ed25519", Signature: []byte{0x01, 0x02}},
		{ID: "a2", Issuer: "org1", Subject: "bob", Timestamp: 2, Algo: "ed25519", Signature: []byte{0x03, 0x04}}, // no prev_hash for legacy path
	}
	enc, err := poa.EncodeRawPOAChain(items)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	chain, err := poa.DecodeRawPOAStream(bytes.NewReader(enc), poa.DefaultStreamLimits)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(chain.Items) != 2 {
		t.Fatalf("expected 2 items got %d", len(chain.Items))
	}
	if len(chain.ChainHash) == 0 {
		t.Fatalf("missing chain hash")
	}
}

func TestRawPOAStreamLimits(t *testing.T) {
	// Single item oversize limit
	item := poa.RawPOAItem{ID: "big", Issuer: "i", Subject: "s", Timestamp: 5, Algo: "ed25519", Signature: bytes.Repeat([]byte{0xFF}, 200)}
	enc, _ := poa.EncodeRawPOAChain([]poa.RawPOAItem{item})
	limits := poa.StreamLimits{MaxItems: 1, MaxItemBytes: 64, MaxTotalBytes: 1024}
	_, err := poa.DecodeRawPOAStream(bytes.NewReader(enc), limits)
	if err == nil {
		t.Fatalf("expected size limit error")
	}
}

func TestRawPOAStreamUnexpectedKey(t *testing.T) {
	// Tamper encoded bytes: change a key label
	items := []poa.RawPOAItem{{ID: "t1", Issuer: "x", Subject: "y", Timestamp: 10, Algo: "ed25519", Signature: []byte{0x01}}}
	enc, _ := poa.EncodeRawPOAChain(items)
	// Find 'id' and corrupt to 'iz'
	idx := bytes.Index(enc, []byte("id"))
	if idx == -1 {
		t.Skip("encoding changed – skip")
	}
	enc[idx+1] = 'z'
	_, err := poa.DecodeRawPOAStream(bytes.NewReader(enc), poa.DefaultStreamLimits)
	if err == nil {
		t.Fatalf("expected decode error for unexpected key")
	}
}

func TestRawPOAStreamPrevHashContinuity(t *testing.T) {
	first := poa.RawPOAItem{ID: "p1", Issuer: "org", Subject: "u", Timestamp: 1, Algo: "ed25519", Signature: []byte{0xAA}}
	b1, err := poa.MarshalRawPOAItem(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	h := sha256.Sum256(b1)
	second := poa.RawPOAItem{ID: "p2", Issuer: "org", Subject: "v", Timestamp: 2, Algo: "ed25519", Signature: []byte{0xBB}, PrevHash: h[:]} // continuity
	enc, err := poa.EncodeRawPOAChain([]poa.RawPOAItem{first, second})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	chain, err := poa.DecodeRawPOAStreamWith(bytes.NewReader(enc), poa.DefaultStreamLimits, poa.RawPOAHashSHA256, true)
	if err != nil {
		t.Fatalf("decode with continuity verify: %v", err)
	}
	if len(chain.Items) != 2 {
		t.Fatalf("expected 2 items")
	}
}

func TestRawPOAStreamPrevHashMismatch(t *testing.T) {
	first := poa.RawPOAItem{ID: "x1", Issuer: "org", Subject: "u", Timestamp: 1, Algo: "ed25519", Signature: []byte{0xAA}}
	second := poa.RawPOAItem{ID: "x2", Issuer: "org", Subject: "v", Timestamp: 2, Algo: "ed25519", Signature: []byte{0xBB}, PrevHash: []byte{0x00, 0x01}} // wrong
	enc, _ := poa.EncodeRawPOAChain([]poa.RawPOAItem{first, second})
	_, err := poa.DecodeRawPOAStreamWith(bytes.NewReader(enc), poa.DefaultStreamLimits, poa.RawPOAHashSHA256, true)
	if err == nil {
		t.Fatalf("expected prev_hash mismatch error")
	}
}

func TestRawPOAStreamIndefiniteArray(t *testing.T) {
	// Build two items and encode as indefinite-length array: 0x9f <item1> <item2> 0xff
	first := poa.RawPOAItem{ID: "i1", Issuer: "org", Subject: "a", Timestamp: 1, Algo: "ed25519", Signature: []byte{0x01}}
	b1, _ := poa.MarshalRawPOAItem(first)
	h := sha256.Sum256(b1)
	second := poa.RawPOAItem{ID: "i2", Issuer: "org", Subject: "b", Timestamp: 2, Algo: "ed25519", Signature: []byte{0x02}, PrevHash: h[:]}
	b2, _ := poa.MarshalRawPOAItem(second)
	var buf bytes.Buffer
	buf.WriteByte(0x9f) // start indefinite array
	buf.Write(b1)
	buf.Write(b2)
	buf.WriteByte(0xff) // break
	chain, err := poa.DecodeRawPOAStreamWith(bytes.NewReader(buf.Bytes()), poa.DefaultStreamLimits, poa.RawPOAHashSHA256, true)
	if err != nil {
		t.Fatalf("indefinite decode error: %v", err)
	}
	if len(chain.Items) != 2 {
		t.Fatalf("expected 2 items got %d", len(chain.Items))
	}
	if chain.HashAlgo.String() != "sha256" {
		t.Fatalf("unexpected algo %s", chain.HashAlgo.String())
	}
}
