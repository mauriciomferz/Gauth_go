//go:build ignore

package main

// openapi_guard.go
// Lightweight guard that enforces the presence and minimal structure of docs/openapi.yaml.
// This is a placeholder toward a richer semantic contract (route parity, component schema ownership, error shape enforcement).
//
// Exit Codes:
//   0 - spec present & minimally valid (has 'openapi:' + 'paths:')
//   2 - spec missing or minimal markers absent
//   3 - other error (IO etc.)
//
// Usage: go run ./scripts/openapi_guard.go
// Integrate in CI (pre-merge) to prevent accidental spec removal or truncation.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func main() {
    if err := run(); err != nil {
        var exit int
        if errors.Is(err, errGuard) { exit = 2 } else { exit = 3 }
        fmt.Fprintf(os.Stderr, "openapi_guard: %v\n", err)
        os.Exit(exit)
    }
}

var errGuard = errors.New("openapi spec guard failure")

func run() error {
    paths := []string{"docs/openapi.yaml", "./docs/openapi.yaml", "../docs/openapi.yaml"}
    var content string
    for _, p := range paths {
        b, err := os.ReadFile(p)
        if err == nil { content = string(b); break }
    }
    if content == "" { return fmt.Errorf("%w: spec file not found in docs/", errGuard) }
    hasOpenAPI := false
    hasPaths := false
    s := bufio.NewScanner(strings.NewReader(content))
    for s.Scan() {
        line := strings.TrimSpace(s.Text())
        if strings.HasPrefix(line, "openapi:") { hasOpenAPI = true }
        if line == "paths:" || strings.HasPrefix(line, "paths:") { hasPaths = true }
        if hasOpenAPI && hasPaths { break }
    }
    if !hasOpenAPI || !hasPaths {
        return fmt.Errorf("%w: missing required top-level keys (openapi:%v paths:%v)", errGuard, hasOpenAPI, hasPaths)
    }
    // Future: parse & validate semantic coverage ratios.
    fmt.Println("✅ openapi guard passed (minimal structure present)")
    return nil
}
