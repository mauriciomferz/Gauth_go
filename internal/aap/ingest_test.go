package aap

import (
	"os"
	"testing"
)

const sampleRFC = `# Sample RFC

## 1. First Section
Implementations MUST provide a way to do X.
This line is informational.
Delegates SHOULD verify Y before proceeding.

## 2. Second Section
Tokens MAY be cached if integrity is preserved.
Systems MUST NOT reuse nonces.
`

func TestParseRFCFile_ExtractsClausesAndNormativeStatements(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "aap.md")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err2 := tmp.WriteString(sampleRFC); err2 != nil {
		t.Fatalf("write sample: %v", err)
	}
	if err2 := tmp.Close(); err2 != nil {
		t.Fatalf("close sample: %v", err)
	}

	clauses, err := ParseRFCFile(tmp.Name(), "0111")
	if err != nil {
		t.Fatalf("ParseRFCFile error: %v", err)
	}
	if len(clauses) != 2 { // two sections with ## headings
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
	// First clause normative statements
	first := clauses[0]
	if first.Title == "" {
		t.Errorf("expected title for first clause")
	}
	if len(first.NormativeStatements) != 2 { // MUST + SHOULD
		t.Fatalf("expected 2 normative statements in first clause, got %d", len(first.NormativeStatements))
	}
	levels := map[string]bool{}
	for _, n := range first.NormativeStatements {
		levels[n.Level] = true
	}
	if !levels["MUST"] || !levels["SHOULD"] {
		t.Errorf("missing expected MUST/SHOULD levels: %+v", levels)
	}

	second := clauses[1]
	if len(second.NormativeStatements) != 2 { // MAY + MUST NOT
		t.Fatalf("expected 2 normative statements in second clause, got %d", len(second.NormativeStatements))
	}
	secLevels := map[string]bool{}
	for _, n := range second.NormativeStatements {
		secLevels[n.Level] = true
	}
	if !secLevels["MAY"] || !secLevels["MUST NOT"] {
		t.Errorf("missing expected MAY/MUST NOT levels: %+v", secLevels)
	}
}
