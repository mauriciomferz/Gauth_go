package main

// gen_gap_matrix.go
// Generates docs/GAP_MATRIX.auto.md from artifacts/gap_matrix.csv and config/capabilities.json.
// Performs drift detection against docs/GAP_MATRIX.md and exits non-zero if discrepancies.
// Usage: go run ./scripts/gapmatrix

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	csvPath           = "artifacts/gap_matrix.csv"
	capabilitiesPath  = "config/capabilities.json"
	markdownPath      = "docs/GAP_MATRIX.md"
	outMarkdownPath   = "docs/GAP_MATRIX.auto.md"
	outJSONPath       = "docs/GAP_MATRIX.auto.json"
	outBadgeJSONPath  = "docs/gap_matrix_badge.json"
	historyPath       = "artifacts/history_gap_matrix.jsonl"
	statusHistoryPath = "artifacts/history_gap_matrix_status.jsonl"
)

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

type parsedRequirement struct {
	ID          string
	Requirement string
	Status      string
	Priority    string
}

// Legacy regex (without an explicit ID column) – kept for backward compatibility when
// curated Markdown omits the ID in human-facing tables.
var tableLineRe = regexp.MustCompile(`^\|\s*([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|`)

var errDrift = errors.New("gap matrix drift detected")

func main() {
	if err := run(); err != nil {
		var exit int
		if errors.Is(err, errDrift) {
			exit = 2
		} else {
			exit = 3
		}
		fmt.Fprintf(os.Stderr, "gen_gap_matrix: %v\n", err)
		os.Exit(exit)
	}
}

func run() error {
	if err := ensurePaths(); err != nil {
		return err
	}
	rows, err := readCSV(csvPath)
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}
	caps, err := readCapabilities(capabilitiesPath)
	if err != nil {
		return fmt.Errorf("read capabilities: %w", err)
	}
	curatedByID, curatedByReq, err := parseCuratedMarkdown(markdownPath)
	if err != nil {
		return fmt.Errorf("parse curated markdown: %w", err)
	}
	drift := compareDrift(rows, curatedByID, curatedByReq)
	counts := computeStatusCounts(rows)
	if err := writeAutoMarkdown(rows, caps, drift, counts); err != nil {
		return fmt.Errorf("write auto markdown: %w", err)
	}
	if err := writeAutoJSON(rows, caps, drift, counts); err != nil {
		return fmt.Errorf("write auto json: %w", err)
	}
	if err := writeBadgeJSON(counts); err != nil {
		return fmt.Errorf("write badge json: %w", err)
	}
	if err := appendHistory(counts); err != nil {
		// History append should not fail the build harshly; emit warning to stderr.
		fmt.Fprintf(os.Stderr, "warn: unable to append history: %v\n", err)
	}
	if err := appendStatusHistory(rows); err != nil {
		fmt.Fprintf(os.Stderr, "warn: unable to append status history: %v\n", err)
	}
	if len(drift) > 0 {
		return fmt.Errorf("%w (%d differences)", errDrift, len(drift))
	}
	return nil
}

func ensurePaths() error {
	for _, p := range []string{csvPath, capabilitiesPath, markdownPath} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("required path missing: %s", p)
		}
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

// parseCuratedMarkdown attempts to extract requirement rows from the human‑maintained
// GAP_MATRIX.md. It supports two layouts:
//
//	(1) Original: | Requirement | Current Implementation | Gap | Status | Priority | ... |
//	(2) ID-first: | ID | Requirement | Current Implementation | Gap | Status | Priority | ... |
//
// It returns two maps: by ID (if present) and by Requirement text.
func parseCuratedMarkdown(p string) (map[string]parsedRequirement, map[string]parsedRequirement, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	byID := make(map[string]parsedRequirement)
	byReq := make(map[string]parsedRequirement)

	var headerIndices struct {
		hasHeader bool
		id        int
		req       int
		status    int
		priority  int
	}

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "|") {
			continue
		}
		// Skip separator lines (|-----| style)
		sepLine := true
		for _, ch := range line {
			if ch != '|' && ch != '-' && ch != ' ' { // presence of any other char means not a pure separator
				sepLine = false
				break
			}
		}
		if sepLine {
			continue
		}

		// Split into trimmed cells (exclude leading/trailing empties from split behavior)
		rawCells := strings.Split(line, "|")
		var cells []string
		for _, c := range rawCells {
			ct := strings.TrimSpace(c)
			if ct == "" {
				continue
			}
			cells = append(cells, ct)
		}
		if len(cells) < 5 { // not a data row of interest
			continue
		}

		// Detect header row
		if !headerIndices.hasHeader && containsAll(cells, []string{"Requirement", "Status", "Priority"}) {
			headerIndices.hasHeader = true
			headerIndices.req = indexOf(cells, "Requirement")
			headerIndices.status = indexOf(cells, "Status")
			headerIndices.priority = indexOf(cells, "Priority")
			headerIndices.id = indexOf(cells, "ID") // -1 if absent
			continue
		}

		if headerIndices.hasHeader {
			// Extract row using header indices.
			if headerIndices.req >= len(cells) || headerIndices.status >= len(cells) || headerIndices.priority >= len(cells) {
				continue
			}
			req := cells[headerIndices.req]
			status := cells[headerIndices.status]
			priority := cells[headerIndices.priority]
			var id string
			if headerIndices.id >= 0 && headerIndices.id < len(cells) {
				id = cells[headerIndices.id]
			}
			pr := parsedRequirement{ID: id, Requirement: req, Status: status, Priority: priority}
			if id != "" {
				byID[id] = pr
			}
			// Always populate byReq for backward compatibility / lookups.
			byReq[req] = pr
		} else {
			// Fallback legacy regex (no header encountered yet). Keeps previous behavior.
			m := tableLineRe.FindStringSubmatch(line)
			if len(m) >= 6 {
				req := strings.TrimSpace(m[1])
				status := strings.TrimSpace(m[4])
				priority := strings.TrimSpace(m[5])
				pr := parsedRequirement{Requirement: req, Status: status, Priority: priority}
				byReq[req] = pr
			}
		}
	}
	return byID, byReq, s.Err()
}

func containsAll(hay []string, needles []string) bool {
	for _, n := range needles {
		if indexOf(hay, n) == -1 {
			return false
		}
	}
	return true
}

func indexOf(sl []string, target string) int {
	for i, v := range sl {
		if strings.EqualFold(strings.TrimSpace(v), target) {
			return i
		}
	}
	return -1
}

func compareDrift(rows []gapRow, curatedByID, curatedByReq map[string]parsedRequirement) []string {
	var diffs []string
	for _, r := range rows {
		var c parsedRequirement
		var ok bool
		if r.ID != "" {
			c, ok = curatedByID[r.ID]
		}
		if !ok { // fallback to requirement text
			c, ok = curatedByReq[r.Requirement]
		}
		if !ok {
			// If curated markdown intentionally omits the row (e.g., consolidated), skip drift check.
			continue
		}
		if normalizeStatus(r.Status) != normalizeStatus(c.Status) || r.Priority != c.Priority {
			key := r.Requirement
			if r.ID != "" {
				key = fmt.Sprintf("%s (%s)", r.ID, r.Requirement)
			}
			diffs = append(diffs, fmt.Sprintf("%s: CSV(Status=%s,Priority=%s) != MD(Status=%s,Priority=%s)", key, r.Status, r.Priority, c.Status, c.Priority))
		}
	}
	return diffs
}

func normalizeStatus(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

type statusCounts struct {
	Implemented int `json:"implemented"`
	Partial     int `json:"partial"`
	Missing     int `json:"missing"`
	Conceptual  int `json:"conceptual"`
	Total       int `json:"total"`
}

func computeStatusCounts(rows []gapRow) statusCounts {
	var c statusCounts
	for _, r := range rows {
		switch normalizeStatus(r.Status) {
		case "implemented":
			c.Implemented++
		case "partial":
			c.Partial++
		case "missing":
			c.Missing++
		case "conceptual":
			c.Conceptual++
		}
	}
	c.Total = len(rows)
	return c
}

func writeAutoMarkdown(rows []gapRow, caps capabilitiesFile, drift []string, counts statusCounts) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# GAuth RFC Gap Matrix (Generated)\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "**Status Summary:** Implemented=%d | Partial=%d | Missing=%d | Conceptual=%d | Total=%d\n\n", counts.Implemented, counts.Partial, counts.Missing, counts.Conceptual, counts.Total)
	if len(drift) > 0 {
		fmt.Fprintf(&b, "**Drift Detected (%d items)**:\n", len(drift))
		for _, d := range drift {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		fmt.Fprintf(&b, "\nCI should fail; reconcile docs/GAP_MATRIX.md and artifacts/gap_matrix.csv.\n\n")
	}
	fmt.Fprintf(&b, "## Capability Snapshot\n\n")
	fmt.Fprintf(&b, "| Capability ID | Version | Stable |\n")
	fmt.Fprintf(&b, "|---------------|---------|--------|\n")
	for _, c := range caps.Capabilities {
		fmt.Fprintf(&b, "| %s | %s | %t |\n", c.ID, c.Version, c.Stable)
	}
	fmt.Fprintf(&b, "\nSchema Version: %d\n\n", caps.SchemaVersion)
	sections := make(map[string][]gapRow)
	var order []string
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
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", r.ID, esc(r.Requirement), r.Status, r.Priority, esc(r.Gap), esc(r.Evidence))
		}
		fmt.Fprintf(&b, "\n")
	}
	return os.WriteFile(outMarkdownPath, []byte(b.String()), 0o644)
}

func esc(s string) string { return strings.ReplaceAll(s, "|", "\\|") }

// writeAutoJSON writes machine-readable JSON including counts & drift info.
func writeAutoJSON(rows []gapRow, caps capabilitiesFile, drift []string, counts statusCounts) error {
	// Enrich evidence for each row.
	type evidenceDetail struct {
		Raw           string   `json:"raw"`
		Files         []string `json:"files"`
		Existing      []string `json:"existing"`
		Missing       []string `json:"missing"`
		TestFiles     []string `json:"test_files"`
		CodeFiles     []string `json:"code_files"`
		ExistingCount int      `json:"existing_count"`
		MissingCount  int      `json:"missing_count"`
		TestFileCount int      `json:"test_file_count"`
		CodeFileCount int      `json:"code_file_count"`
	}
	type enrichedRow struct {
		gapRow
		EvidenceDetail evidenceDetail `json:"evidence_detail"`
	}
	var enriched []enrichedRow
	for _, r := range rows {
		ed := evidenceDetail{Raw: r.Evidence}
		parts := splitEvidence(r.Evidence)
		ed.Files = parts
		for _, p := range parts {
			clean := p
			if i := strings.IndexRune(clean, ':'); i != -1 { // allow file:line style
				clean = clean[:i]
			}
			if clean == "" {
				continue
			}
			if fi, err := os.Stat(clean); err == nil && !fi.IsDir() {
				ed.Existing = append(ed.Existing, clean)
				if strings.HasSuffix(clean, "_test.go") {
					ed.TestFiles = append(ed.TestFiles, clean)
				} else if strings.HasSuffix(clean, ".go") || strings.HasSuffix(clean, ".md") {
					ed.CodeFiles = append(ed.CodeFiles, clean)
				}
			} else {
				ed.Missing = append(ed.Missing, clean)
			}
		}
		ed.ExistingCount = len(ed.Existing)
		ed.MissingCount = len(ed.Missing)
		ed.TestFileCount = len(ed.TestFiles)
		ed.CodeFileCount = len(ed.CodeFiles)
		enriched = append(enriched, enrichedRow{gapRow: r, EvidenceDetail: ed})
	}
	type jsonOut struct {
		Generated    string           `json:"generated"`
		Counts       statusCounts     `json:"counts"`
		Drift        []string         `json:"drift,omitempty"`
		Capabilities capabilitiesFile `json:"capabilities"`
		Rows         []enrichedRow    `json:"rows"`
	}
	out := jsonOut{
		Generated:    time.Now().UTC().Format(time.RFC3339),
		Counts:       counts,
		Drift:        drift,
		Capabilities: caps,
		Rows:         enriched,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outJSONPath, b, 0o644)
}

// writeBadgeJSON emits a compact JSON structure suitable for consumption by
// shields.io dynamic endpoint badges or other dashboards.
// Shape (example):
//
//	{
//	  "schema": 1,
//	  "implemented": 12,
//	  "partial": 8,
//	  "missing": 4,
//	  "conceptual": 1,
//	  "total": 25,
//	  "implemented_pct": 48.0,
//	  "coverage_score": "48% Implemented / 80% w+Partial"
//	}
func writeBadgeJSON(counts statusCounts) error {
	type badge struct {
		Schema         int     `json:"schema"`
		Implemented    int     `json:"implemented"`
		Partial        int     `json:"partial"`
		Missing        int     `json:"missing"`
		Conceptual     int     `json:"conceptual"`
		Total          int     `json:"total"`
		ImplementedPct float64 `json:"implemented_pct"`
		WithPartialPct float64 `json:"with_partial_pct"`
		CoverageScore  string  `json:"coverage_score"`
		Generated      string  `json:"generated"`
	}
	if counts.Total == 0 {
		counts.Total = 1 // guard div by zero (should not happen in practice)
	}
	implPct := (float64(counts.Implemented) / float64(counts.Total)) * 100
	withPartialPct := (float64(counts.Implemented+counts.Partial) / float64(counts.Total)) * 100
	b := badge{
		Schema:         1,
		Implemented:    counts.Implemented,
		Partial:        counts.Partial,
		Missing:        counts.Missing,
		Conceptual:     counts.Conceptual,
		Total:          counts.Total,
		ImplementedPct: round2(implPct),
		WithPartialPct: round2(withPartialPct),
		CoverageScore:  fmt.Sprintf("%d%% Implemented / %d%% w+Partial", int(implPct+0.5), int(withPartialPct+0.5)),
		Generated:      time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outBadgeJSONPath, data, 0o644)
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

// splitEvidence splits the Evidence field on '|' while trimming spaces.
func splitEvidence(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, "|")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// appendHistory writes a single JSON object line with counts and percentages to the
// history JSONL file for trend analysis. Creates the file if it does not exist.
func appendHistory(counts statusCounts) error {
	if counts.Total == 0 {
		return nil
	}
	implPct := (float64(counts.Implemented) / float64(counts.Total)) * 100
	withPartialPct := (float64(counts.Implemented+counts.Partial) / float64(counts.Total)) * 100
	type hist struct {
		Timestamp      string  `json:"ts"`
		Implemented    int     `json:"implemented"`
		Partial        int     `json:"partial"`
		Missing        int     `json:"missing"`
		Conceptual     int     `json:"conceptual"`
		Total          int     `json:"total"`
		ImplementedPct float64 `json:"implemented_pct"`
		WithPartialPct float64 `json:"with_partial_pct"`
	}
	h := hist{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Implemented:    counts.Implemented,
		Partial:        counts.Partial,
		Missing:        counts.Missing,
		Conceptual:     counts.Conceptual,
		Total:          counts.Total,
		ImplementedPct: round2(implPct),
		WithPartialPct: round2(withPartialPct),
	}
	b, err := json.Marshal(h)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(string(b) + "\n"); err != nil {
		return err
	}
	return nil
}

// appendStatusHistory records the status of each requirement (by ID if present, else requirement text)
// for detection of newly Missing requirements in CI. Format (one line JSON):
// {"ts":"...","statuses":[{"id":"sec1.item1","status":"Implemented","priority":"P0"}, ...]}
func appendStatusHistory(rows []gapRow) error {
	type entry struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}
	type snapshot struct {
		TS       string  `json:"ts"`
		Statuses []entry `json:"statuses"`
	}
	snap := snapshot{TS: time.Now().UTC().Format(time.RFC3339)}
	for _, r := range rows {
		id := r.ID
		if id == "" {
			id = r.Requirement // fallback text key
		}
		snap.Statuses = append(snap.Statuses, entry{ID: id, Status: r.Status, Priority: r.Priority})
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(statusHistoryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(string(b) + "\n"); err != nil {
		return err
	}
	return nil
}
