package harnesslib

import "testing"

func TestCheckThresholdsPassing(t *testing.T) {
	s := Summary{CoveragePercent: 85.0, GapMissing: 2, MissingSymbols: 1, MissingTests: 0}
	opts := Options{MinSymbolCoverage: 80, MaxGapMissing: 5, MaxMissingSymbols: 3, MaxMissingTests: 2}
	fails := CheckThresholds(s, opts)
	if len(fails) != 0 {
		t.Fatalf("expected 0 failures got %d", len(fails))
	}
}

func TestCheckThresholdsFailing(t *testing.T) {
	s := Summary{CoveragePercent: 60.0, GapMissing: 10, MissingSymbols: 7, MissingTests: 4}
	opts := Options{MinSymbolCoverage: 80, MaxGapMissing: 5, MaxMissingSymbols: 3, MaxMissingTests: 2}
	fails := CheckThresholds(s, opts)
	if len(fails) != 4 {
		t.Fatalf("expected 4 failures got %d", len(fails))
	}
	kinds := map[string]bool{}
	for _, f := range fails {
		kinds[f.Kind] = true
	}
	for _, k := range []string{"symbol_coverage", "gap_missing", "missing_symbols", "missing_tests"} {
		if !kinds[k] {
			t.Fatalf("missing failure kind %s", k)
		}
	}
}
