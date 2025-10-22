package harnesslib

import (
	"os"
	"strings"
	"testing"
)

func TestGapItemsCSV(t *testing.T) {
	// minimal gap matrix
	sample := `{
      "generated":"2025-10-18","version":"t","sections":[{"id":"s","name":"Sec","items":[{"id":"s.i1","requirement":"R,comma","status":"Implemented","gap":"G|pipe","evidence":["file.go:10"],"priority":"P0"}]}]
    }`
	f, err := os.CreateTemp(t.TempDir(), "gap_matrix*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := f.WriteString(sample); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	os.Setenv("GAP_MATRIX_PATH", f.Name())
	defer os.Unsetenv("GAP_MATRIX_PATH")
	ar := Analyze([]Clause{})
	rep := BuildReport(ar)
	csv := rep.GapItemsCSV()
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header+1), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Section,ID,Requirement,Status,Priority,Gap,Evidence") {
		t.Fatalf("bad header: %s", lines[0])
	}
	// ensure commas quoted appropriately (R,comma should be quoted by csv writer)
	if !strings.Contains(lines[1], "R,comma") {
		t.Fatalf("row missing requirement field: %s", lines[1])
	}
	if !strings.Contains(lines[1], "G|pipe") {
		t.Fatalf("row missing gap field: %s", lines[1])
	}
}

func TestSymbolEvidenceCSV(t *testing.T) {
	// Build artificial report with Evidence
	r := Report{Evidence: map[string][]string{"SymA": {"a.go:1", "b.go:2"}, "SymB": {"c.go:3"}}}
	csv := r.SymbolEvidenceCSV()
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) != 1+len(r.Evidence) {
		t.Fatalf("expected header + %d rows got %d", len(r.Evidence), len(lines))
	}
	if lines[0] != "Symbol,Locations" {
		t.Fatalf("bad header: %s", lines[0])
	}
	// verify ordering sorted (SymA before SymB)
	if !strings.HasPrefix(lines[1], "SymA,") {
		t.Fatalf("expected SymA first, got %s", lines[1])
	}
}
