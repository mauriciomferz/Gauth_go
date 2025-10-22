// Package harnesslib includes history tracking for conformance runs enabling trend dashboards.
package harnesslib

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// HistoryEntry represents one run record.
type HistoryEntry struct {
	Timestamp      time.Time
	Coverage       float64
	GapMissing     int
	GapPartial     int
	GapImplemented int
	MissingSymbols int
	MissingTests   int
}

// LoadHistory parses history CSV previously appended by CLI.
func LoadHistory(path string) ([]HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	var entries []HistoryEntry
	headerProcessed := false
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !headerProcessed {
			headerProcessed = true
			continue
		}
		if len(rec) < 7 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, rec[0])
		if err != nil {
			continue // skip malformed row
		}
		cov, err := strconv.ParseFloat(rec[1], 64)
		if err != nil {
			continue
		}
		gm, err := strconv.Atoi(rec[2])
		if err != nil {
			continue
		}
		gp, err := strconv.Atoi(rec[3])
		if err != nil {
			continue
		}
		gi, err := strconv.Atoi(rec[4])
		if err != nil {
			continue
		}
		ms, err := strconv.Atoi(rec[5])
		if err != nil {
			continue
		}
		mt, err := strconv.Atoi(rec[6])
		if err != nil {
			continue
		}
		entries = append(entries, HistoryEntry{Timestamp: ts, Coverage: cov, GapMissing: gm, GapPartial: gp, GapImplemented: gi, MissingSymbols: ms, MissingTests: mt})
	}
	return entries, nil
}

// TrendMetrics summarizes recent history.
type TrendMetrics struct {
	TotalRuns            int
	CoverageLatest       float64
	CoverageMovingAvg    float64
	CoverageDelta        float64 // latest - previous
	GapMissingLatest     int
	GapMissingDelta      int
	GapImplementedLatest int
	GapImplementedDelta  int
}

// ComputeTrend computes metrics given entries (chronological order assumed as appended). If not chronological, caller should sort.
func ComputeTrend(entries []HistoryEntry, window int) TrendMetrics {
	tm := TrendMetrics{TotalRuns: len(entries)}
	if len(entries) == 0 {
		return tm
	}
	latest := entries[len(entries)-1]
	tm.CoverageLatest = latest.Coverage
	tm.GapMissingLatest = latest.GapMissing
	tm.GapImplementedLatest = latest.GapImplemented
	// Moving average over last window (or all if fewer)
	start := 0
	if window > 0 && len(entries) > window {
		start = len(entries) - window
	}
	var sum float64
	for i := start; i < len(entries); i++ {
		sum += entries[i].Coverage
	}
	tm.CoverageMovingAvg = sum / float64(len(entries)-start)
	if len(entries) >= 2 {
		prev := entries[len(entries)-2]
		tm.CoverageDelta = latest.Coverage - prev.Coverage
		tm.GapMissingDelta = latest.GapMissing - prev.GapMissing
		tm.GapImplementedDelta = latest.GapImplemented - prev.GapImplemented
	}
	return tm
}

// RenderTrendMarkdown produces a small dashboard markdown.
func RenderTrendMarkdown(entries []HistoryEntry, tm TrendMetrics) string {
	var b strings.Builder
	b.WriteString("# Conformance Trend Dashboard\n\n")
	b.WriteString("## Summary\n")
	b.WriteString(fmt.Sprintf("Total Runs: %d\n\n", tm.TotalRuns))
	b.WriteString(fmt.Sprintf("Coverage Latest: %.2f%%\n", tm.CoverageLatest))
	b.WriteString(fmt.Sprintf("Coverage Moving Avg: %.2f%%\n", tm.CoverageMovingAvg))
	b.WriteString(fmt.Sprintf("Coverage Δ (latest-prev): %.2f%%\n\n", tm.CoverageDelta))
	b.WriteString(fmt.Sprintf("Gap Missing Latest: %d (Δ %d)\n", tm.GapMissingLatest, tm.GapMissingDelta))
	b.WriteString(fmt.Sprintf("Gap Implemented Latest: %d (Δ %d)\n\n", tm.GapImplementedLatest, tm.GapImplementedDelta))
	b.WriteString("## Recent Runs\n")
	b.WriteString("| Timestamp | Coverage | GapMissing | GapPartial | GapImplemented | MissingSymbols | MissingTests |\n")
	b.WriteString("|-----------|----------|-----------|-----------|---------------|---------------|-------------|\n")
	// Show last 20
	start := 0
	if len(entries) > 20 {
		start = len(entries) - 20
	}
	for i := start; i < len(entries); i++ {
		e := entries[i]
		b.WriteString(fmt.Sprintf("| %s | %.2f | %d | %d | %d | %d | %d |\n", e.Timestamp.Format(time.RFC3339), e.Coverage, e.GapMissing, e.GapPartial, e.GapImplemented, e.MissingSymbols, e.MissingTests))
	}
	return b.String()
}
