package rfc0111

import (
	"os"
	"testing"
	"time"
)

// TestCanonicalDigestDomainV2 ensures enabling domain v2 alters digest when multi-sig threshold present.
func TestCanonicalDigestDomainV2(t *testing.T) {
	poa := &PowerOfAttorney{ID: "domv2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice", "bob"}, Threshold: 2}
	// Baseline digest (v1 domain)
	os.Unsetenv("GAUTH_MULTI_SIG_DOMAIN_V2")
	os.Unsetenv("GAUTH_MULTI_SIG_WEIGHTS")
	d1, c1, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest v1 err: %v", err)
	}
	if len(d1) == 0 || c1 == nil {
		t.Fatalf("empty digest v1")
	}
	// Enable domain v2 and supply weights to embed
	os.Setenv("GAUTH_MULTI_SIG_DOMAIN_V2", "1")
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", "alice=3,bob=1")
	d2, c2, err2 := CanonicalPOADigest(poa)
	if err2 != nil {
		t.Fatalf("digest v2 err: %v", err2)
	}
	if d1 == d2 {
		t.Fatalf("expected different digest under v2 domain, got same %s", d1)
	}
	if string(c1) != string(c2) {
		t.Fatalf("canonical payload must remain identical; changed unexpectedly")
	}
}
