package harnesslib

import "fmt"

// Options defines gating thresholds and output preferences.
type Options struct {
	MinSymbolCoverage float64 // percent (0-100). If >0 enforce Symbols coverage.
	MaxGapMissing     int     // if >=0 enforce maximum allowed missing GAP items.
	MaxMissingSymbols int     // if >=0 enforce missing symbol count cap.
	MaxMissingTests   int     // if >=0 enforce missing tests cap.
}

// ThresholdFailure represents a gating violation.
type ThresholdFailure struct {
	Kind    string
	Message string
}

// CheckThresholds evaluates summary against provided options.
func CheckThresholds(s Summary, o Options) []ThresholdFailure {
	var out []ThresholdFailure
	if o.MinSymbolCoverage > 0 && s.CoveragePercent < o.MinSymbolCoverage {
		out = append(out, ThresholdFailure{
			Kind:    "symbol_coverage",
			Message: fmt.Sprintf("coverage %.2f < %.2f", s.CoveragePercent, o.MinSymbolCoverage),
		})
	}
	if o.MaxGapMissing >= 0 && s.GapMissing > o.MaxGapMissing {
		out = append(out, ThresholdFailure{
			Kind:    "gap_missing",
			Message: fmt.Sprintf("gap missing %d > %d", s.GapMissing, o.MaxGapMissing),
		})
	}
	if o.MaxMissingSymbols >= 0 && s.MissingSymbols > o.MaxMissingSymbols {
		out = append(out, ThresholdFailure{
			Kind:    "missing_symbols",
			Message: fmt.Sprintf("missing symbols %d > %d", s.MissingSymbols, o.MaxMissingSymbols),
		})
	}
	if o.MaxMissingTests >= 0 && s.MissingTests > o.MaxMissingTests {
		out = append(out, ThresholdFailure{
			Kind:    "missing_tests",
			Message: fmt.Sprintf("missing tests %d > %d", s.MissingTests, o.MaxMissingTests),
		})
	}
	return out
}
