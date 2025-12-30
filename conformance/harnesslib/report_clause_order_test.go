package harnesslib

import (
	"strings"
	"testing"
)

func TestClauseOrderingPlacesPlaceholderFirst(t *testing.T) {
	clauses := []Clause{
		{ID: "0115:2.-scope-semantics", Title: "2. Scope Semantics", RFC: "0115"},
		{ID: "AAP-0115-(placeholder-extract)", Title: "AAP002 (Placeholder Extract)", RFC: "0115"},
		{ID: "0115:10.-revocation-semantics", Title: "10. Revocation Semantics", RFC: "0115"},
		{ID: "0115:3.-validity-period", Title: "3. Validity Period", RFC: "0115"},
	}
	rep := Report{GeneratedAt: "2025-01-01T00:00:00Z", Clauses: clauses, Summary: Summary{}}
	md := rep.ToMarkdown()
	// Find first clause row after header
	// Ensure placeholder appears before others
	placeIdx := strings.Index(md, "| AAP-0115-(placeholder-extract) |")
	scopeIdx := strings.Index(md, "| 0115:2.-scope-semantics |")
	if placeIdx == -1 || scopeIdx == -1 {
		t.Fatalf("expected both placeholder and scope clauses present")
	}
	if placeIdx > scopeIdx {
		t.Fatalf("expected placeholder extract before scope; got placeIdx=%d scopeIdx=%d", placeIdx, scopeIdx)
	}
	// Numeric ordering: 2 before 3 before 10
	validIdx := strings.Index(md, "| 0115:3.-validity-period |")
	revIdx := strings.Index(md, "| 0115:10.-revocation-semantics |")
	if !(scopeIdx < validIdx && validIdx < revIdx) {
		t.Fatalf("numeric ordering incorrect: scopeIdx=%d validIdx=%d revIdx=%d", scopeIdx, validIdx, revIdx)
	}
}
