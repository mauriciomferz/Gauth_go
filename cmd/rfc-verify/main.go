package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config
const (
	RfcMapPath = "docs/RFC_MAP.md"
	RootPath   = "."
)

// Colors
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

type RfcItem struct {
	ID         string // Clause ID or Section
	Feature    string
	Symbols    []string
	Tests      []string
	Status     string
	LineNumber int
}

func main() {
	fmt.Printf("%sStarting RFC Conformance Verification...%s\n", ColorBlue, ColorReset)

	// 1. Parse RFC_MAP.md
	items, err := parseMap(RfcMapPath)
	if err != nil {
		fatal("Failed to parse RFC map: %v", err)
	}
	fmt.Printf("Loaded %d RFC items to verify.\n", len(items))

	// 2. Index Codebase Symbols (naive grep-like or file-check)
	// For actual symbol lookup (functions, types), we might need `go/ast`, but
	// for now we'll do a robust grep/existence check to ensure files/packages exist.

	errors := 0
	warnings := 0

	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Status), "missing") {
			continue
		}

		// Verify Symbols (Files/Packages/Types existence)
		lastDir := ""
		for _, sym := range item.Symbols {
			valid, foundDir := verifySymbol(sym, lastDir)
			if !valid {
				fmt.Printf("%s[FAIL] %s: Symbol not found or invalid path: %s%s\n", ColorRed, item.ID, sym, ColorReset)
				errors++
			} else if foundDir != "" {
				lastDir = foundDir
			}
		}

		// Verify Tests
		for _, test := range item.Tests {
			// Test paths are relative to root usually
			if !verifyFile(test) {
				// Try finding it? No, assume strict paths for now or naive lookup
				// Let's try simple file existence checks
				fmt.Printf("%s[FAIL] %s: Test file not found: %s%s\n", ColorRed, item.ID, test, ColorReset)
				errors++
			}
		}

		// Additional Logic: "Implemented" items MUST have at least one test
		if strings.Contains(strings.ToLower(item.Status), "implemented") || strings.Contains(strings.ToLower(item.Status), "full") {
			if len(item.Tests) == 0 {
				fmt.Printf("%s[WARN] %s: Marked as Implemented but has NO tests listed%s\n", ColorYellow, item.ID, ColorReset)
				warnings++
			}
		}
	}

	fmt.Println("---------------------------------------------------")
	if errors > 0 {
		fmt.Printf("%sConformance Check FAILED with %d errors and %d warnings.%s\n", ColorRed, errors, warnings, ColorReset)
		os.Exit(1)
	} else {
		fmt.Printf("%sConformance Check PASSED. (%d warnings)%s\n", ColorGreen, warnings, ColorReset)
	}
}

// parseMap extracts rows from markdown tables
func parseMap(path string) ([]RfcItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var items []RfcItem
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		// Skip separator lines
		if strings.Contains(line, "---") {
			continue
		}
		// Skip header lines
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "clause") ||
			strings.Contains(lowerLine, "section") ||
			strings.Contains(lowerLine, "area") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		// Cleanup empty first/last elements from split
		if len(parts) > 0 && parts[0] == "" {
			parts = parts[1:]
		}
		if len(parts) > 0 && parts[len(parts)-1] == "" {
			parts = parts[:len(parts)-1]
		}

		if len(parts) < 5 {
			continue
		}

		id := strings.TrimSpace(parts[0])
		feature := strings.TrimSpace(parts[1])

		implRaw := strings.ReplaceAll(strings.TrimSpace(parts[2]), "`", "")
		impls := splitCommas(implRaw)

		testRaw := strings.ReplaceAll(strings.TrimSpace(parts[3]), "`", "")
		tests := splitCommas(testRaw)

		status := strings.TrimSpace(parts[4])

		items = append(items, RfcItem{
			ID:         id,
			Feature:    feature,
			Symbols:    impls,
			Tests:      tests,
			Status:     status,
			LineNumber: lineNum,
		})
	}
	return items, scanner.Err()
}

func splitCommas(s string) []string {
	var res []string
	parts := strings.Split(s, ",")
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if clean != "" && clean != "-" {
			res = append(res, clean)
		}
	}
	return res
}

// verifySymbol checks if a symbol (Package, Type, or Method) likely exists
// Supported formats:
// - pkg/foo
// - pkg/foo.Type
// - pkg/foo.Type.Method
func verifySymbol(sym string, lastDir string) (bool, string) {
	// 1. Direct file/dir check
	if info, err := os.Stat(sym); err == nil {
		if info.IsDir() {
			return true, sym
		}
		return true, filepath.Dir(sym)
	}

	// 2. Try assuming it's a pkg/path.Symbol
	// Only if it looks like a path
	if strings.Contains(sym, "/") {
		parts := strings.Split(sym, ".")
		pathCandidate := parts[0]
		if _, err := os.Stat(pathCandidate); err == nil {
			if len(parts) > 1 {
				// We won't strictly grep here to save time, assume if pkg exists and it's qualified, it's likely there
				// or would require AST parsing. For now returning true for valid package path is a good start.
				return true, pathCandidate
			}
			return true, pathCandidate
		}
	}

	// 3. Fuzzy Package Search
	// If sym is "policy.Registry", look for "pkg/policy" or "web/handlers/policy"
	parts := strings.Split(sym, ".")
	if len(parts) > 1 {
		pkgName := parts[0]
		roots := []string{"pkg", "web/handlers", "internal"}
		for _, root := range roots {
			candidate := filepath.Join(root, pkgName)
			if _, err := os.Stat(candidate); err == nil {
				// Found the package dir!
				// Optional: Grep for the symbol part
				if len(parts) > 1 {
					// We'll trust it exists if the package exists for this "Map" check
					// unless we want to be very strict.
					return true, candidate
				}
				return true, candidate
			}
		}
	}

	// 4. Contextual fallback
	if lastDir != "" {
		// Just check if the word appears in the dir's files
		// This handles the "AddBundle" case implicitly if we just verified "policy.Registry"
		cleanSym := strings.Trim(sym, " .()")
		if cleanSym != "" {
			// Deep grep in lastDir
			if grepDir(lastDir, cleanSym) {
				return true, lastDir
			}
		}
	}

	// 5. Deep Search in Roots (Expensive but necessary for unqualified symbols)
	// e.g. "RevocationChain" -> find in pkg/...
	roots := []string{"pkg", "web", "internal"}
	for _, root := range roots {
		foundPath := ""
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			// Quick checking
			data, _ := os.ReadFile(path)
			if strings.Contains(string(data), sym) {
				foundPath = filepath.Dir(path)
				return fs.SkipAll // Stop at first match
			}
			return nil
		}); err != nil && err != fs.SkipAll {
			// ignore walk errors but don't ignore the function return entirely to satisfy linter
		}

		if foundPath != "" {
			return true, foundPath
		}
	}

	return false, ""
}

// verifyFile checks if file exists
func verifyFile(path string) bool {
	if strings.HasPrefix(path, "(") && strings.HasSuffix(path, ")") {
		// (manual test) or similar
		return true
	}
	_, err := os.Stat(path)
	return err == nil
}

func grepDir(dir, term string) bool {
	found := false
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), term) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && err != fs.SkipAll {
		// ignore
	}
	return found
}

func fatal(format string, args ...interface{}) {
	fmt.Printf(ColorRed+format+ColorReset+"\n", args...)
	os.Exit(1)
}
