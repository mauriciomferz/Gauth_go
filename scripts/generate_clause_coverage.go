//go:build ignore

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CoverageSection represents one RFC clause mapping.
type CoverageSection struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Expected []string  `json:"expected"`
	Tests    []TestRef `json:"tests"`
}

// TestRef stores test reference mapping.
type TestRef struct {
	File string `json:"file"`
	Func string `json:"func"`
}

type Rfc struct {
	Sections []CoverageSection `json:"sections"`
}

type Template struct {
	Generated string         `json:"generated"`
	RFC       map[string]Rfc `json:"aap"`
}

var (
	markerPrefix = "//clause:"
	// Matches Go test func signature: func TestXxx(t *testing.T) {
	testFuncRx = regexp.MustCompile(`^func (Test[^(]+)\(`)
)

// RunClauseCoverage executes the clause coverage generation workflow.
func RunClauseCoverage() error {
	templatePath := "docs/coverage_template.json"
	outPath := "docs/CLAUSE_TEST_COVERAGE.json"

	b, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	var tpl Template
	if err := json.Unmarshal(b, &tpl); err != nil {
		return fmt.Errorf("unmarshal template: %w", err)
	}

	// Prepare marker map: clause -> []TestRef
	markers := make(map[string][]TestRef)

	// Walk test files
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			// Skip vendor, bin, build
			base := filepath.Base(path)
			if base == "vendor" || base == "bin" || base == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return processTestFile(path, markers)
	})
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	// Inject tests into template
	for/aapID, r := range tpl.RFC {
		for i, sec := range r.Sections {
			if refs, ok := markers[sec.ID]; ok {
				sec.Tests = append(sec.Tests, refs...)
			}
			r.Sections[i] = sec
		}
		tpl.RFC/aapID] = r
	}
	// Coverage metrics
	covered, total := 0, 0
	for _, r := range tpl.RFC {
		for _, s := range r.Sections {
			if len(s.Tests) > 0 {
				covered++
			}
			total++
		}
	}
	pct := 0.0
	if total > 0 {
		pct = (float64(covered) / float64(total) * 100
	}

	tpl.Generated = time.Now().UTC().Format(time.RFC3339)

	out, err := json.MarshalIndent(tpl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("Clause coverage: %d/%d (%.2f%%)\n", covered, total, pct)
	if covered == 0 {
		return errors.New("no clauses covered; add //clause markers to tests")
	}

	return nil
}

func processTestFile(path string, markers map[string][]TestRef) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open test file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var currentMarkers []string
	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, markerPrefix) {
			clause := strings.TrimPrefix(trim, markerPrefix)
			clause = strings.TrimSpace(clause)
			if clause != "" {
				currentMarkers = append(currentMarkers, clause)
			}
			continue
		}
		if strings.HasPrefix(trim, "func Test") {
			// Extract function name
			if m := testFuncRx.FindStringSubmatch(trim); len(m) == 2 {
				fn := m[1]
				for _, clause := range currentMarkers {
					markers[clause] = append(markers[clause], TestRef{File: path, Func: fn})
				}
				currentMarkers = nil // reset for next function
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	return nil
}
