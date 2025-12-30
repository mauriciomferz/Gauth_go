// Package conformance provides lightweight utilities for mapping test functions
// to documented clause identifiers (RFC / ADR) to produce simple coverage reports.
// Experimental; output formats may evolve.
package conformance

// coverage_report.go: tooling to generate a clause-to-test coverage matrix.
// Initial scaffold focuses on collecting test names and mapping them to documented clauses.
// Future expansion will parse RFC / ADR markdown files for clause identifiers.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ClauseCoverage represents mapping of a clause ID to tests that assert it.
type ClauseCoverage struct {
	ClauseID string   `json:"clause_id"`
	Tests    []string `json:"tests"`
}

// Report is the top-level structure for JSON/Markdown emission.
type Report struct {
	Clauses []ClauseCoverage `json:"clauses"`
	Orphans []string         `json:"orphans"` // tests without recognized clause tag
}

// ExtractTests walks the repository gathering Go test function names (TestXxx).
func ExtractTests(root string) ([]string, error) {
	var tests []string
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil {
				continue
			}
			name := fn.Name.Name
			if strings.HasPrefix(name, "Test") {
				tests = append(tests, name)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tests, nil
}

// DiscoverClauseTags maps tests to clause IDs by looking for inline comments like: // CLAUSE: AAP001-3.2.1
func DiscoverClauseTags(root string) (map[string][]string, error) {
	mapping := make(map[string][]string)
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				txt := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
				if strings.HasPrefix(txt, "CLAUSE:") {
					id := strings.TrimSpace(strings.TrimPrefix(txt, "CLAUSE:"))
					if id != "" {
						// Associate with file-level tests; fine-grained association could parse surrounding AST nodes.
						mapping[id] = append(mapping[id], filepath.Base(path))
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

// GenerateReport builds a coverage report struct using discovered clause tags and test list.
func GenerateReport(root string) (*Report, error) {
	tests, err := ExtractTests(root)
	if err != nil {
		return nil, err
	}
	clauseMap, err := DiscoverClauseTags(root)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	clauses := make([]ClauseCoverage, 0, len(clauseMap))
	for id, files := range clauseMap {
		clauses = append(clauses, ClauseCoverage{ClauseID: id, Tests: files})
		for _, t := range files {
			seen[t] = true
		}
	}
	var orphans []string
	for _, t := range tests {
		if !seen[t+".go"] { // heuristic
			orphans = append(orphans, t)
		}
	}
	return &Report{Clauses: clauses, Orphans: orphans}, nil
}

// EmitMarkdown renders a simple markdown table for the coverage report.
func EmitMarkdown(r *Report) string {
	var b strings.Builder
	b.WriteString("# Clause Coverage\n\n")
	b.WriteString("| Clause ID | Tests |\n|-----------|-------|\n")
	for _, c := range r.Clauses {
		b.WriteString(fmt.Sprintf("| %s | %s |\n", c.ClauseID, strings.Join(c.Tests, ", ")))
	}
	if len(r.Orphans) > 0 {
		b.WriteString("\n## Orphan Tests\n\n")
		for _, o := range r.Orphans {
			b.WriteString("- " + o + "\n")
		}
	}
	return b.String()
}

// Placeholder main-style helper (not an entrypoint) for future CLI wiring.
func GenerateAndPrintMarkdown(root string) (string, error) {
	rep, err := GenerateReport(root)
	if err != nil {
		return "", err
	}
	md := EmitMarkdown(rep)
	return md, nil
}
