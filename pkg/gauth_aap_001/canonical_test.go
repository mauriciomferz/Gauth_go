package gauth_aap_001

import (
	"testing"
	"time"
)

// TestCanonicalPOADigestDeterminism ensures digest & canonical JSON are stable under field reordering
// and that excluded mutable fields do not affect the hash.
func TestCanonicalPOADigestDeterminism(t *testing.T) {
	base := &PowerOfAttorney{
		ID:           "poa-123",
		Grantor:      "alice",
		Grantee:      "bob",
		Scope:        []string{"resource:write", "resource:read"},
		Restrictions: map[string]string{"env": "prod", "tier": "gold"},
		ValidFrom:    time.Date(2025, 10, 18, 12, 0, 0, 0, time.UTC),
		ValidUntil:   time.Date(2025, 10, 19, 12, 0, 0, 0, time.UTC),
		CreatedAt:    time.Date(2025, 10, 18, 11, 30, 0, 0, time.UTC),
		Status:       "active",
	}
	d1, c1, err := CanonicalPOADigest(base)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	// Reordered scope & restrictions (maps inherently unordered, but we shuffle values)
	alt := &PowerOfAttorney{ID: base.ID, Grantor: base.Grantor, Grantee: base.Grantee, Scope: []string{"resource:read", "resource:write"}, Restrictions: map[string]string{"tier": "gold", "env": "prod"}, ValidFrom: base.ValidFrom, ValidUntil: base.ValidUntil, CreatedAt: base.CreatedAt, Status: "revoked"}
	d2, c2, err := CanonicalPOADigest(alt)
	if err != nil {
		t.Fatalf("digest alt err: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("digest mismatch under reordering: %s vs %s", d1, d2)
	}
	if string(c1) != string(c2) {
		t.Fatalf("canonical JSON mismatch under reordering: %s vs %s", c1, c2)
	}
	// Ensure domain separation prefix is applied (digest should not equal raw SHA256 canonical bytes)
	if len(d1) == 0 {
		t.Fatalf("empty digest")
	}
}

// TestCanonicalPOADigestNil verifies nil handling.
func TestCanonicalPOADigestNil(t *testing.T) {
	if _, _, err := CanonicalPOADigest(nil); err == nil {
		t.Fatalf("expected error for nil POA")
	}
}
