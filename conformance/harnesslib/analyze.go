// Package harnesslib contains the active conformance analysis implementation: scanning RFC
// markdown clauses, mapping symbols, computing coverage and GAP metrics, and exporting reports.
package harnesslib

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type AnalysisResult struct {
	Clauses   []Clause
	Failures  []string
	Summary   Summary
	Evidence  map[string][]string // symbol -> file:line list
	GapMatrix gapMatrix
}

type mappingEntry struct {
	ClausePrefix string   `json:"clause_prefix"`
	Symbols      []string `json:"symbols"`
	TestsGlob    string   `json:"tests_glob"`
}

type clauseMap struct {
	Entries []mappingEntry `json:"entries"`
}

type Summary struct {
	MappedClauses      int     `json:"mapped_clauses"`
	MappedClausesFound int     `json:"mapped_clauses_found"`
	RequiredSymbols    int     `json:"required_symbols"`
	SymbolsFound       int     `json:"symbols_found"`
	ClausesWithTests   int     `json:"clauses_with_tests"`
	ClausesTestsFound  int     `json:"clauses_tests_found"`
	CoveragePercent    float64 `json:"coverage_percent"`
	MissingClauses     int     `json:"missing_clauses"`
	MissingSymbols     int     `json:"missing_symbols"`
	MissingTests       int     `json:"missing_tests"`
	GapTotal           int     `json:"gap_total"`
	GapImplemented     int     `json:"gap_implemented"`
	GapPartial         int     `json:"gap_partial"`
	GapMissing         int     `json:"gap_missing"`
}

// GAP matrix structures
type gapItem struct {
	ID          string   `json:"id"`
	Requirement string   `json:"requirement"`
	Status      string   `json:"status"`
	Gap         string   `json:"gap"`
	Evidence    []string `json:"evidence"`
	Priority    string   `json:"priority"`
}
type gapSection struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Items []gapItem `json:"items"`
}
type gapMatrix struct {
	Generated string       `json:"generated"`
	Version   string       `json:"version"`
	Sections  []gapSection `json:"sections"`
}

func loadGapMatrix() (gapMatrix, error) {
	var gm gapMatrix
	path := os.Getenv("GAP_MATRIX_PATH")
	if path == "" {
		path = "docs/conformance_gaps.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return gm, err
	}
	if err := json.Unmarshal(b, &gm); err != nil {
		return gm, err
	}
	return gm, nil
}

// BuildSymbolIndex walks the repository root gathering top-level function, type, var, const identifiers.
// Returns map[symbol][]file:line and slice of parse/read errors encountered.
func BuildSymbolIndex(root string) (map[string][]string, []string) {
	var goFiles []string
	var parseErrors []string
	includeTests := os.Getenv("CONFORMANCE_INCLUDE_TESTS") == "1"
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".go" {
			if strings.Contains(path, string(os.PathSeparator)+"vendor"+string(os.PathSeparator)) {
				return nil
			}
			if !includeTests && strings.HasSuffix(path, "_test.go") {
				return nil
			}
			goFiles = append(goFiles, path)
		}
		return nil
	}); err != nil {
		parseErrors = append(parseErrors, fmt.Sprintf("WalkDir error: %v", err))
	}
	fset := token.NewFileSet()
	symbolLoc := map[string][]string{}
	for _, gf := range goFiles {
		src, err := os.ReadFile(gf)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("read:%s:%v", gf, err))
			continue
		}
		file, err := parser.ParseFile(fset, gf, src, parser.SkipObjectResolution)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("parse:%s:%v", gf, err))
			continue
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				pos := fset.Position(d.Pos())
				name := d.Name.Name
				symbolLoc[name] = append(symbolLoc[name], formatLoc(gf, pos.Line))
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						pos := fset.Position(s.Pos())
						symbolLoc[s.Name.Name] = append(symbolLoc[s.Name.Name], formatLoc(gf, pos.Line))
					case *ast.ValueSpec:
						for _, id := range s.Names {
							pos := fset.Position(id.Pos())
							symbolLoc[id.Name] = append(symbolLoc[id.Name], formatLoc(gf, pos.Line))
						}
					}
				}
			}
		}
	}
	return symbolLoc, parseErrors
}

// Analyze matches clauses against mapping entries and verifies symbol & test presence.
func Analyze(clauses []Clause) AnalysisResult {
	cm, _ := loadClauseMap()
	gm, _ := loadGapMatrix()
	var failures []string
	summary := Summary{}
	evidence := map[string][]string{}
	// Determine repository root: harnesslib lives under conformance/harnesslib so go two levels up.
	cwd, _ := filepath.Abs(".")
	// If current working directory already looks like project root (contains go.mod) keep it; otherwise ascend.
	root := cwd
	if _, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr != nil {
		// Try parent directories.
		parent := filepath.Dir(root)
		if _, statErr2 := os.Stat(filepath.Join(parent, "go.mod")); statErr2 == nil {
			root = parent
		} else {
			gp := filepath.Dir(parent)
			if _, statErr3 := os.Stat(filepath.Join(gp, "go.mod")); statErr3 == nil {
				root = gp
			}
		}
	}

	debug := os.Getenv("CONFORMANCE_DEBUG") == "1"
	symbolLoc, parseErrors := BuildSymbolIndex(root)

	// testsExist closure uses repo root for glob resolution.
	testsExist := func(glob string) bool {
		matches, _ := filepath.Glob(filepath.Join(root, glob))
		return len(matches) > 0
	}
	summary.MappedClauses = len(cm.Entries)
	for _, me := range cm.Entries {
		clauseFound := false
		for _, c := range clauses {
			if c.ID == me.ClausePrefix || strings.HasPrefix(c.ID, me.ClausePrefix) {
				clauseFound = true
				break
			}
			// Normalized comparison fallback (drops numbering/punctuation)
			if normalizeClauseID(c.ID) == normalizeClauseID(me.ClausePrefix) {
				clauseFound = true
				break
			}
		}
		if !clauseFound {
			failures = append(failures, "clause missing: "+me.ClausePrefix)
			summary.MissingClauses++
			// Do NOT count symbols towards required coverage if clause not found; treat as structural gap first.
			continue
		}
		summary.MappedClausesFound++
		summary.RequiredSymbols += len(me.Symbols)
		for _, symOrig := range me.Symbols {
			// Allow package-qualified names (e.g., policy.Registry) by stripping prefix before lookup.
			sym := symOrig
			if idx := strings.LastIndex(symOrig, "."); idx != -1 {
				sym = symOrig[idx+1:]
			}
			locs := symbolLoc[sym]
			if len(locs) == 0 {
				failures = append(failures, "symbol missing: "+me.ClausePrefix+" -> "+symOrig)
				summary.MissingSymbols++
			} else {
				summary.SymbolsFound++
				evidence[symOrig] = append(evidence[symOrig], locs...)
			}
		}
		if me.TestsGlob != "" {
			summary.ClausesWithTests++
			if testsExist(me.TestsGlob) {
				summary.ClausesTestsFound++
			} else {
				failures = append(failures, "tests missing: "+me.ClausePrefix+" glob="+me.TestsGlob)
				summary.MissingTests++
			}
		}
	}
	sort.Strings(failures)
	for k := range evidence {
		sort.Strings(evidence[k])
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[conformance] root=%s symbols=%d failures=%d parse_errs=%d\n", root, len(symbolLoc), len(failures), len(parseErrors))
		if locs := symbolLoc["CanonicalPOADigest"]; len(locs) == 0 {
			fmt.Fprintf(os.Stderr, "[conformance][warn] CanonicalPOADigest not indexed\n")
		} else {
			fmt.Fprintf(os.Stderr, "[conformance][info] CanonicalPOADigest locs=%v\n", locs)
		}
	}

	if summary.RequiredSymbols > 0 {
		summary.CoveragePercent = float64(summary.SymbolsFound) / float64(summary.RequiredSymbols) * 100.0
	}
	// GAP matrix counts
	for _, sec := range gm.Sections {
		for _, it := range sec.Items {
			summary.GapTotal++
			switch strings.ToLower(it.Status) {
			case "implemented":
				summary.GapImplemented++
			case "partial":
				summary.GapPartial++
			case "missing":
				summary.GapMissing++
			}
		}
	}
	return AnalysisResult{Clauses: clauses, Failures: failures, Summary: summary, Evidence: evidence, GapMatrix: gm}
}

func loadClauseMap() (clauseMap, error) {
	var cm clauseMap
	b, err := os.ReadFile("conformance/clause_map.json")
	if err != nil {
		return cm, err
	}
	_ = json.Unmarshal(b, &cm)
	return cm, nil
}

type Report struct {
	GeneratedAt       string              `json:"generated_at"`
	Clauses           []Clause            `json:"clauses"`
	Failures          []string            `json:"failures"`
	Summary           Summary             `json:"summary"`
	Evidence          map[string][]string `json:"evidence"`
	FailureCategories map[string][]string `json:"failure_categories"`
	GapMatrix         gapMatrix           `json:"gap_matrix"`
}

func BuildReport(ar AnalysisResult) Report {
	cats := map[string][]string{"clause_missing": {}, "symbol_missing": {}, "tests_missing": {}}
	for _, f := range ar.Failures {
		switch {
		case strings.HasPrefix(f, "clause missing:"):
			cats["clause_missing"] = append(cats["clause_missing"], f)
		case strings.HasPrefix(f, "symbol missing:"):
			cats["symbol_missing"] = append(cats["symbol_missing"], f)
		case strings.HasPrefix(f, "tests missing:"):
			cats["tests_missing"] = append(cats["tests_missing"], f)
		}
	}
	return Report{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Clauses: ar.Clauses, Failures: ar.Failures, Summary: ar.Summary, Evidence: ar.Evidence, FailureCategories: cats, GapMatrix: ar.GapMatrix}
}

func formatLoc(file string, line int) string { return file + ":" + itoa(line) }

// normalizeClauseID reduces a clause id to a stable matching token: lowercase, remove digits & punctuation except colons.
func normalizeClauseID(id string) string {
	id = strings.ToLower(id)
	// Replace punctuation (except colon) with hyphen, then collapse multiple hyphens.
	repl := func(r rune) rune {
		if r == ':' {
			return r
		}
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return -1
		} // drop digits (section numbering)
		return '-' // punctuation/space -> dash
	}
	var b strings.Builder
	prevDash := false
	for _, r := range id {
		rr := repl(r)
		if rr == -1 {
			continue
		}
		if rr == '-' {
			if prevDash {
				continue
			}
			prevDash = true
			b.WriteRune('-')
		} else {
			prevDash = false
			b.WriteRune(rr)
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
