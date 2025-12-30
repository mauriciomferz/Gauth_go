package gauth_aap_001

import (
	"testing"
	"time"
)

// TestCanonicalDigestDomainV2 ensures enabling domain v2 alters digest when multi-sig threshold present.
func TestCanonicalDigestDomainV2(t *testing.T) {
	// Baseline: single signature (threshold=1) -> V1 domain.
	poa := &PowerOfAttorney{ID: "domv2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice"}, Threshold: 1}
	d1, c1, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest v1 err: %v", err)
	}
	// Activate multi-sig context: add second signer & raise threshold -> automatic V2 domain.
	poa.Signers = []string{"alice", "bob"}
	poa.Threshold = 2
	d2, c2, err2 := CanonicalPOADigest(poa)
	if err2 != nil {
		t.Fatalf("digest v2 err: %v", err2)
	}
	if d1 == d2 {
		t.Fatalf("expected different digest under V2 domain, got same %s", d1)
	}
	// Canonical JSON must remain identical because threshold/signers are domain-only inputs (not serialized).
	if string(c1) != string(c2) {
		t.Fatalf("canonical payload changed unexpectedly")
	}
}
