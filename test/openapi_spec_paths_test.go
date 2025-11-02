package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAPISpecPaths(t *testing.T) {
	wd, _ := os.Getwd()
	// Candidate paths relative to current working dir (which may be module root or /test)
	candidates := []string{
		filepath.Join(wd, "docs", "openapi.yaml"),                            // if wd=root
		filepath.Join(wd, "..", "docs", "openapi.yaml"),                     // if wd=/test
		filepath.Join(wd, "..", "..", "docs", "openapi.yaml"),              // fallback
	}
	var path string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			path = c
			break
		}
	}
	if path == "" {
		t.Fatalf("could not locate openapi.yaml in candidate paths: %v", candidates)
	}
	b, err := os.ReadFile(path)
	if err != nil { t.Fatalf("read openapi spec: %v", err) }
	content := string(b)
	mustContain := []string{
		"/api/v1/crypto/bls/pop/verify:",
		"/api/v1/crypto/bls/aggregate:",
		"/api/v1/crypto/algorithms:",
		"/api/v1/anchor/chain:",
		"/api/v1/anchor/emitCombined:",
		"/api/v1/anchor/verifyChain:",
		"version: 0.3.2-beta",
		"CombinedAnchorToken:",
		"ErrorResponse:",
	}
	for _, s := range mustContain {
		if !strings.Contains(content, s) {
			t.Fatalf("openapi.yaml missing expected substring: %s", s)
		}
	}
}
