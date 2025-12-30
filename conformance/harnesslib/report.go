package harnesslib

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func (r Report) ToJSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r Report) ToMarkdown() string {
	var b strings.Builder
	// Safety metadata header comment for cache detection / tooling hooks.
	fmt.Fprintf(&b, "<!-- conformance-meta generated=%s mapped_clauses=%d found_clauses=%d required_symbols=%d symbols_found=%d coverage=%.2f gap_impl=%d gap_partial=%d gap_missing=%d gap_total=%d -->\n", r.GeneratedAt, r.Summary.MappedClauses, r.Summary.MappedClausesFound, r.Summary.RequiredSymbols, r.Summary.SymbolsFound, r.Summary.CoveragePercent, r.Summary.GapImplemented, r.Summary.GapPartial, r.Summary.GapMissing, r.Summary.GapTotal)
	b.WriteString("# Conformance Report\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", r.GeneratedAt)

	// Summary section
	totalRequired := r.Summary.RequiredSymbols
	coveragePct := r.Summary.CoveragePercent
	b.WriteString("## Summary\n")
	fmt.Fprintf(&b, "Mapped Clauses: %d / %d\n\n", r.Summary.MappedClausesFound, r.Summary.MappedClauses)
	fmt.Fprintf(&b, "Symbols: %d / %d (%.2f%% coverage)\n\n", r.Summary.SymbolsFound, totalRequired, coveragePct)
	fmt.Fprintf(&b, "Test Globs: %d present of %d required\n\n", r.Summary.ClausesTestsFound, r.Summary.ClausesWithTests)
	fmt.Fprintf(&b, "Missing: clauses=%d symbols=%d tests=%d\n\n", r.Summary.MissingClauses, r.Summary.MissingSymbols, r.Summary.MissingTests)
	fmt.Fprintf(&b, "GAP Matrix: implemented=%d partial=%d missing=%d total=%d\n\n", r.Summary.GapImplemented, r.Summary.GapPartial, r.Summary.GapMissing, r.Summary.GapTotal)

	// Clauses section
	b.WriteString("## Clauses\n\n")
	if len(r.Clauses) == 0 {
		b.WriteString("_No clauses scanned._\n\n")
	} else {
		b.WriteString("| Clause ID | Title | RFC  |\n")
		b.WriteString("| ----------------------------------- | ------------------------------ | ---- |\n")
		// Custom ordering: placeholder extracts first per RFC, then numeric sections ascending.
		sort.Slice(r.Clauses, func(i, j int) bool {
			ci, cj := r.Clauses[i], r.Clauses[j]
			isPlaceI := strings.Contains(ci.ID, "placeholder-extract")
			isPlaceJ := strings.Contains(cj.ID, "placeholder-extract")
			if isPlaceI != isPlaceJ {
				return isPlaceI && !isPlaceJ
			}
			// Extract leading numeric section after RFC prefix pattern '/aap>:<n>.'
			secNum := func(id string) int {
				// id like 0111:4.-audit-logging or 0115:10.-revocation-semantics
				parts := strings.Split(id, ":")
				if len(parts) < 2 {
					return 0
				}
				// section starts at parts[1]; take up to first '.'
				rest := parts[1]
				dot := strings.Index(rest, ".")
				if dot == -1 {
					return 0
				}
				numStr := rest[:dot]
				// handle multi-digit numbers
				n := 0
				for _, r := range numStr {
					if r < '0' || r > '9' {
						return 0
					}
					n = n*10 + int(r-'0')
				}
				return n
			}
			si, sj := secNum(ci.ID), secNum(cj.ID)
			if si != sj {
				return si < sj
			}
			// Fallback: lexicographic ID
			return ci.ID < cj.ID
		})
		for _, c := range r.Clauses {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", c.ID, c.Title, c.RFC)
		}
		b.WriteString("\n")
	}

	if len(r.Failures) == 0 {
		b.WriteString("_No failures detected._\n\n")
	} else {
		b.WriteString("### Failures\n\n")
		for _, f := range r.Failures {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		b.WriteString("\n")
	}

	// Evidence table (deduplicated locations)
	b.WriteString("## Evidence\n\n")
	if len(r.Evidence) == 0 {
		b.WriteString("_No evidence collected._\n")
	} else {
		b.WriteString("| Symbol | Locations |\n")
		b.WriteString("| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |\n")
		// Deterministic symbol ordering
		syms := make([]string, 0, len(r.Evidence))
		for s := range r.Evidence {
			syms = append(syms, s)
		}
		sort.Strings(syms)
		for _, s := range syms {
			locs := r.Evidence[s]
			// Deduplicate identical locations while preserving original order of first occurrence.
			seen := make(map[string]struct{}, len(locs))
			dedup := make([]string, 0, len(locs))
			for _, l := range locs {
				if _, ok := seen[l]; ok {
					continue
				}
				seen[l] = struct{}{}
				dedup = append(dedup, l)
			}
			joined := strings.Join(dedup, "<br>")
			fmt.Fprintf(&b, "| %s | %s |\n", s, joined)
		}
		b.WriteString("\n")
	}

	// GAP Details Section
	if len(r.GapMatrix.Sections) > 0 {
		b.WriteString("## GAP Details\n\n")
		fmt.Fprintf(&b, "Source Generated: %s\n\n", r.GapMatrix.Generated)
		b.WriteString("| Section | ID | Requirement | Status | Priority | Gap | Evidence |\n")
		b.WriteString("| -------------------------------------- | ----------- | ----------------------------------------- | ----------- | -------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |\n")
		// Flatten items
		type row struct {
			Section, ID, Req, Status, Priority, Gap string
			Evidence                                []string
		}
		var rows []row
		for _, gs := range r.GapMatrix.Sections {
			for _, it := range gs.Items {
				rows = append(rows, row{Section: gs.Name, ID: it.ID, Req: it.Requirement, Status: it.Status, Priority: it.Priority, Gap: it.Gap, Evidence: it.Evidence})
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		for _, rw := range rows {
			ev := ""
			if len(rw.Evidence) > 0 {
				ev = strings.Join(rw.Evidence, "<br>")
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", rw.Section, rw.ID, rw.Req, rw.Status, rw.Priority, rw.Gap, ev)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "_GAP status distribution: implemented=%d partial=%d missing=%d total=%d_\n\n", r.Summary.GapImplemented, r.Summary.GapPartial, r.Summary.GapMissing, r.Summary.GapTotal)
	}

	return b.String()
}

// ToSymbolLocationsMarkdown renders a condensed symbol → locations mapping markdown file.
// Each symbol lists count and collapsible details (simple text fallback; consumers can post-process for actual HTML collapsible if desired).
func (r Report) ToSymbolLocationsMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- symbol-locations generated=%s total_symbols=%d -->\n", r.GeneratedAt, len(r.Evidence))
	b.WriteString("# Conformance Symbol Locations\n\n")
	b.WriteString("| Symbol | Location Count | Locations |\n")
	b.WriteString("|--------|----------------|----------|\n")
	syms := make([]string, 0, len(r.Evidence))
	for s := range r.Evidence {
		syms = append(syms, s)
	}
	sort.Strings(syms)
	for _, s := range syms {
		locs := r.Evidence[s]
		// Deduplicate same as main markdown
		seen := map[string]struct{}{}
		dedup := make([]string, 0, len(locs))
		for _, l := range locs {
			if _, ok := seen[l]; ok {
				continue
			}
			seen[l] = struct{}{}
			dedup = append(dedup, l)
		}
		joined := strings.Join(dedup, "<br>")
		fmt.Fprintf(&b, "| %s | %d | %s |\n", s, len(dedup), joined)
	}
	return b.String()
}
