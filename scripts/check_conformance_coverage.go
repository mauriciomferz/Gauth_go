//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	rfMapPath := "docs/RFC_MAP.md"
	if len(os.Args) > 1 {
		rfMapPath = os.Args[1]
	}

	fmt.Printf("🔍 Checking conformance coverage in %s...\n", rfMapPath)

	file, err := os.Open(rfMapPath)
	if err != nil {
		fmt.Printf("❌ Failed to open file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	inTable := false
	testsColIndex := -1
	failures := 0
	checks := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		trimmed := strings.TrimSpace(line)

		// Detect table start/header
		if strings.HasPrefix(trimmed, "|") {
			parts := splitTableLine(trimmed)
			if len(parts) == 0 {
				continue
			}

			// check if header row
			if !inTable {
				colIdx := findColumnIndex(parts, "Tests")
				if colIdx != -1 {
					inTable = true
					testsColIndex = colIdx
					continue // skip header
				} else {
					// Maybe a different kind of table (Section/Feature/Conformance), skip for now
					continue
				}
			}

			// Skip separator row
			if strings.Contains(parts[0], "---") {
				continue
			}

			// Data row
			if testsColIndex != -1 && testsColIndex < len(parts) {
				testsVal := strings.TrimSpace(parts[testsColIndex])
				if testsVal != "" && testsVal != "-" {
					paths := strings.Split(testsVal, ",")
					for _, p := range paths {
						p = strings.TrimSpace(p)
						p = strings.Trim(p, "`") // Strip markdown code ticks
						if p == "" {
							continue
						}
						checks++
						if err := verifyPath(p); err != nil {
							fmt.Printf("❌ Line %d: Test file not found: %s\n", lineNum, p)
							failures++
						} else {
							// fmt.Printf("✅ %s\n", p)
						}
					}
				}
			}

		} else {
			// Not a table line, reset
			inTable = false
			testsColIndex = -1
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("❌ Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Total Checks: %d\n", checks)
	fmt.Printf("Failures: %d\n", failures)

	if failures > 0 {
		os.Exit(1)
	}
	fmt.Println("✅ All compliance links verified.")
}

func splitTableLine(line string) []string {
	// Remove leading/trailing pipes if present
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	return strings.Split(line, "|")
}

func findColumnIndex(parts []string, name string) int {
	for i, p := range parts {
		if strings.EqualFold(strings.TrimSpace(p), name) {
			return i
		}
	}
	return -1
}

func verifyPath(path string) error {
	// Paths in RFC_MAP are usually relative to project root
	// We assume script run from project root
	_, err := os.Stat(path)
	return err
}
