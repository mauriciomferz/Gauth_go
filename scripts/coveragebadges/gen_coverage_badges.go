// Command gen_coverage_badges generates per-package coverage SVG badges.
// Usage: go run ./scripts/coveragebadges/gen_coverage_badges.go [outputDir]
// If outputDir not supplied it defaults to build/badges.
// This is isolated in its own directory to avoid multiple main() collisions.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Matches the final percentage on the aggregated total line from `go tool cover -func` output.
// Example line: "total:\t(statements)\t87.5%" (tabs/spaces vary).
var reTotal = regexp.MustCompile(`^total:.*?([0-9]+\.?[0-9]*)%$`)

func main() {
	flag.Parse()
	outDir := flag.Arg(0)
	if outDir == "" {
		outDir = filepath.Join("build", "badges")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("create output dir: %v", err)
	}
	pkgs, err := listPackages()
	if err != nil {
		fatalf("list packages: %v", err)
	}
	sort.Strings(pkgs)
	overall := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		cov, err := coverageFor(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: coverage for %s failed: %v\n", p, err)
			continue
		}
		badgePath := filepath.Join(outDir, pkgFileName(p)+".svg")
		if err := os.WriteFile(badgePath, []byte(renderBadge(p, cov)), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: write badge %s: %v\n", badgePath, err)
			continue
		}
		overall = append(overall, fmt.Sprintf("%s=%.1f", p, cov))
		fmt.Printf("badge: %s %.1f%%\n", p, cov)
	}
	idx := filepath.Join(outDir, "index.txt")
	_ = os.WriteFile(idx, []byte(strings.Join(overall, "\n")), 0o600)
}

func listPackages() ([]string, error) {
	// Use import paths, then convert to relative ./subdir form for local testing.
	modPath, _ := modulePath()
	cmd := exec.Command("go", "list", "./...")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pkgs []string
	for _, imp := range lines {
		if strings.Contains(imp, "/vendor/") {
			continue
		}
		if strings.Contains(imp, "backup_") {
			continue
		}
		// Derive relative path
		rel := imp
		if modPath != "" && strings.HasPrefix(imp, modPath) {
			rel = strings.TrimPrefix(imp, modPath)
			rel = strings.TrimPrefix(rel, "/")
		}
		if rel == "" { // root module
			rel = "."
		} else {
			rel = "./" + rel
		}
		// Heuristics to skip binaries/example noise
		if strings.HasPrefix(rel, "./cmd") || strings.HasPrefix(rel, "./examples") || strings.HasPrefix(rel, "./scripts") || strings.HasPrefix(rel, "./test") {
			continue
		}
		pkgs = append(pkgs, rel)
	}
	return pkgs, nil
}

func modulePath() (string, error) {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", scanner.Err()
}

func coverageFor(pkg string) (float64, error) {
	prof := filepath.Join(os.TempDir(), fmt.Sprintf("cov_%d.out", time.Now().UnixNano()))
	// Flags must precede package import path.
	cmd := exec.Command("go", "test", "-coverprofile", prof, "-covermode=atomic", pkg)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "[no test files]") {
			return 0, nil
		}
		return 0, fmt.Errorf("test run failed: %v output=%s", err, string(out))
	}
	tool := exec.Command("go", "tool", "cover", "-func", prof)
	funcOut, err := tool.Output()
	if err != nil {
		return 0, fmt.Errorf("cover tool: %v", err)
	}
	var cov float64
	for _, line := range strings.Split(string(funcOut), "\n") {
		line = strings.TrimSpace(line)
		if m := reTotal.FindStringSubmatch(line); m != nil {
			if _, err := fmt.Sscanf(m[1], "%f", &cov); err != nil {
				// Log error but continue with cov = 0
				fmt.Printf("Warning: failed to parse coverage value %q: %v\n", m[1], err)
			}
			break
		}
	}
	_ = os.Remove(prof)
	return cov, nil
}

func renderBadge(pkg string, cov float64) string {
	color := colorFor(cov)
	label := escape(pkg)
	value := fmt.Sprintf("%.1f%%", cov)
	wLabel := 10 + len(label)*6
	wValue := 10 + len(value)*6
	totalWidth := wLabel + wValue
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
  <linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>
  <clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="%s"/>
    <rect width="%d" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="15">%s</text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="15">%s</text>
  </g>
</svg>`, totalWidth, label, value, totalWidth, wLabel, wLabel, wValue, color, totalWidth, wLabel/2, label, wLabel/2, label, wLabel+wValue/2, value, wLabel+wValue/2, value)
}

func colorFor(cov float64) string {
	switch {
	case cov >= 90:
		return "#4c1"
	case cov >= 80:
		return "#97CA00"
	case cov >= 70:
		return "#a4a61d"
	case cov >= 60:
		return "#dfb317"
	case cov >= 50:
		return "#fe7d37"
	default:
		return "#e05d44"
	}
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}

func pkgFileName(p string) string {
	// Use last path element for brevity
	parts := strings.Split(p, "/")
	last := parts[len(parts)-1]
	last = strings.ReplaceAll(last, " ", "_")
	return last
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
