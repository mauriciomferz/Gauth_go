package rfc0111

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestMultiSigDomainSeparationDeterminism validates that enabling GAUTH_MULTI_SIG_DOMAIN_V2 produces
// identical digests regardless of input weight mapping order and that changing threshold or weights
// changes the digest (domain separation effectiveness).
func TestMultiSigDomainSeparationDeterminism(t *testing.T) {
	// Ensure v2 domain active
	os.Setenv("GAUTH_MULTI_SIG_DOMAIN_V2", "1")
	defer os.Unsetenv("GAUTH_MULTI_SIG_DOMAIN_V2")

	basePOA := &PowerOfAttorney{ID: "p1", Grantor: "g1", Grantee: "g2", Scope: []string{"read", "write"}, Restrictions: map[string]string{"env": "dev"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(3600, 0).UTC(), CreatedAt: time.Unix(5, 0).UTC(), Threshold: 2, Signers: []string{"A", "B", "C"}}

	weights := []string{"A=3", "B=1", "C=2"}
	permuted := [][]string{
		{"A=3", "B=1", "C=2"},
		{"C=2", "A=3", "B=1"},
		{"B=1", "C=2", "A=3"},
	}
	var firstDigest string
	for i, wset := range permuted {
		os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", strings.Join(wset, ","))
		d, _, err := CanonicalPOADigest(basePOA)
		if err != nil {
			t.Fatalf("digest err: %v", err)
		}
		if i == 0 {
			firstDigest = d
		} else if d != firstDigest {
			t.Fatalf("digest changed with weight order: %s vs %s", firstDigest, d)
		}
	}
	// Changing a weight value must change digest
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", "A=4,B=1,C=2")
	dChanged, _, _ := CanonicalPOADigest(basePOA)
	if dChanged == firstDigest {
		t.Fatalf("digest did not change after weight value modification")
	}
	// Changing threshold should also change digest
	basePOA.Threshold = 3
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", strings.Join(weights, ","))
	dThr, _, _ := CanonicalPOADigest(basePOA)
	if dThr == firstDigest {
		t.Fatalf("digest did not change after threshold modification")
	}
	// Disable v2 domain; digest should differ from v2 digest even with same weights/threshold (unless threshold==1)
	os.Unsetenv("GAUTH_MULTI_SIG_DOMAIN_V2")
	// Reset threshold to 2 for fairness
	basePOA.Threshold = 2
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", strings.Join(weights, ","))
	dLegacy, _, _ := CanonicalPOADigest(basePOA)
	if dLegacy == firstDigest {
		t.Fatalf("legacy digest equals v2 digest; domain separation failed")
	}
}

// TestMultiSigWeightsSortingProperty performs randomized weight order permutations to ensure digest stability.
func TestMultiSigWeightsSortingProperty(t *testing.T) {
	os.Setenv("GAUTH_MULTI_SIG_DOMAIN_V2", "1")
	defer os.Unsetenv("GAUTH_MULTI_SIG_DOMAIN_V2")
	basePOA := &PowerOfAttorney{ID: "p2", Grantor: "gg", Grantee: "hh", Scope: []string{"exec"}, Restrictions: map[string]string{}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(7200, 0).UTC(), CreatedAt: time.Unix(7, 0).UTC(), Threshold: 2, Signers: []string{"S1", "S2", "S3", "S4"}}
	canonicalWeights := []string{"S1=5", "S2=1", "S3=3", "S4=2"}
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", strings.Join(canonicalWeights, ","))
	baseDigest, _, err := CanonicalPOADigest(basePOA)
	if err != nil {
		t.Fatalf("base digest err: %v", err)
	}
	// Generate permutations by rotating and swapping pairs
	permSets := [][]string{}
	for i := 0; i < len(canonicalWeights); i++ {
		rot := append([]string{}, canonicalWeights[i:]...)
		rot = append(rot, canonicalWeights[:i]...)
		permSets = append(permSets, rot)
	}
	// Add swapped pairs
	swapped := append([]string{}, canonicalWeights...)
	swapped[0], swapped[1] = swapped[1], swapped[0]
	permSets = append(permSets, swapped)
	for _, wset := range permSets {
		os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", strings.Join(wset, ","))
		d, _, err := CanonicalPOADigest(basePOA)
		if err != nil {
			t.Fatalf("digest err: %v", err)
		}
		if d != baseDigest {
			t.Fatalf("digest changed across weight order permutation: %s != %s (%v)", d, baseDigest, wset)
		}
	}
	// Confirm that altering a weight value changes digest.
	altered := append([]string{}, canonicalWeights...)
	altered[2] = "S3=9"
	sort.Strings(altered) // order doesn't matter, just consistency
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", strings.Join(altered, ","))
	dAlter, _, _ := CanonicalPOADigest(basePOA)
	if dAlter == baseDigest {
		t.Fatalf("digest unchanged after weight value alteration")
	}
}

// randomString and helpers moved to separate test utilities file to avoid duplicate package declaration.

// TestCanonicalDigestDeterministic validates same POA instance produces identical digest across repeated calls.
func TestCanonicalDigestDeterministic(t *testing.T) {
	p := buildRandomPOA(5, 4)
	d1, c1, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	d2, c2, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("digest err2: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("digest mismatch: %s vs %s", d1, d2)
	}
	if string(c1) != string(c2) {
		t.Fatalf("canonical bytes mismatch")
	}
}

// TestCanonicalDigestOrderingInvariance ensures scope/restriction ordering changes do not alter canonical digest (they are sorted).
func TestCanonicalDigestOrderingInvariance(t *testing.T) {
	p := buildRandomPOA(6, 5)
	d1, _, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	// Shuffle scope and restrictions insertion order by reconstructing PoA with reversed slices/map insertion.
	revScope := make([]string, len(p.Scope))
	for i := range p.Scope {
		revScope[i] = p.Scope[len(p.Scope)-1-i]
	}
	revRestrictions := map[string]string{}
	keys := make([]string, 0, len(p.Restrictions))
	for k := range p.Restrictions {
		keys = append(keys, k)
	}
	for i := len(keys) - 1; i >= 0; i-- {
		k := keys[i]
		revRestrictions[k] = p.Restrictions[k]
	}
	p2 := &PowerOfAttorney{ID: p.ID, Grantor: p.Grantor, Grantee: p.Grantee, Scope: revScope, Restrictions: revRestrictions, ValidFrom: p.ValidFrom, ValidUntil: p.ValidUntil, CreatedAt: p.CreatedAt, Status: p.Status, UpdatedAt: p.UpdatedAt}
	d2, _, err := CanonicalPOADigest(p2)
	if err != nil {
		t.Fatalf("digest err2: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("ordering produced different digest: %s vs %s", d1, d2)
	}
}

// TestCanonicalDigestIgnoresMutableFields mutating UpdatedAt or Status must not change digest.
func TestCanonicalDigestIgnoresMutableFields(t *testing.T) {
	p := buildRandomPOA(3, 2)
	d1, _, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	p.Status = "revoked"
	p.UpdatedAt = p.UpdatedAt.Add(time.Minute * 30)
	d2, _, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("digest err2: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("digest changed after mutable field update: %s vs %s", d1, d2)
	}
}

// TestCanonicalDigestDomainSeparation ensures the domain prefix influences output (changing prefix would alter digest).
func TestCanonicalDigestDomainSeparation(t *testing.T) {
	p := buildRandomPOA(2, 1)
	d1, canon, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	// Recompute manually without prefix; should differ.
	// (Copy of algorithm minus domain) for verification only.
	h := sha256Sum(canon) // helper below uses raw SHA256 no domain
	if d1 == h {
		t.Fatalf("digest identical without domain prefix; domain separation ineffective")
	}
}

func sha256Sum(b []byte) string {
	// small helper replicates hash sans domain for TestCanonicalDigestDomainSeparation
	h := sha256.New()
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
