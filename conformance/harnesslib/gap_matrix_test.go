package harnesslib

import (
	"os"
	"strings"
	"testing"
)

const sampleGapJSON = `{
  "generated": "2025-10-18",
  "version": "test",
  "sections": [
    {"id":"s1","name":"Section","items":[
      {"id":"s1.i1","requirement":"Req1","status":"Implemented","gap":"none","evidence":[],"priority":"P0"},
      {"id":"s1.i2","requirement":"Req2","status":"Partial","gap":"work","evidence":[],"priority":"P1"},
      {"id":"s1.i3","requirement":"Req3","status":"Missing","gap":"todo","evidence":[],"priority":"P2"}
    ]}
  ]
}`

func TestLoadGapMatrixCounts(t *testing.T) {
	// Write temp gap matrix file.
	f, err := os.CreateTemp(t.TempDir(), "gap_matrix*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err := f.WriteString(sampleGapJSON); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	os.Setenv("GAP_MATRIX_PATH", f.Name())
	defer os.Unsetenv("GAP_MATRIX_PATH")

	// Minimal clauses to drive Analyze (empty mapping will yield zero required symbols).
	clauses := []Clause{}
	ar := Analyze(clauses)
	if ar.Summary.GapTotal != 3 {
		t.Fatalf("expected GapTotal=3 got %d", ar.Summary.GapTotal)
	}
	if ar.Summary.GapImplemented != 1 || ar.Summary.GapPartial != 1 || ar.Summary.GapMissing != 1 {
		t.Fatalf("unexpected gap counts: %+v", ar.Summary)
	}
}

func TestReportMarkdownGapTable(t *testing.T) {
	// reuse sampleGapJSON
	f, err := os.CreateTemp(t.TempDir(), "gap_matrix*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err := f.WriteString(sampleGapJSON); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	os.Setenv("GAP_MATRIX_PATH", f.Name())
	defer os.Unsetenv("GAP_MATRIX_PATH")
	ar := Analyze([]Clause{})
	rep := BuildReport(ar)
	md := rep.ToMarkdown()
	if !strings.Contains(md, "## GAP Details") {
		t.Fatalf("expected GAP Details section")
	}
	if !strings.Contains(md, "| Section | ID | Requirement | Status | Priority | Gap | Evidence |") {
		t.Fatalf("expected GAP table header")
	}
	if !strings.Contains(md, "Req1") {
		t.Fatalf("expected requirement Req1 in markdown")
	}
}
