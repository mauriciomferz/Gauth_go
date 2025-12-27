//go:build ignore

package main

// gen_gap_matrix.go
// Generates docs/GAP_MATRIX.auto.md from artifacts/gap_matrix.csv and config/capabilities.json.
// Also performs a drift detection against docs/GAP_MATRIX.md (human curated) and exits non-zero
// if Status or Priority differ for any Requirement present in the CSV.
//
// Usage:
//   go run scripts/gen_gap_matrix.go
//   (Optionally add a Makefile target: gap-matrix)
//
// Exit codes:
//   0 - success, generated file, no drift
//   2 - drift detected (differences printed to stderr)
//   3 - generation error
//
// This keeps Markdown narrative while allowing structured enforcement via CSV.
// Future: integrate with conformance/report.json to enrich evidence linking.

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	csvPath          = "artifacts/gap_matrix.csv"
	capabilitiesPath = "config/capabilities.json"
	markdownPath     = "docs/GAP_MATRIX.md"
	outMarkdownPath  = "docs/GAP_MATRIX.auto.md"
	outJSONPath      = "docs/GAP_MATRIX.auto.json"
)

// gapRow represents one flattened GAP entry from CSV.
// CSV Header: Section,ID,Requirement,Status,Priority,Gap,Evidence
// Section can contain commas? We assume not. Trim spaces.

type gapRow struct {
	Section     string
	ID          string
	Requirement string
	Status      string
	Priority    string
	Gap         string
	Evidence    string
}

type capabilitiesFile struct {
	SchemaVersion int `json:"schema_version"`
	Capabilities  []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"capabilities"`
}

// parsedRequirement captures parsed data from curated Markdown tables.
// We only extract Requirement, Status, Priority to compare.

type parsedRequirement struct {
	Requirement string
	Status      string
	Priority    string
}

var tableLineRe = regexp.MustCompile(`^\|\s*([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|`) // capture first 5 columns

func main() {
	if err := run(); err != nil {
		var exitCode int
		if errors.Is(err, errDrift) {
			exitCode = 2
		} else {
			exitCode = 3
		}
		fmt.Fprintf(os.Stderr, "gen_gap_matrix: %v\n", err)
		os.Exit(exitCode)
	}
}

var errDrift = errors.New("gap matrix drift detected")

func run() error {
	rows, err := readCSV(csvPath)
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}
	caps, err := readCapabilities(capabilitiesPath)
	if err != nil {
		return fmt.Errorf("read capabilities: %w", err)
	}
	curated, err := parseCuratedMarkdown(markdownPath)
	if err != nil {
		return fmt.Errorf("parse curated markdown: %w", err)
	}
	// Compare drift: for each CSV row, if curated has matching Requirement name, compare Status & Priority.
	drift := compareDrift(rows, curated)
	if err := writeAutoMarkdown(rows, caps, drift); err != nil {
		return fmt.Errorf("write auto markdown: %w", err)
	}
	if err := writeAutoJSON(rows, caps, drift); err != nil {
		return fmt.Errorf("write auto json: %w", err)
	}
	if err := writeBadgeSVG(rows); err != nil {
		return fmt.Errorf("write badge svg: %w", err)
	}
	if len(drift) > 0 {
		return fmt.Errorf("%w (%d differences)", errDrift, len(drift))
	}
	return nil
}

func readCSV(p string) ([]gapRow, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cr := csv.NewReader(f)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return nil, err
	}
	if len(header) < 7 {
		return nil, fmt.Errorf("unexpected header length: %d", len(header))
	}
	var rows []gapRow
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) < 7 {
			continue
		}
		rows = append(rows, gapRow{
			Section:     strings.TrimSpace(rec[0]),
			ID:          strings.TrimSpace(rec[1]),
			Requirement: strings.TrimSpace(rec[2]),
			Status:      strings.TrimSpace(rec[3]),
			Priority:    strings.TrimSpace(rec[4]),
			Gap:         strings.TrimSpace(rec[5]),
			Evidence:    strings.TrimSpace(rec[6]),
		})
	}
	return rows, nil
}

func readCapabilities(p string) (capabilitiesFile, error) {
	var cf capabilitiesFile
	b, err := os.ReadFile(p)
	if err != nil {
		return cf, err
	}
	if err := json.Unmarshal(b, &cf); err != nil {
		return cf, err
	}
	return cf, nil
}

func parseCuratedMarkdown(p string) (map[string]parsedRequirement, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	res := make(map[string]parsedRequirement)
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "| Requirement ") || strings.HasPrefix(line, "|-------------") {
			// header separator; skip
			continue
		}
		if strings.HasPrefix(line, "| ") {
			m := tableLineRe.FindStringSubmatch(line)
			if len(m) >= 6 {
				// m[1] is first column (Requirement); m[4] Status; m[5] Priority (with possible whitespace)
				req := strings.TrimSpace(m[1])
				status := strings.TrimSpace(m[4])
				priority := strings.TrimSpace(m[5])
				res[req] = parsedRequirement{Requirement: req, Status: status, Priority: priority}
			}
		}
	}
	return res, s.Err()
}

// compareDrift returns differences found between CSV structured rows and curated markdown tables.
// Keyed on Requirement name for now; future: use IDs embedded inside Markdown.
func compareDrift(rows []gapRow, curated map[string]parsedRequirement) []string {
	var diffs []string
	for _, r := range rows {
		c, ok := curated[r.Requirement]
		if !ok {
			// Requirement absent from curated markdown
			continue
		}
		if normalizeStatus(r.Status) != normalizeStatus(c.Status) || r.Priority != c.Priority {
			diffs = append(diffs, fmt.Sprintf("%s: CSV(Status=%s,Priority=%s) != MD(Status=%s,Priority=%s)", r.Requirement, r.Status, r.Priority, c.Status, c.Priority))
		}
	}
	return diffs
}

func normalizeStatus(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func writeAutoMarkdown(rows []gapRow, caps capabilitiesFile, drift []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# GAuth RFC Gap Matrix (Generated)\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	if len(drift) > 0 {
		fmt.Fprintf(&b, "**Drift Detected (%d items)**:\n", len(drift))
		for _, d := range drift {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		fmt.Fprintf(&b, "\nCI should fail; update docs/GAP_MATRIX.md or artifacts/gap_matrix.csv to reconcile.\n\n")
	}
	fmt.Fprintf(&b, "## Capability Snapshot\n\n")
	fmt.Fprintf(&b, "| Capability ID | Version | Stable |\n")
	fmt.Fprintf(&b, "|---------------|---------|--------|\n")
	for _, c := range caps.Capabilities {
		fmt.Fprintf(&b, "| %s | %s | %t |\n", c.ID, c.Version, c.Stable)
	}
	fmt.Fprintf(&b, "\nSchema Version: %d\n\n", caps.SchemaVersion)
	// Group rows by Section
	sections := make(map[string][]gapRow)
	order := []string{}
	for _, r := range rows {
		if _, ok := sections[r.Section]; !ok {
			order = append(order, r.Section)
		}
		sections[r.Section] = append(sections[r.Section], r)
	}
	for _, sec := range order {
		fmt.Fprintf(&b, "## %s\n\n", sec)
		fmt.Fprintf(&b, "| ID | Requirement | Status | Priority | Gap | Evidence |\n")
		fmt.Fprintf(&b, "|----|-------------|--------|----------|-----|----------|\n")
		for _, r := range sections[sec] {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", r.ID, escMD(r.Requirement), r.Status, r.Priority, escMD(r.Gap), escMD(r.Evidence))
		}
		fmt.Fprintf(&b, "\n")
	}
	return os.WriteFile(outMarkdownPath, []byte(b.String()), 0o644)
}

// writeAutoJSON emits a machine-consumable JSON snapshot including counts useful for badges.
func writeAutoJSON(rows []gapRow, caps capabilitiesFile, drift []string) error {
	type jsonRow struct {
		Section     string `json:"section"`
		ID          string `json:"id"`
		Requirement string `json:"requirement"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		Gap         string `json:"gap"`
		Evidence    string `json:"evidence"`
	}
	counts := map[string]int{"implemented": 0, "partial": 0, "missing": 0, "conceptual": 0}
	outRows := make([]jsonRow, 0, len(rows))
	for _, r := range rows {
		counts[normalizeStatus(r.Status)]++
		outRows = append(outRows, jsonRow{Section: r.Section, ID: r.ID, Requirement: r.Requirement, Status: r.Status, Priority: r.Priority, Gap: r.Gap, Evidence: r.Evidence})
	}
	total := len(rows)
	payload := map[string]any{
		"generated_at":                time.Now().UTC().Format(time.RFC3339),
		"schema_version":              1,
		"counts":                      counts,
		"total":                       total,
		"drift_items":                 drift,
		"capabilities_schema_version": caps.SchemaVersion,
		"entries":                     outRows,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outJSONPath, b, 0o644)
}

// writeBadgeSVG emits a lightweight static SVG badge summarizing Implemented / Total requirements.
// It intentionally avoids external network dependencies (e.g. shields.io) so local and CI generation are deterministic.
func writeBadgeSVG(rows []gapRow) error {
	var implemented int
	for _, r := range rows {
		if normalizeStatus(r.Status) == "implemented" {
			implemented++
		}
	}
	total := len(rows)
	pct := float64(implemented) / float64(total) * 100.0
	// Color thresholds (simple): <40 red, <70 orange, else green.
	color := "#4c1"
	if pct < 40 {
		color = "#e05d44" // red
	} else if pct < 70 {
		color = "#fe7d37" // orange
	}
	label := "gap"
	value := fmt.Sprintf("%d_of_%d", implemented, total)
	// Basic SVG (widths approximated; not pixel-perfect but sufficient). Dynamic width scaling by char count.
	labelTextWidth := 30 + len(label)*6
	valueTextWidth := 40 + len(value)*6
	totalWidth := labelTextWidth + valueTextWidth
	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
  <linearGradient id="grad" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>
  <mask id="round"><rect width="%d" height="20" rx="3" fill="#fff"/></mask>
  <g mask="url(#round)">
	<rect width="%d" height="20" fill="#555"/>
	<rect x="%d" width="%d" height="20" fill="%s"/>
	<rect width="%d" height="20" fill="url(#grad)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
	<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
	<text x="%d" y="15">%s</text>
	<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
	<text x="%d" y="15">%s</text>
  </g>
</svg>
`, totalWidth, label, value, totalWidth, labelTextWidth, labelTextWidth, valueTextWidth, color, totalWidth, labelTextWidth/2, label, labelTextWidth/2, label, labelTextWidth+valueTextWidth/2, value, labelTextWidth+valueTextWidth/2, value)
	badgeDir := filepath.Join("docs", "badges")
	// #nosec G301
	if err := os.MkdirAll(badgeDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(badgeDir, "gap-matrix.svg"), []byte(svg), 0o644)
}

func escMD(s string) string {
	// Basic escaping for pipe characters inside cells.
	return strings.ReplaceAll(s, "|", "\\|")
}

// ensureRelativePaths ensures script is executed from repo root (so relative paths resolve).
func ensureRelativePaths() error {
	for _, p := range []string{csvPath, capabilitiesPath, markdownPath} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("required path not found: %s (run from repo root)", p)
		}
	}
	return nil
}

func init() {
	// Validate we are in repo root early; ignore error if running tests that stub paths.
	_ = ensureRelativePaths()
}

// Optional: if we later embed IDs in curated markdown, we can extend parser.
// For now we rely on Requirement names as keys for drift detection.

// NOTE: Keep this tool small; for complex transformations consider moving into internal tooling package.

// Future Enhancements:
// - Parse conformance/report.json gap_matrix to enrich Evidence.
// - Add JSON output (docs/GAP_MATRIX.auto.json) for machine processing.
// - Add badge generation (Implemented/Partial/Missing counts) integrated with gen_coverage_badges.go.
// - Integrate fuzz/property status extraction for dimension metrics.
