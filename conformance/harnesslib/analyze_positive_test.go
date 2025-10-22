package harnesslib

import (
	"os"
	"testing"
)

func TestAnalyzeFindsCanonicalPOADigest(t *testing.T) {
	// Ensure clause map includes canonical serialization entry referencing CanonicalPOADigest.
	// Clause prefix might not match scanned placeholder; use direct function symbol only.
	cm := `{"entries":[{"clause_prefix":"0115:9.-canonical-serialization","symbols":["CanonicalPOADigest"],"tests_glob":"**/canonical_test.go"}]}`
	if err := os.WriteFile("conformance/clause_map.json", []byte(cm), 0o644); err != nil {
		t.Fatalf("write clause_map: %v", err)
	}
	clauses := []Clause{{ID: "0115:9.-canonical-serialization", Title: "9. Canonical Serialization", RFC: "0115"}}
	ar := Analyze(clauses)
	// Expect symbol found OR tests missing; but not symbol missing.
	for _, f := range ar.Failures {
		if f == "symbol missing: 0115:9.-canonical-serialization -> CanonicalPOADigest" {
			t.Fatalf("expected CanonicalPOADigest to be found; failures=%+v", ar.Failures)
		}
	}
}
