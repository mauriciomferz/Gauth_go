//go:build ignore

package main

// Conformance CI script: parses docs/RFC_MAP.md and ensures that for each row with Status of
// Implemented or Partial, at least one referenced test file exists in the repository.
// Exit non-zero if gaps detected, printing a machine-readable summary.
// Usage: go run scripts/conformance_ci.go

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Clause struct {
	ID       string
	Title    string
	ImplSyms []string
	Tests    []string
	Status   string
	Line     int
}

var rowRe = regexp.MustCompile(`^\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|`)

// RunConformance executes the conformance CI check logic. Returns exit code and optional error.
func RunConformance(args []string) (int, error) {
	jsonOnly := false
	for _, a := range args {
		if a == "--json" {
			jsonOnly = true
		}
	}
	mapPath := filepath.Join("docs", "RFC_MAP.md")
	f, err := os.Open(mapPath)
	if err != nil {
		return 2, fmt.Errorf("open RFC_MAP.md: %w", err)
	}
	defer f.Close()
	var clauses []Clause
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		m := rowRe.FindStringSubmatch(line)
		if len(m) == 6 {
			id := strings.TrimSpace(m[1])
			// Ignore header separator row and cross-cutting table entries
			if id == "Clause ID" || strings.HasPrefix(id, "---") || id == "Area" {
				continue
			}
			title := strings.TrimSpace(m[2])
			implCell := strings.TrimSpace(m[3])
			testsCell := strings.TrimSpace(m[4])
			status := strings.TrimSpace(m[5])
			var tests []string
			if testsCell != "--" && testsCell != "" && !strings.Contains(testsCell, "TODO") {
				// split by whitespace or comma
				for _, part := range strings.Split(testsCell, ",") {
					p := strings.TrimSpace(part)
					if p != "" {
						tests = append(tests, p)
					}
				}
			}
			var implSyms []string
			if implCell != "" && implCell != "--" && !strings.Contains(implCell, "(planned)") {
				// Extract only backticked segments, ignore narrative text
				// e.g. `policy.Registry`, `AddBundle`, `VerifyChain` => tokens
				btParts := strings.Split(implCell, "`")
				// Backticked tokens will appear at odd indices (1,3,5,...)
				for i := 1; i < len(btParts); i += 2 {
					token := strings.TrimSpace(btParts[i])
					if token != "" {
						implSyms = append(implSyms, token)
					}
				}
			}
			clauses = append(clauses, Clause{ID: id, Title: title, ImplSyms: implSyms, Tests: tests, Status: status, Line: lineNum})
		}
	}
	if err := scanner.Err(); err != nil {
		return 2, fmt.Errorf("scan RFC_MAP.md: %w", err)
	}
	// Build file existence cache
	repoRoot, _ := os.Getwd()
	existing := make(map[string]struct{})
	filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err == nil {
			existing[rel] = struct{}{}
		}
		return nil
	})
	// Pre-scan all source content for symbol presence (simple token matching)
	symbolFound := make(map[string]bool)
	// Build a map from filename to lines for test func scan later
	testFuncRe := regexp.MustCompile(`^func\s+(Test[A-Za-z0-9_]+)\s*\(`)
	fileContentCache := make(map[string][]string)
	for rel := range existing {
		// Only scan .go files
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		bs, err := os.ReadFile(rel)
		if err != nil {
			continue
		}
		content := string(bs)
		lines := strings.Split(content, "\n")
		fileContentCache[rel] = lines
		// Naive symbol token presence: mark any symbol whose raw substring appears
		// Will refine per clause later
	}
	// Collect all impl symbols
	var allSymbols []string
	for _, c := range clauses {
		allSymbols = append(allSymbols, c.ImplSyms...)
	}
	// Scan for symbols (word boundary approximate using regexp)
	for _, sym := range allSymbols {
		if sym == "" {
			continue
		}
		// Support dotted symbols like Type.Method or Type.Field
		var rx *regexp.Regexp
		if strings.Contains(sym, ".") {
			parts := strings.Split(sym, ".")
			// allow arbitrary whitespace or pointer receiver between type and method/field occurrences in code
			// Simplify: search for method/field name alone plus type name separately
			// Mark symbol found if either full pattern (Type.* name) or both pieces appear somewhere.
			tName := regexp.QuoteMeta(parts[0])
			mName := regexp.QuoteMeta(parts[1])
			fullPattern := regexp.MustCompile(tName + `.*` + mName)
			// quick two-pass scan
			foundFull := false
			foundType := false
			foundMember := false
			for _, lines := range fileContentCache {
				for _, ln := range lines {
					if !foundFull && fullPattern.FindStringIndex(ln) != nil {
						foundFull = true
					}
					if !foundType && regexp.MustCompile(`\b`+tName+`\b`).FindStringIndex(ln) != nil {
						foundType = true
					}
					if !foundMember && regexp.MustCompile(`\b`+mName+`\b`).FindStringIndex(ln) != nil {
						foundMember = true
					}
				}
				if foundFull || (foundType && foundMember) {
					symbolFound[sym] = true
					break
				}
			}
			continue
		} else {
			rx = regexp.MustCompile(`\b` + regexp.QuoteMeta(sym) + `\b`)
		}
		for _, lines := range fileContentCache {
			for _, ln := range lines {
				if rx.FindStringIndex(ln) != nil {
					symbolFound[sym] = true
					break
				}
			}
			if symbolFound[sym] {
				break
			}
		}
	}
	var missingTests []Clause
	var missingSymbols []Clause
	var missingSymbolNames []string
	var missingTestFuncs []Clause
	// Build identifier corpus for fuzzy suggestions (simple identifier extraction from .go files)
	identRx := regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)
	corpusSet := make(map[string]struct{})
	for rel, lines := range fileContentCache {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		for _, ln := range lines {
			for _, id := range identRx.FindAllString(ln, -1) {
				corpusSet[id] = struct{}{}
			}
		}
	}
	corpus := make([]string, 0, len(corpusSet))
	for id := range corpusSet {
		corpus = append(corpus, id)
	}
	sort.Strings(corpus)
	for _, c := range clauses {
		if c.Status == "Implemented" || c.Status == "Partial" {
			// Test file existence check
			testPresent := false
			if len(c.Tests) > 0 {
				for _, tf := range c.Tests {
					// direct path match
					if _, ok := existing[tf]; ok {
						testPresent = true
						break
					}
					base := filepath.Base(tf)
					for rel := range existing {
						if filepath.Base(rel) == base {
							testPresent = true
							break
						}
					}
					if testPresent {
						break
					}
				}
			}
			if !testPresent {
				missingTests = append(missingTests, c)
			}
			// Implementation symbol presence
			symMissing := false
			for _, sym := range c.ImplSyms {
				if sym != "" && !symbolFound[sym] {
					symMissing = true
					missingSymbolNames = append(missingSymbolNames, sym)
				}
			}
			if symMissing {
				missingSymbols = append(missingSymbols, c)
			}
			// Optional: check that referenced test files contain at least one Test* func
			if testPresent {
				hasTestFunc := false
				for _, tf := range c.Tests {
					base := filepath.Base(tf)
					for rel, lines := range fileContentCache {
						if filepath.Base(rel) == base {
							for _, ln := range lines {
								if testFuncRe.MatchString(strings.TrimSpace(ln)) {
									hasTestFunc = true
									break
								}
							}
						}
						if hasTestFunc {
							break
						}
					}
					if hasTestFunc {
						break
					}
				}
				if !hasTestFunc {
					missingTestFuncs = append(missingTestFuncs, c)
				}
			}
		}
	}
	if len(missingTests) == 0 && len(missingSymbols) == 0 && len(missingTestFuncs) == 0 {
		if jsonOnly {
			fmt.Print("{\n  \"status\": \"ok\", \"missing_tests\": [], \"missing_symbols\": [], \"missing_test_funcs\": []\n}\n")
		} else {
			fmt.Println("CONFORMANCE_OK: all Implemented/Partial clauses have test files, symbols present, and test funcs detected")
		}
		return 0, nil
	}
	if !jsonOnly {
		fmt.Printf("CONFORMANCE_GAPS: tests_missing=%d symbols_missing=%d testfuncs_missing=%d\n", len(missingTests), len(missingSymbols), len(missingTestFuncs))
		for _, c := range missingTests {
			fmt.Printf("- TEST_FILE_MISSING %s line=%d tests=%v\n", c.ID, c.Line, c.Tests)
		}
		for _, c := range missingSymbols {
			fmt.Printf("- IMPL_SYMBOL_MISSING %s line=%d symbols=%v\n", c.ID, c.Line, c.ImplSyms)
			for _, sym := range c.ImplSyms {
				if symbolFound[sym] {
					continue
				}
				if sugg := fuzzySuggest(sym, corpus); len(sugg) > 0 {
					fmt.Printf("  - SUGGEST symbol=%s suggestions=%v\n", sym, sugg)
				}
			}
		}
		for _, c := range missingTestFuncs {
			fmt.Printf("- TEST_FUNC_MISSING %s line=%d tests=%v\n", c.ID, c.Line, c.Tests)
		}
	}
	// JSON output (always produced if --json or at end for machine consumption)
	var b strings.Builder
	b.WriteString("{\n  \"status\": \"gaps\",\n  \"missing_tests\": [\n")
	for i, c := range missingTests {
		b.WriteString(fmt.Sprintf("    {\"id\":\"%s\",\"line\":%d}", c.ID, c.Line))
		if i < len(missingTests)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ],\n  \"missing_symbols\": [\n")
	for i, c := range missingSymbols {
		// Build suggestions map per symbol
		suggMap := make(map[string][]string)
		for _, sym := range c.ImplSyms {
			if symbolFound[sym] {
				continue
			}
			suggMap[sym] = fuzzySuggest(sym, corpus)
		}
		var parts []string
		for k, v := range suggMap {
			if len(v) > 0 {
				parts = append(parts, k+":"+strings.Join(v, ";"))
			}
		}
		b.WriteString(fmt.Sprintf("    {\"id\":\"%s\",\"line\":%d,\"symbols\":\"%s\",\"suggestions\":\"%s\"}", c.ID, c.Line, escape(strings.Join(c.ImplSyms, ",")), escape(strings.Join(parts, ","))))
		if i < len(missingSymbols)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ],\n  \"missing_test_funcs\": [\n")
	for i, c := range missingTestFuncs {
		b.WriteString(fmt.Sprintf("    {\"id\":\"%s\",\"line\":%d}", c.ID, c.Line))
		if i < len(missingTestFuncs)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ]\n}\n")
	fmt.Print(b.String())
	return 1, nil
}

func escape(s string) string {
	return strings.ReplaceAll(s, "\"", "'")
}

// fuzzySuggest returns up to 5 candidate identifiers similar to target using
// (a) case-insensitive substring match; if none found then (b) bounded Levenshtein distance (<=3).
// Corpus is assumed reasonably small (scanned identifiers); algorithm keeps it O(n*len) without heavy optimization.
func fuzzySuggest(target string, corpus []string) []string {
	if target == "" {
		return nil
	}
	tl := strings.ToLower(target)
	// Collect substring matches first
	subs := make([]string, 0, 5)
	for _, id := range corpus {
		if len(subs) >= 5 {
			break
		}
		if strings.Contains(strings.ToLower(id), tl) && id != target {
			subs = append(subs, id)
		}
	}
	if len(subs) > 0 {
		return subs
	}
	// Levenshtein with early exit if distance > 3
	type cand struct {
		w string
		d int
	}
	cands := make([]cand, 0, 8)
	for _, id := range corpus {
		if id == target {
			continue
		}
		d := levenshteinBounded(target, id, 3)
		if d >= 0 && d <= 3 {
			cands = append(cands, cand{w: id, d: d})
		}
	}
	// Sort by distance then lexicographically
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].d == cands[j].d {
			return cands[i].w < cands[j].w
		}
		return cands[i].d < cands[j].d
	})
	out := make([]string, 0, 5)
	for _, c := range cands {
		if len(out) >= 5 {
			break
		}
		out = append(out, c.w)
	}
	return out
}

// levenshteinBounded computes Levenshtein distance with an upper bound; returns -1 if distance exceeds bound.
func levenshteinBounded(a, b string, max int) int {
	ar := []rune(a)
	br := []rune(b)
	la := len(ar)
	lb := len(br)
	if la == 0 {
		if lb <= max {
			return lb
		}
		return -1
	}
	if lb == 0 {
		if la <= max {
			return la
		}
		return -1
	}
	// If absolute length diff > max distance quick reject
	if la-lb > max || lb-la > max {
		return -1
	}
	// DP rows reduced to two slices; early abandon if min in row > max
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		minRow := cur[0]
		for j := 1; j <= lb; j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			v := del
			if ins < v {
				v = ins
			}
			if sub < v {
				v = sub
			}
			cur[j] = v
			if v < minRow {
				minRow = v
			}
		}
		if minRow > max {
			return -1
		}
		prev, cur = cur, prev
	}
	if prev[lb] > max {
		return -1
	}
	return prev[lb]
}
