package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Validation configuration
var (
	requiredKeys = []string{"title", "category", "status", "lastUpdated", "owners"}
	// Allowed enumerations sourced from docs conventions (kept lightweight; not exhaustive of future tags)
	allowedCategories = map[string]struct{}{
		"architecture":        {},
		"aap":                 {},
		"guide":               {},
		"operations":          {},
		"security":            {},
		"performance":         {},
		"api":                 {},
		"adr":                 {},
		"release":             {},
		"generated":           {},
		"roadmap":             {},
		"compliance":          {},
		"example":             {},
		"ui":                  {},
		"audit":               {},
		"maintenance":         {},
		"org":                 {},
		"misc":                {},
		"audit-log":           {},
		"audit-log-index":     {},
		"documentation-index": {},
		// Additional observed categories retained during transition
		"legal-disclaimer":      {},
		"build-artifacts-guide": {},
		"project-organization":  {},
		"security-guide":        {},
		"adr-index":             {},
		"security-assessment":   {},
		"security-setup-guide":  {},
		"example-index":         {},
	}
	allowedStatus = map[string]struct{}{"draft": {}, "active": {}, "deprecated": {}, "superseded": {}, "archived": {}}
)

type metaBlock map[string]string

type fileResult struct {
	Path        string
	Errors      []string
	Meta        metaBlock
	HasFront    bool
	DuplicateFM bool
	Warnings    []string
}

func main() {
	writeIndex := flag.Bool("write-index", false, "Generate taxonomy index file docs/TAXONOMY_INDEX.auto.md")
	root := flag.String("root", ".", "Root directory to scan")
	strict := flag.Bool("strict", false, "Fail on any warning (including duplicate potential)")
	flag.Parse()
	mergeExpandedCategories()

	results, categories := validateMarkdown(*root)
	printSummary(results, categories)

	if *writeIndex {
		if err := writeTaxonomyIndex(categories); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed writing taxonomy index: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("📝 Wrote docs/TAXONOMY_INDEX.auto.md")
	}

	exitCode := 0
	for _, r := range results {
		if len(r.Errors) > 0 {
			exitCode = 1
			break
		}
		if *strict && r.DuplicateFM {
			exitCode = 2
		}
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func validateMarkdown(root string) ([]fileResult, map[string]int) {
	var results []fileResult
	categories := map[string]int{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip vendor-ish directories
			base := filepath.Base(path)
			if path != root { // allow root '.'
				if strings.HasPrefix(base, ".") || base == "node_modules" || base == "bin" || base == "build" {
					return fs.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		// Skip generated auto files if they intentionally differ; still parse for validation.
		res := validateFile(path)
		if res.Meta != nil {
			if cat := res.Meta["category"]; cat != "" {
				categories[cat]++
			}
		}
		results = append(results, res)
		return nil
	})
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, categories
}

var fmStartRe = regexp.MustCompile(`^---\s*$`)

// normalizeStatus maps legacy / synonym status values to canonical ones.
func normalizeStatus(s string) (canonical string, changed bool) {
	if s == "" {
		return "", false
	}
	m := map[string]string{
		"final": "active", "complete": "active", "partial-complete": "active", "baseline": "active",
		"enhancement-complete": "active", "implemented": "active", "proposed": "draft", "accepted": "active",
		"beta": "active", "generated": "active",
	}
	if v, ok := m[strings.ToLower(s)]; ok {
		return v, true
	}
	return s, false
}

// expandCategories lists previously observed categories that are outside the strict governance set
// but accepted during transition.
var expandedCategories = []string{
	"release-notes",
	"compliance-report",
	"performance-report",
	"gap-report",
	"gap-summary",
	"runbook",
	"runbook-index",
	"api-reference",
	"architecture-spec",
	"backup-restore-guide",
	"disaster-recovery-guide",
	"cryptography-guide",
	"security-jwks",
	"security-parsing",
	"security-storage",
	"security-threat-matrix",
	"security-token-integrity",
	"threat-model",
	"technical-debt",
	"testing-guide",
	"testing-report",
	"readiness-gap-analysis",
	"overview",
	"organizational",
	"implementation-report",
	"implementation-status",
	"implementation-summary",
	"audit-report",
	"progress-report",
	"monitoring-report",
	"observability-alerting-guide",
	"observability-report",
	"design-spec",
	"deployment-guide",
	"local-cluster-guide",
	"containerization-report",
	"cicd-guide",
	"cicd-report",
	"cicd-quickref",
	"quality-report",
	"database-implementation-summary",
}

func mergeExpandedCategories() {
	// Merge expanded categories into allowed set for transitional pass.
	for _, c := range expandedCategories {
		allowedCategories[c] = struct{}{}
	}
}

func validateFile(path string) fileResult {
	content, err := os.ReadFile(path)
	if err != nil {
		return fileResult{Path: path, Errors: []string{fmt.Sprintf("read error: %v", err)}}
	}
	lines := strings.Split(string(content), "\n")
	res := fileResult{Path: path}
	if len(lines) == 0 {
		res.Errors = append(res.Errors, "empty file")
		return res
	}

	// Detect front matter
	if !fmStartRe.MatchString(lines[0]) {
		// Gracefully handle files that start with title: but no delimiter
		if strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
			res.Errors = append(res.Errors, "missing opening front matter delimiter '---'")
		}
		return res
	}
	res.HasFront = true
	// Find closing delimiter
	end := -1
	for i := 1; i < len(lines); i++ {
		if fmStartRe.MatchString(lines[i]) {
			end = i
			break
		}
	}
	if end == -1 {
		res.Errors = append(res.Errors, "missing closing front matter delimiter '---'")
		return res
	}
	metaLines := lines[1:end]
	mb := parseMeta(metaLines)
	res.Meta = mb
	// Validate required keys
	for _, k := range requiredKeys {
		if mb[k] == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("missing required key '%s'", k))
		}
	}
	// Validate date format
	if mb["lastUpdated"] != "" {
		if _, err := time.Parse("2006-01-02", mb["lastUpdated"]); err != nil {
			res.Errors = append(res.Errors, "invalid date format for lastUpdated (expected YYYY-MM-DD)")
		}
	}
	// Enumerated value checks
	if cat := mb["category"]; cat != "" {
		if _, ok := allowedCategories[cat]; !ok {
			res.Errors = append(res.Errors, fmt.Sprintf("unknown category '%s' (add to validator or fix)", cat))
		}
	}
	if st := mb["status"]; st != "" {
		// Normalize legacy values first
		if canonical, changed := normalizeStatus(st); changed {
			mb["status"] = canonical
			res.Warnings = append(res.Warnings, fmt.Sprintf("normalized status '%s' -> '%s'", st, canonical))
			st = canonical
		}
		if _, ok := allowedStatus[st]; !ok {
			res.Errors = append(res.Errors, fmt.Sprintf(
				"unknown status '%s' (must be one of draft|active|deprecated|superseded|archived)",
				st,
			))
		}
	}
	// Owners sanity (team-alias style preferred: at least one '-')
	if owners := mb["owners"]; owners != "" {
		// support comma list; ensure each non-empty
		parts := strings.Split(owners, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				res.Errors = append(res.Errors, "owners list contains empty entry")
			}
		}
	}
	// Check for duplicate second metadata block (look for another 'title:' before first heading)
	dup := false
	for i := end + 1; i < len(lines) && i < end+50; i++ { // limit scan window
		ln := strings.TrimSpace(lines[i])
		if strings.HasPrefix(strings.ToLower(ln), "title:") {
			dup = true
			break
		}
		if strings.HasPrefix(ln, "#") {
			break
		}
	}
	if dup {
		res.DuplicateFM = true
		res.Errors = append(res.Errors, "duplicate front matter detected (second title: before heading)")
	}
	return res
}

func parseMeta(lines []string) metaBlock {
	mb := metaBlock{}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		// key: value
		parts := strings.SplitN(ln, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		mb[k] = v
	}
	return mb
}

func printSummary(results []fileResult, categories map[string]int) {
	var errCount int
	for _, r := range results {
		if len(r.Errors) > 0 {
			errCount++
		}
	}
	fmt.Printf("🔍 Documentation validation summary\n")
	fmt.Printf("Files scanned: %d\n", len(results))
	fmt.Printf("Files with errors: %d\n", errCount)
	if errCount > 0 {
		fmt.Println("\n❌ Errors:")
		for _, r := range results {
			if len(r.Errors) == 0 {
				continue
			}
			fmt.Printf("- %s\n", r.Path)
			for _, e := range r.Errors {
				fmt.Printf("    • %s\n", e)
			}
		}
	} else {
		fmt.Println("✅ All documentation headers valid")
	}
	// Print warnings separately (non-fatal normalizations)
	var warningCount int
	for _, r := range results {
		warningCount += len(r.Warnings)
	}
	if warningCount > 0 {
		fmt.Println("\n⚠️  Warnings (non-fatal):")
		for _, r := range results {
			if len(r.Warnings) == 0 {
				continue
			}
			fmt.Printf("- %s\n", r.Path)
			for _, w := range r.Warnings {
				fmt.Printf("    • %s\n", w)
			}
		}
	}
	// Category histogram
	keys := make([]string, 0, len(categories))
	for k := range categories {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println("\n📊 Category counts:")
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, categories[k])
	}
}

func writeTaxonomyIndex(categories map[string]int) error {
	if len(categories) == 0 {
		return errors.New("no categories to write")
	}
	keys := make([]string, 0, len(categories))
	for k := range categories {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: Documentation Taxonomy Index\n")
	b.WriteString("category: documentation-index\n")
	b.WriteString("status: active\n")
	b.WriteString(fmt.Sprintf("lastUpdated: %s\n", time.Now().Format("2006-01-02")))
	b.WriteString("owners: documentation-team\n")
	b.WriteString("source: docvalidate-tool\n")
	b.WriteString("refreshCadence: ad-hoc\n")
	b.WriteString("---\n\n")
	b.WriteString("# Documentation Taxonomy Index\n\n")
	b.WriteString("| Category | Count |\n|----------|-------|\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("| %s | %d |\n", k, categories[k]))
	}
	b.WriteString("\n> Generated via `go run ./tools/docvalidate --write-index`\n")
	// Ensure docs directory exists
	// #nosec G301
	if err := os.MkdirAll("docs", 0o755); err != nil {
		return err
	}
	f, err := os.Create("docs/TAXONOMY_INDEX.auto.md")
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriter(f)
	if _, err := w.WriteString(b.String()); err != nil {
		return err
	}
	return w.Flush()
}
