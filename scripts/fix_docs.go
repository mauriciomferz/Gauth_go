package scripts

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// 1. Run validation script to get list of issues
	cmd := exec.Command("bash", "scripts/docs_index.sh", "--validate")
	output, err := cmd.CombinedOutput()
	// Process output even if it failed (exit code 1)
	if err != nil && len(output) == 0 {
		fmt.Printf("Error running validation: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		issueType := parts[0]
		filePath := parts[1]

		// Only handle files that exist
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		if issueType == "MISSING_HEADER" {
			fixMissingHeader(filePath)
			count++
		} else if issueType == "INCOMPLETE" && len(parts) >= 3 {
			missingFields := parts[2]
			fixIncompleteHeader(filePath, missingFields)
			count++
		}
	}
	fmt.Printf("Fixed %d files\n", count)
}

func fixMissingHeader(path string) {
	fmt.Printf("Fixing MISSING header: %s\n", filepath.Base(path))
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	filename := filepath.Base(path)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.Title(strings.ToLower(title))

	header := fmt.Sprintf(`---
title: %s
category: Uncategorized
status: draft
lastUpdated: %s
owners: [system]
---

`, title, time.Now().Format("2006-01-02"))

	newContent := append([]byte(header), content...)
	ioutil.WriteFile(path, newContent, 0644)
}

func fixIncompleteHeader(path string, missingFields string) {
	fmt.Printf("Fixing INCOMPLETE header: %s (missing: %s)\n", filepath.Base(path), missingFields)
	// Simple append strategy for incomplete headers is risky without parsing YAML.
	// For now, let's just log it. The user has many MISSING headers which is the blocker.
	// Implementing robust YAML injection here is complex.
	// If the file starts with ---, we could try to inject, but let's stick to missing first.
	// Wait, actually, let's try to append if it's simple valid YAML.

	// Read file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[0] == "---" {
		// Find end of block
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				endIdx = i
				break
			}
		}

		if endIdx > 0 {
			// Inject fields before the closing ---
			missing := strings.Fields(missingFields)
			var injection []string
			for _, field := range missing {
				val := "unknown"
				if field == "lastUpdated" {
					val = time.Now().Format("2006-01-02")
				}
				if field == "owners" {
					val = "[system]"
				}
				if field == "status" {
					val = "draft"
				}
				if field == "category" {
					val = "Uncategorized"
				}
				if field == "title" {
					t := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
					val = strings.ReplaceAll(t, "_", " ")
				}
				injection = append(injection, fmt.Sprintf("%s: %s", field, val))
			}

			// Insert
			newLines := append(lines[:endIdx], append(injection, lines[endIdx:]...)...)
			newContent := strings.Join(newLines, "\n")
			ioutil.WriteFile(path, []byte(newContent), 0644)
			return
		}
	}
}
