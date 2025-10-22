package main

// openapi_guard: minimal structural guard for docs/openapi.yaml.
// Avoids mixing packages in scripts/ by isolating in subdirectory with its own main.
// Exit codes:
//   0 success
//   2 guard failure (missing markers)
//   3 other error

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var errGuard = errors.New("openapi spec guard failure")

func main() {
	if err := run(); err != nil {
		var exit int
		if errors.Is(err, errGuard) {
			exit = 2
		} else {
			exit = 3
		}
		fmt.Fprintf(os.Stderr, "openapi_guard: %v\n", err)
		os.Exit(exit)
	}
}

func run() error {
	paths := []string{"docs/openapi.yaml", "./docs/openapi.yaml", "../docs/openapi.yaml"}
	var content string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err == nil {
			content = string(b)
			break
		}
	}
	if content == "" {
		return fmt.Errorf("%w: spec not found", errGuard)
	}
	hasOpenAPI := false
	hasPaths := false
	s := bufio.NewScanner(strings.NewReader(content))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "openapi:") {
			hasOpenAPI = true
		}
		if line == "paths:" || strings.HasPrefix(line, "paths:") {
			hasPaths = true
		}
		if hasOpenAPI && hasPaths {
			break
		}
	}
	if !hasOpenAPI || !hasPaths {
		return fmt.Errorf("%w: missing openapi:%v paths:%v", errGuard, hasOpenAPI, hasPaths)
	}
	fmt.Println("✅ openapi guard passed")
	return nil
}
