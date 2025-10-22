// Package main provides the primary conformance harness runner. It scans RFC markdown
// clauses, computes coverage/gap metrics, enforces configurable thresholds, and can
// emit markdown/JSON/CSV artifacts plus historical trend dashboards. This entry point
// is the canonical tool used in CI to guard minimum conformance baselines.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/conformance/harnesslib"
)

func main() {
	var csvOut string
	var minCov float64
	var maxGapMissing int
	var maxMissingSymbols int
	var maxMissingTests int
	var historyFile string
	var markdownOut string
	var jsonOut string
	var trendOut string
	var trendWindow int
	var rfcFilesCSV string
	var autoScan bool
	var symbolLocOut string
	flag.StringVar(&csvOut, "csv-out", "", "directory to write CSV exports (gap_matrix.csv, symbol_evidence.csv)")
	flag.Float64Var(&minCov, "min-symbol-coverage", 0, "minimum required symbol coverage percent")
	flag.IntVar(&maxGapMissing, "max-gap-missing", -1, "max allowed missing GAP items (-1 disable)")
	flag.IntVar(&maxMissingSymbols, "max-missing-symbols", -1, "max allowed missing required symbols (-1 disable)")
	flag.IntVar(&maxMissingTests, "max-missing-tests", -1, "max allowed missing test globs (-1 disable)")
	flag.StringVar(&historyFile, "history-file", "", "CSV file path to append run history")
	flag.StringVar(&markdownOut, "markdown-out", "", "path to write markdown report (optional)")
	flag.StringVar(&jsonOut, "json-out", "", "path to write JSON report (optional)")
	flag.StringVar(&trendOut, "trend-markdown-out", "", "path to write trend dashboard markdown (requires --history-file)")
	flag.IntVar(&trendWindow, "trend-window", 10, "window size for moving coverage average (default 10, <=0 means all runs)")
	flag.StringVar(&rfcFilesCSV, "rfc-files", "",
		"comma separated list of RFC markdown files to scan (assign prefix from filename like rfc0115.md -> 0115)")
	flag.BoolVar(&autoScan, "auto-scan-rfcs", true,
		"automatically scan known RFC markdown under docs/rfc if no --rfc-files provided")
	flag.StringVar(&symbolLocOut, "symbol-locations-out", "", "path to write condensed symbol locations markdown (optional)")
	flag.Parse()

	clauses := []harnesslib.Clause{}
	// RFC scanning logic
	var rfcPaths []string
	if rfcFilesCSV != "" {
		for _, p := range splitComma(rfcFilesCSV) {
			if p != "" {
				rfcPaths = append(rfcPaths, p)
			}
		}
	} else if autoScan {
		// Attempt to discover RFC markdown files
		candidates := []string{"docs/rfc/rfc0111.md", "docs/rfc/rfc0115.md"}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				rfcPaths = append(rfcPaths, c)
			}
		}
	}
	for _, p := range rfcPaths {
		prefix := deriveRFCPrefix(p)
		if prefix == "" {
			continue
		}
		cls, err := harnesslib.ScanFile(p, prefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error %s: %v\n", p, err)
			continue
		}
		clauses = append(clauses, cls...)
	}
	ar := harnesslib.Analyze(clauses)
	rep := harnesslib.BuildReport(ar)

	// Threshold gating
	opts := harnesslib.Options{MinSymbolCoverage: minCov, MaxGapMissing: maxGapMissing, MaxMissingSymbols: maxMissingSymbols, MaxMissingTests: maxMissingTests}
	thFailures := harnesslib.CheckThresholds(rep.Summary, opts)
	if len(thFailures) > 0 {
		for _, tf := range thFailures {
			fmt.Fprintf(os.Stderr, "THRESHOLD VIOLATION %s: %s\n", tf.Kind, tf.Message)
		}
	}

	// Write report artifacts
	if markdownOut != "" {
		md := rep.ToMarkdown()
		fmt.Fprintf(os.Stderr, "[conformance-debug] writing markdown to %s bytes=%d generated_at=%s mapped=%d/%d symbols=%d/%d\n", markdownOut, len(md), rep.GeneratedAt, rep.Summary.MappedClausesFound, rep.Summary.MappedClauses, rep.Summary.SymbolsFound, rep.Summary.RequiredSymbols)
		if err := os.WriteFile(markdownOut, []byte(md), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "[conformance-debug] write error: %v\n", err)
		}
	}
	if jsonOut != "" {
		jb, jErr := rep.ToJSON()
		if jErr != nil {
			fmt.Fprintf(os.Stderr, "[conformance-debug] json encode error: %v\n", jErr)
		} else if wErr := os.WriteFile(jsonOut, jb, 0o644); wErr != nil {
			fmt.Fprintf(os.Stderr, "[conformance-debug] json write error: %v\n", wErr)
		}
	}
	if symbolLocOut != "" {
		md := rep.ToSymbolLocationsMarkdown()
		if err := os.WriteFile(symbolLocOut, []byte(md), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "symbol locations write error: %v\n", err)
		}
	}

	// CSV exports
	if csvOut != "" {
		if err := os.MkdirAll(csvOut, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "csv out mkdir: %v\n", err)
		}
		if err := harnesslib.WriteGapCSV(filepath.Join(csvOut, "gap_matrix.csv"), rep); err != nil {
			fmt.Fprintf(os.Stderr, "gap csv write error: %v\n", err)
		}
		if err := harnesslib.WriteSymbolEvidenceCSV(filepath.Join(csvOut, "symbol_evidence.csv"), rep); err != nil {
			fmt.Fprintf(os.Stderr, "symbol evidence csv write error: %v\n", err)
		}
	}

	// History append: columns timestamp, coverage, gap_missing, gap_partial, gap_implemented, missing_symbols, missing_tests
	if historyFile != "" {
		appendHistory(historyFile, rep)
		// Trend dashboard generation if requested or if csvOut provided (auto default)
		// Determine output path: explicit flag overrides, else history_trend.md in csvOut (if set), else none.
		outPath := trendOut
		if outPath == "" && csvOut != "" {
			outPath = filepath.Join(csvOut, "history_trend.md")
		}
		if outPath != "" {
			entries, err := harnesslib.LoadHistory(historyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "trend load history error: %v\n", err)
			} else {
				tm := harnesslib.ComputeTrend(entries, trendWindow)
				md := harnesslib.RenderTrendMarkdown(entries, tm)
				if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "trend write error: %v\n", err)
				}
			}
		}
	}

	if len(thFailures) > 0 {
		os.Exit(2)
	}
}

func appendHistory(path string, rep harnesslib.Report) {
	newFile := false
	if _, err := os.Stat(path); err != nil {
		newFile = true
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "history open: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	if newFile {
		if err := w.Write([]string{"timestamp", "coverage", "gap_missing", "gap_partial", "gap_implemented", "missing_symbols", "missing_tests"}); err != nil {
			fmt.Fprintf(os.Stderr, "history header write: %v\n", err)
		}
	}
	s := rep.Summary
	if err := w.Write([]string{time.Now().UTC().Format(time.RFC3339), fmt.Sprintf("%.2f", s.CoveragePercent), fmt.Sprintf("%d", s.GapMissing), fmt.Sprintf("%d", s.GapPartial), fmt.Sprintf("%d", s.GapImplemented), fmt.Sprintf("%d", s.MissingSymbols), fmt.Sprintf("%d", s.MissingTests)}); err != nil {
		fmt.Fprintf(os.Stderr, "history row write: %v\n", err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "history flush error: %v\n", err)
	}
}

// splitComma splits a comma separated list trimming spaces
func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// deriveRFCPrefix extracts numeric part from filename like rfc0115.md -> 0115
func deriveRFCPrefix(path string) string {
	base := filepath.Base(path)
	// find sequence of digits of length >=3 preceding .md
	for i := 0; i < len(base); i++ {
		if base[i] == 'r' && i+3 < len(base) && strings.HasPrefix(base[i:], "rfc") {
			// collect digits after 'rfc'
			digits := ""
			for j := i + 3; j < len(base) && base[j] >= '0' && base[j] <= '9'; j++ {
				digits += string(base[j])
			}
			if len(digits) >= 3 {
				return digits
			}
		}
	}
	return ""
}

// (end of file)
