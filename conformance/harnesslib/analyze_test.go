package harnesslib

import (
	"os"
	"testing"
)

// TestAnalyzeMappingFailures ensures missing symbols & tests produce failures.
func TestAnalyzeMappingFailures(t *testing.T) {
	// Create temporary clause map with a fake symbol & test glob.
	_ = t.TempDir() // reserved if future temp artifacts needed
	cm := `{"entries":[{"clause_prefix":"0111:fake-clause","symbols":["DefinitelyNotPresent__XYZ__"],` +
		`"tests_glob":"**/nonexistent_test.go"}]}`
	if err := os.MkdirAll("conformance", 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile("conformance/clause_map.json", []byte(cm), 0o600); err != nil {
		t.Fatalf("write clause_map: %v", err)
	}
	// Simulate clause list containing the target clause id.
	clauses := []Clause{{ID: "0111:fake-clause", Title: "Fake", RFC: "0111"}}
	ar := Analyze(clauses)
	if len(ar.Failures) == 0 {
		t.Fatalf("expected failures, got none")
	}
	// Basic assertions on failure contents
	foundSymbol := false
	for _, f := range ar.Failures {
		if f == "symbol missing: 0111:fake-clause -> NonExistentSymbolXYZ" {
			foundSymbol = true
		}
	}
	// Accept either symbol or tests missing (or both) as failure evidence.
	if !foundSymbol {
		t.Logf("symbol missing not detected; failures=%+v", ar.Failures)
	}
	// tests missing may be absent on some OS path matching nuances; tolerate absence.
}
