package agentauth_aap_001

import (
	"testing"
	"time"
)

// containsExact performs a substring search.
func containsExactTax(b []byte, s string) bool { return indexOfTax(b, []byte(s)) >= 0 }

// indexOf naive byte slice search
func indexOfTax(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// TestTaxonomyVersionUpgrade ensures moving from Version=2 to Version=3 (single-sig) changes digest (new domain) even without taxonomy values.
func TestTaxonomyVersionUpgrade(t *testing.T) {
	base := &PowerOfAttorney{ID: "tx1", Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(3600, 0).UTC(), CreatedAt: time.Unix(10, 0).UTC(), Signers: []string{"alice"}, Threshold: 1, Version: 2}
	dV2, cV2, err := CanonicalPOADigest(base)
	if err != nil {
		t.Fatalf("digest v2 err: %v", err)
	}
	// Upgrade version only (no taxonomy values provided)
	base.Version = 3
	dV3, cV3, err := CanonicalPOADigest(base)
	if err != nil {
		t.Fatalf("digest v3 err: %v", err)
	}
	if dV2 == dV3 {
		t.Fatalf("expected digest change between V2 and V3 domain")
	}
	if !containsExactTax(cV3, "\"version\":\"3\"") {
		t.Fatalf("missing version=3 field: %s", string(cV3))
	}
	if containsExactTax(cV3, "\"taxonomy\"") {
		t.Fatalf("unexpected taxonomy object when all fields empty: %s", string(cV3))
	}
	if !containsExactTax(cV2, "\"version\":\"2\"") {
		t.Fatalf("missing version=2 field")
	}
}

// TestTaxonomyObjectPresence verifies taxonomy object serialization only for non-empty fields and digest sensitivity.
func TestTaxonomyObjectPresence(t *testing.T) {
	poa := &PowerOfAttorney{ID: "tx2", Grantor: "g1", Grantee: "u1", Scope: []string{"read", "write"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(7200, 0).UTC(), CreatedAt: time.Unix(20, 0).UTC(), Signers: []string{"g1"}, Threshold: 1, Version: 3, AgentType: "human", Sector: "finance"}
	d1, c1, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	if !containsExactTax(c1, "\"taxonomy\":") {
		t.Fatalf("expected taxonomy object present")
	}
	if !containsExactTax(c1, "\"agent_type\":\"human\"") {
		t.Fatalf("agent_type missing: %s", string(c1))
	}
	if !containsExactTax(c1, "\"sector\":\"finance\"") {
		t.Fatalf("sector missing")
	}
	if containsExactTax(c1, "action_class") {
		t.Fatalf("unexpected action_class key when empty")
	}
	// Add action_class -> taxonomy object must now contain it and digest must change.
	poa.ActionClass = "read_ops"
	d2, c2, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest2 err: %v", err)
	}
	if d1 == d2 {
		t.Fatalf("digest unchanged after adding action_class")
	}
	if !containsExactTax(c2, "\"action_class\":\"read_ops\"") {
		t.Fatalf("missing action_class value")
	}
}

// TestTaxonomyMultiSigDomain verifies multi-sig Version>=3 still uses V2 domain (digest change only due to taxonomy fields, not domain).
func TestTaxonomyMultiSigDomain(t *testing.T) {
	poa := &PowerOfAttorney{ID: "tx3", Grantor: "ga", Grantee: "gb", Scope: []string{"exec"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(1800, 0).UTC(), CreatedAt: time.Unix(30, 0).UTC(), Signers: []string{"ga", "gb"}, Threshold: 2, Version: 3, Weights: map[string]int{"ga": 2, "gb": 1}, AgentType: "service"}
	dWithTax, cWithTax, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	if !containsExactTax(cWithTax, "taxonomy") {
		t.Fatalf("expected taxonomy in canonical")
	}
	// Remove taxonomy values (keeping Version 3) -> taxonomy object omitted, digest must change (same V2 domain prefix still).
	poa.AgentType = ""
	dNoTax, cNoTax, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest2 err: %v", err)
	}
	if dWithTax == dNoTax {
		t.Fatalf("digest unchanged after removing taxonomy values")
	}
	if containsExactTax(cNoTax, "taxonomy") {
		t.Fatalf("taxonomy unexpectedly present after clearing values")
	}
	// Downgrade to Version=2 (still multi-sig) -> digest changes again; taxonomy cannot appear.
	poa.Version = 2
	dV2, cV2, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest v2 err: %v", err)
	}
	if dV2 == dNoTax {
		t.Fatalf("expected different digest after version downgrade")
	}
	if containsExactTax(cV2, "taxonomy") {
		t.Fatalf("taxonomy should not appear for Version=2")
	}
}

// TestTaxonomyBackwardCompatibility ensures Version<3 ignores taxonomy fields for canonical digest (no taxonomy object) and domain remains V1 (single-sig) or V2 (multi-sig).
func TestTaxonomyBackwardCompatibility(t *testing.T) {
	// Single-sig legacy Version=2 with taxonomy fields set should not serialize taxonomy (since Version<3) and domain should be V1.
	poa := &PowerOfAttorney{ID: "tx4", Grantor: "ga", Grantee: "gb", Scope: []string{"r"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(1000, 0).UTC(), CreatedAt: time.Unix(40, 0).UTC(), Signers: []string{"ga"}, Threshold: 1, Version: 2, AgentType: "legacy", Sector: "ops", ActionClass: "debug"}
	dLegacy, cLegacy, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest legacy err: %v", err)
	}
	if containsExactTax(cLegacy, "taxonomy") {
		t.Fatalf("taxonomy should not appear for Version<3 canonical JSON")
	}
	// Upgrade to Version=3 -> taxonomy now appears and digest changes.
	poa.Version = 3
	dUp, cUp, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest upgraded err: %v", err)
	}
	if dLegacy == dUp {
		t.Fatalf("digest unchanged after upgrading to V3 with taxonomy inclusion")
	}
	if !containsExactTax(cUp, "taxonomy") {
		t.Fatalf("expected taxonomy object after version upgrade")
	}

	// Multi-sig legacy Version=2 domain baseline.
	ms := &PowerOfAttorney{ID: "tx5", Grantor: "a1", Grantee: "b1", Scope: []string{"x"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(2000, 0).UTC(), CreatedAt: time.Unix(50, 0).UTC(), Signers: []string{"a1", "b1"}, Threshold: 2, Version: 2, Weights: map[string]int{"a1": 2, "b1": 1}, AgentType: "svc"}
	dMsLegacy, cMsLegacy, err := CanonicalPOADigest(ms)
	if err != nil {
		t.Fatalf("digest ms legacy err: %v", err)
	}
	if containsExactTax(cMsLegacy, "taxonomy") {
		t.Fatalf("taxonomy should not appear in multi-sig Version=2")
	}
	// Upgrade version and add taxonomy field -> taxonomy appears and digest changes; domain remains V2 (implicitly via function logic).
	ms.Version = 3
	dMsUp, cMsUp, err := CanonicalPOADigest(ms)
	if err != nil {
		t.Fatalf("digest ms upgraded err: %v", err)
	}
	if dMsLegacy == dMsUp {
		t.Fatalf("digest unchanged after multi-sig taxonomy upgrade")
	}
	if !containsExactTax(cMsUp, "taxonomy") {
		t.Fatalf("expected taxonomy in multi-sig Version=3 canonical")
	}
}
