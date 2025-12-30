package gauth_aap_001

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"testing"
	"time"
)

// local copies of small helpers (avoid cross-test coupling): genKey + signPOA
func genKeyWeight(t *testing.T) (priv ed25519.PrivateKey, pub ed25519.PublicKey, keyID string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	h := sha256.Sum256(pub)
	keyID = hex.EncodeToString(h[:6])
	return priv, pub, keyID
}

func signPOAWeight(t *testing.T, poa *PowerOfAttorney, keyID string, priv ed25519.PrivateKey) *POASignature {
	t.Helper()
	dig, canon, derr := CanonicalPOADigest(poa)
	if derr != nil {
		t.Fatalf("canonical digest failed: %v", derr)
	}
	sig := ed25519.Sign(priv, canon)
	return &POASignature{Algorithm: algEd25519, KeyID: keyID, DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig)}
}

// TestWeightedMultiSignatureSuccess ensures cumulative weights meeting threshold succeed.
func TestWeightedMultiSignatureSuccess(t *testing.T) {
	// Configure environment weights: alice=3,bob=2,carol=1 threshold=5 -> need any combination summing >=5
	os.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "alice=3,bob=2,carol=1")
	os.Setenv("AGENTAUTH_MULTI_SIG_THRESHOLD", "5") // threshold interpreted as weight requirement in verify logic
	// Structural rule requires len(Signers) >= Threshold; include placeholder signers d1,d2 (not signed) for structural validity.
	poa := &PowerOfAttorney{ID: "w1", Grantor: "grantor", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol", "d1", "d2"}, Threshold: 5, MultiSignatures: map[string]*POASignature{}}
	// Keys per signer
	privA, pubA, keyIDA := genKeyWeight(t)
	privB, pubB, keyIDB := genKeyWeight(t)
	_, pubC, keyIDC := genKeyWeight(t)
	// provider with all publics
	prov := &testProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB, keyIDC: pubC}, active: &testSigner{keyID: keyIDA, priv: privA}}
	// Map signer logical IDs to keyIDs for this test (signers slice contains logical names; we will use keyIDs in signatures)
	// Provide signatures for alice (3) and bob (2) -> total 5 satisfies threshold
	poa.MultiSignatures["alice"] = signPOAWeight(t, poa, keyIDA, privA)
	poa.MultiSignatures["bob"] = signPOAWeight(t, poa, keyIDB, privB)
	svc := NewService(nil, authzAllowAll{}, WithKeyProvider(prov), WithStrictAuthenticity())
	if err := svc.verifyMultiSignatures(poa); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

// TestWeightedMultiSignatureInsufficient ensures failure when cumulative weight below threshold.
func TestWeightedMultiSignatureInsufficient(t *testing.T) {
	os.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "alice=3,bob=2,carol=1")
	os.Setenv("AGENTAUTH_MULTI_SIG_THRESHOLD", "6") // require weight 6
	// Include placeholders to satisfy structural threshold (need >=6 signers) -> add d1,d2,d3
	poa := &PowerOfAttorney{ID: "w2", Grantor: "grantor", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob", "carol", "d1", "d2", "d3"}, Threshold: 6, MultiSignatures: map[string]*POASignature{}}
	privA, pubA, keyIDA := genKeyWeight(t)
	privB, pubB, keyIDB := genKeyWeight(t)
	_, pubC, keyIDC := genKeyWeight(t)
	prov := &testProvider{pubs: map[string]ed25519.PublicKey{keyIDA: pubA, keyIDB: pubB, keyIDC: pubC}, active: &testSigner{keyID: keyIDA, priv: privA}}
	// alice(3) + bob(2) => total 5 < 6 -> insufficient_weight_valid error expected
	poa.MultiSignatures["alice"] = signPOAWeight(t, poa, keyIDA, privA)
	poa.MultiSignatures["bob"] = signPOAWeight(t, poa, keyIDB, privB)
	svc := NewService(nil, authzAllowAll{}, WithKeyProvider(prov), WithStrictAuthenticity())
	err := svc.verifyMultiSignatures(poa)
	if err == nil {
		t.Fatalf("expected failure due to insufficient weight")
	}
	if err.Error() == "" || (err != nil && !containsSubstring(err.Error(), "insufficient_weight_valid")) {
		t.Fatalf("expected insufficient_weight_valid error, got: %v", err)
	}
}

// containsSubstring tiny helper (avoid importing strings for single usage) - local implementation.
func containsSubstring(haystack, needle string) bool {
	// naive search
	hl := len(haystack)
	nl := len(needle)
	if nl == 0 {
		return true
	}
	if nl > hl {
		return false
	}
	for i := 0; i <= hl-nl; i++ {
		if haystack[i:i+nl] == needle {
			return true
		}
	}
	return false
}
