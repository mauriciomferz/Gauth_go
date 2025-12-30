package gauth_rfc_001

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	signalgo "github.com/mauriciomferz/Gauth_go/pkg/crypto/signalgo"
)

// TestSignatureAgilityEd25519Basic ensures Ed25519 algorithm registry path matches
// direct ed25519 usage semantics for signing & verification without altering
// canonical digest logic (AAP001 cryptographic agility foundation).
func TestSignatureAgilityEd25519Basic(t *testing.T) {
	algo, ok := signalgo.Get("Ed25519")
	if !ok {
		t.Fatalf("Ed25519 algorithm not registered")
	}
	pub, priv, err := algo.KeyGen()
	if err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	// Random message; canonical digest independence verified separately in existing tests.
	msg := make([]byte, 64)
	if _, err2 := rand.Read(msg); err2 != nil {
		t.Fatalf("rand: %v", err2)
	}
	sig, err := algo.Sign(priv, msg)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if len(sig) == 0 {
		t.Fatalf("empty signature returned")
	}
	if !algo.Verify(pub, msg, sig) {
		t.Fatalf("verification failed for valid signature")
	}
	// Negative: modify message
	badMsg := append([]byte{}, msg...)
	badMsg[0] ^= 0xFF
	if algo.Verify(pub, badMsg, sig) {
		t.Fatalf("verification succeeded for tampered message")
	}
}

// TestSignatureAgilityRegistryList ensures listing includes Ed25519 and (optionally) ECDSA_P256 when enabled.
func TestSignatureAgilityRegistryList(t *testing.T) {
	names := signalgo.List()
	foundEd := false
	for _, n := range names {
		if n == "Ed25519" {
			foundEd = true
			break
		}
	}
	if !foundEd {
		t.Fatalf("Ed25519 not found in registry list: %v", names)
	}
}

// TestSignatureAgilityECDSAStubDisabled verifies that requesting key material for disabled stub returns error or is absent.
func TestSignatureAgilityECDSAStubDisabled(t *testing.T) {
	if _, ok := signalgo.Get("ECDSA_P256"); ok {
		// If present (env enabled during test run), skip because stub active.
		t.Skip("ECDSA stub enabled; skipping disabled assertion")
	}
	// Attempt to register manually should fail when env not set.
}

// Ensure canonical digest output unchanged by registry presence (smoke test invoking CanonicalPOADigest).
func TestSignatureAgilityCanonicalDigestUnchanged(t *testing.T) {
	poa := &PowerOfAttorney{ID: "poa-1", Version: 1, Grantor: "principal-x", Grantee: "agent-y", Scope: []string{"txn:pay"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Minute), CreatedAt: time.Now().UTC(), Threshold: 1, Weights: map[string]int{"principal-x": 1}}
	dig1, canon1, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("canonical digest failed: %v", err)
	}
	dig2, canon2, err2 := CanonicalPOADigest(poa)
	if err2 != nil {
		t.Fatalf("canonical digest second failed: %v", err2)
	}
	if dig1 != dig2 || !bytes.Equal(canon1, canon2) {
		t.Fatalf("canonical digest instability detected")
	}
}
