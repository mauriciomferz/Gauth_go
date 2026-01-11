// Package harnesslib provides report export utilities (CSV generation for GAP items and symbol
// evidence) complementing the analysis functions.
package harnesslib

import (
	"encoding/csv"
	"os"
	"sort"
	"strings"
)

// GapItemsCSV returns a CSV string of all gap items: Section,ID,Requirement,Status,Priority,Gap,Evidence.
// Evidence contains pipe-separated evidence locations.
func (r Report) GapItemsCSV() string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	if err := w.Write([]string{"Section", "ID", "Requirement", "Status", "Priority", "Gap", "Evidence"}); err != nil {
		return "" // return empty on write failure
	}
	for _, sec := range r.GapMatrix.Sections {
		for _, it := range sec.Items {
			ev := strings.Join(it.Evidence, "|")
			if err := w.Write([]string{sec.Name, it.ID, it.Requirement, it.Status, it.Priority, it.Gap, ev}); err != nil {
				return "" // abort on first write error
			}
		}
	}
	w.Flush()
	return sb.String()
}

// SymbolEvidenceCSV returns CSV with columns: Symbol,Locations (pipe-separated)
func (r Report) SymbolEvidenceCSV() string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	if err := w.Write([]string{"Symbol", "Locations"}); err != nil {
		return ""
	}
	keys := make([]string, 0, len(r.Evidence))
	for k := range r.Evidence {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		locs := append([]string{}, r.Evidence[k]...)
		sort.Strings(locs)
		if err := w.Write([]string{k, strings.Join(locs, "|")}); err != nil {
			return ""
		}
	}
	w.Flush()
	return sb.String()
}

// WriteGapCSV writes the gap items CSV to the given path.
func WriteGapCSV(path string, r Report) error {
	return os.WriteFile(path, []byte(r.GapItemsCSV()), 0o600)
}

// WriteSymbolEvidenceCSV writes the symbol evidence CSV to the given path.
func WriteSymbolEvidenceCSV(path string, r Report) error {
	return os.WriteFile(path, []byte(r.SymbolEvidenceCSV()), 0o600)
}
