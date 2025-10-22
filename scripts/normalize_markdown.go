//go:build ignore

package main

// normalize_markdown.go
// Bulk normalization tool for project-owned Markdown files.
// Usage: go run scripts/normalize_markdown.go
// Applies header (Title, Last Updated, Status), footer, and light formatting rules.
// Exclusions: node_modules/, CHANGELOG.md, docs/GENERATED_API.md
// Safe against fenced code blocks (does not mutate inside them beyond trailing space trim).

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var skipBase = map[string]struct{}{ // file basenames to skip wholesale
	"CHANGELOG.md":     {},
	"GENERATED_API.md": {},
}

var footerMarker = "Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md"

func main() {
	root, _ := os.Getwd()
	date := time.Now().Format("2006-01-02")
	var processed, changed int
	var modifiedFiles []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if _, skip := skipBase[d.Name()]; skip {
			return nil
		}
		// Only project-owned: skip vendored or binary doc dirs heuristics if needed
		// (Currently just node_modules handled.)
		if err2 := normalizeFile(path, date, &modifiedFiles); err2 != nil {
			fmt.Fprintf(os.Stderr, "[warn] %s: %v\n", path, err2)
		} else {
			processed++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking: %v\n", err)
		os.Exit(1)
	}
	changed = len(modifiedFiles)
	fmt.Printf("Normalization complete. Files scanned=%d, modified=%d\n", processed, changed)
	if changed > 0 {
		fmt.Println("Modified files:")
		for _, f := range modifiedFiles {
			fmt.Println(" - ", f)
		}
	}
}

var (
	headingRegex     = regexp.MustCompile(`^# +.+`)
	lastUpdatedRegex = regexp.MustCompile(`^> +Last Updated: +\d{4}-\d{2}-\d{2}`)
	statusRegex      = regexp.MustCompile(`^> +Status: +.*`)
)

func deriveTitle(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".md")
	// Replace separators with space then title case
	parts := strings.FieldsFunc(base, func(r rune) bool { return r == '-' || r == '_' })
	c := cases.Title(language.English)
	for i, p := range parts {
		parts[i] = c.String(p)
	}
	return strings.Join(parts, " ")
}

func normalizeFile(path, date string, modified *[]string) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := splitLines(string(original))
	if len(lines) == 0 {
		return nil
	}

	changed := false
	inFence := false
	fenceDelimiter := "" // track ``` or ````

	// Detect existing heading
	idx := firstNonEmpty(lines)
	title := deriveTitle(path)
	if idx == -1 { // empty file, create minimal scaffold
		lines = []string{"# " + title, "", fmt.Sprintf("> Last Updated: %s", date), "> Status: Active", ""}
		changed = true
	} else if !headingRegex.MatchString(lines[idx]) {
		// Insert heading at top
		newHead := "# " + title
		// Prepend preserving any leading comments (none typical for md)
		lines = append([]string{newHead, ""}, lines...)
		changed = true
		idx = 0
	}

	// Ensure metadata block right after heading
	// Find position immediately after heading (skip blank line we ensured)
	metaInsertPos := 1
	// Scan existing for Last Updated & Status lines within first 10 lines
	var foundUpdated, foundStatus int = -1, -1
	for i := 0; i < len(lines) && i < 15; i++ {
		if lastUpdatedRegex.MatchString(lines[i]) {
			foundUpdated = i
		}
		if statusRegex.MatchString(lines[i]) {
			foundStatus = i
		}
	}
	// Update / insert Last Updated
	if foundUpdated >= 0 {
		if !strings.Contains(lines[foundUpdated], date) { // update date
			lines[foundUpdated] = fmt.Sprintf("> Last Updated: %s", date)
			changed = true
		}
	} else {
		// Insert after heading
		insertLine(&lines, metaInsertPos, fmt.Sprintf("> Last Updated: %s", date))
		insertLine(&lines, metaInsertPos+1, "> Status: Active")
		changed = true
	}
	if foundUpdated >= 0 && foundStatus == -1 { // updated existed but no status
		// Insert status immediately after last updated line
		insertLine(&lines, foundUpdated+1, "> Status: Active")
		changed = true
	} else if foundStatus >= 0 {
		// Potentially update status if deprecated keyword present
		if containsDeprecated(lines) && !strings.Contains(strings.ToLower(lines[foundStatus]), "deprecated") {
			lines[foundStatus] = "> Status: Deprecated"
			changed = true
		}
	}

	// List normalization & trailing space trim (excluding fenced blocks)
	for i, line := range lines {
		l := line
		// Fence detection: support ``` or ```` at line start
		if strings.HasPrefix(l, "```") || strings.HasPrefix(l, "````") {
			if !inFence { // entering
				inFence = true
				if strings.HasPrefix(l, "````") {
					fenceDelimiter = "````"
				} else {
					fenceDelimiter = "```"
				}
			} else if strings.HasPrefix(l, fenceDelimiter) { // exiting
				inFence = false
				fenceDelimiter = ""
			}
		}
		if !inFence {
			// list markers normalization
			if strings.HasPrefix(l, "* ") || strings.HasPrefix(l, "+ ") {
				lines[i] = "- " + strings.TrimSpace(l[2:])
				changed = true
			} else if strings.HasPrefix(l, "  * ") || strings.HasPrefix(l, "  + ") {
				// nested list minimal
				prefix := "  - "
				lines[i] = prefix + strings.TrimSpace(l[4:])
				changed = true
			}
			// Trim trailing spaces
			trimmed := strings.TrimRight(lines[i], " \t")
			if trimmed != lines[i] {
				lines[i] = trimmed
				changed = true
			}
		}
	}

	// Footer addition
	if !containsFooter(lines) && len(lines) > 30 {
		lines = append(lines, "", "---", footerMarker)
		changed = true
	}

	// Ensure single blank line after heading
	if len(lines) > 1 && lines[1] != "" {
		lines = append(lines[:1], append([]string{""}, lines[1:]...)...)
		changed = true
	}

	if !changed {
		return nil
	}
	// Ensure final newline
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	// Write back
	return writeFileIfDifferent(path, []byte(content), original, modified)
}

func splitLines(s string) []string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func firstNonEmpty(lines []string) int {
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			return i
		}
	}
	return -1
}

func insertLine(lines *[]string, idx int, val string) {
	if idx < 0 {
		idx = 0
	}
	l := *lines
	if idx >= len(l) {
		*lines = append(l, val)
		return
	}
	*lines = append(l[:idx], append([]string{val}, l[idx:]...)...)
}

func containsFooter(lines []string) bool {
	for _, l := range lines {
		if strings.Contains(l, footerMarker) {
			return true
		}
	}
	return false
}

func containsDeprecated(lines []string) bool {
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), "deprecated") {
			return true
		}
	}
	return false
}

func writeFileIfDifferent(path string, newContent, oldContent []byte, modified *[]string) error {
	if bytes.Equal(newContent, oldContent) {
		return nil
	}
	if err := os.WriteFile(path, newContent, 0o644); err != nil {
		return err
	}
	*modified = append(*modified, path)
	return nil
}
