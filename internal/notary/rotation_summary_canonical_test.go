package notary

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

// TestRotationSummaryCanonicalStability ensures that signing uses stable canonical JSON bytes
// so re-signing the same summary with the same key yields identical signature.
func TestRotationSummaryCanonicalStability(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	sum := &RotationSummary{ChainLength: 3, HeadHash: "hh", AggregateHash: "agg", GeneratedAt: "2025-10-20T12:00:00Z"}
	if err := SignRotationSummary(sum, priv, "kid1"); err != nil {
		t.Fatalf("sign first: %v", err)
	}
	sig1 := sum.Signature
	// Recreate an identical struct and sign again
	sum2 := &RotationSummary{ChainLength: 3, HeadHash: "hh", AggregateHash: "agg", GeneratedAt: "2025-10-20T12:00:00Z"}
	if err := SignRotationSummary(sum2, priv, "kid1"); err != nil {
		t.Fatalf("sign second: %v", err)
	}
	sig2 := sum2.Signature
	if sig1 != sig2 {
		t.Fatalf("expected deterministic signature, got %s vs %s", sig1, sig2)
	}
	// Basic verification path
	b1, _ := base64.RawURLEncoding.DecodeString(sig1)
	if len(b1) != ed25519.SignatureSize {
		t.Fatalf("decoded signature size mismatch")
	}
}
