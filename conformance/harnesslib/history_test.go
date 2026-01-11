package harnesslib

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadHistoryAndTrend(t *testing.T) {
	sampleHistoryCSV := "timestamp,coverage,gap_missing,gap_partial,gap_implemented,missing_symbols,missing_tests\n" +
		time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339) + ",75.00,10,5,3,7,2\n" +
		time.Now().Add(-1*time.Hour).UTC().Format(time.RFC3339) + ",80.00,9,5,4,6,2\n" +
		time.Now().UTC().Format(time.RFC3339) + ",82.50,8,5,5,5,2\n"
	f, err := os.CreateTemp(t.TempDir(), "history*.csv")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err2 := f.WriteString(sampleHistoryCSV); err2 != nil {
		t.Fatalf("write: %v", err2)
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}
	entries, err := LoadHistory(f.Name())
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries got %d", len(entries))
	}
	tm := ComputeTrend(entries, 0)
	if tm.TotalRuns != 3 {
		t.Fatalf("TotalRuns mismatch")
	}
	if tm.CoverageLatest < 82.0 {
		t.Fatalf("unexpected latest coverage %.2f", tm.CoverageLatest)
	}
	if tm.CoverageDelta <= 0 {
		t.Fatalf("expected positive coverage delta got %.2f", tm.CoverageDelta)
	}
	md := RenderTrendMarkdown(entries, tm)
	if len(md) == 0 || !containsAll(md, []string{"Conformance Trend Dashboard", "Coverage Latest", "Recent Runs"}) {
		t.Fatalf("markdown output missing expected sections")
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
