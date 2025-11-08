package rfc0111

import (
	"fmt"
	"testing"
	"time"
)

// TestCanonicalVersionWeightsPresence ensures canonical JSON contains version always and weights only when provided.
func TestCanonicalVersionWeightsPresence(t *testing.T) {
	// Single-sig POA (threshold=1) => no weights domain, Version always present.
	poa := &PowerOfAttorney{ID: "vw1", Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(), Signers: []string{"alice"}, Threshold: 1, Version: 1}
	d1, c1, err := CanonicalPOADigest(poa)
	if err != nil {
		t.Fatalf("digest single err: %v", err)
	}
	if !containsExact(c1, "\"version\":\"1\"") {
		t.Fatalf("canonical missing version field: %s", string(c1))
	}
	if containsExact(c1, "\"weights\"") {
		t.Fatalf("single-sig canonical unexpectedly contains weights: %s", string(c1))
	}
	if len(d1) == 0 {
		t.Fatalf("empty digest")
	}

	// Multi-sig with weights: ensure weights serialized (values as strings) and digest changes when weight changes.
	poa.Signers = []string{"alice", "carol"}
	poa.Threshold = 2
	poa.Weights = map[string]int{"alice": 3, "carol": 1}
	d2, c2, err2 := CanonicalPOADigest(poa)
	if err2 != nil {
		t.Fatalf("digest multi err: %v", err2)
	}
	if d1 == d2 {
		t.Fatalf("expected different digest for multi-sig domain vs single-sig")
	}
	if !containsExact(c2, "\"weights\":{\"alice\":\"3\",\"carol\":\"1\"}") && !containsExactPermutedWeights(c2, poa.Weights) {
		t.Fatalf("canonical missing expected weights object: %s", string(c2))
	}

	// Modify a weight -> digest must change, canonical must reflect new value.
	poa.Weights["carol"] = 5
	d3, c3, err3 := CanonicalPOADigest(poa)
	if err3 != nil {
		t.Fatalf("digest modified weights err: %v", err3)
	}
	if d2 == d3 {
		t.Fatalf("digest unchanged after weight modification")
	}
	if !containsExact(c3, "\"carol\":\"5\"") {
		t.Fatalf("updated weight not reflected in canonical: %s", string(c3))
	}
}

// containsExact does a substring search using byte slice.
func containsExact(b []byte, s string) bool {
	return indexOf(b, []byte(s)) >= 0
}

// containsExactPermutedWeights tolerates any ordering of keys in weights object (since sorted order ensures one deterministic mapping).
func containsExactPermutedWeights(b []byte, w map[string]int) bool {
	if len(w) != 2 {
		return false
	}
	var keys []string
	for k := range w {
		keys = append(keys, k)
	}
	a := keys[0]
	bKey := keys[1]
	pat1 := fmt.Sprintf("\"weights\":{\"%s\":\"%d\",\"%s\":\"%d\"}", a, w[a], bKey, w[bKey])
	pat2 := fmt.Sprintf("\"weights\":{\"%s\":\"%d\",\"%s\":\"%d\"}", bKey, w[bKey], a, w[a])
	return containsExact(b, pat1) || containsExact(b, pat2)
}

// indexOf naive byte slice search (avoids pulling strings repeatedly)
func indexOf(haystack, needle []byte) int {
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
