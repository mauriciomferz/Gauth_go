//go:build ignore

package main

// ui_smoke_test.go - lightweight UI smoke test
// Run with: go test -run TestUISmoke ./scripts
// Purpose: Ensures critical interactive elements exist in embedded index.html before deployment.
// This is a fast HTML presence check, not a full integration test.

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// fetchRoot performs a GET to the provided URL (defaults to http://localhost:8080/index.html) and returns body.
func fetchIndexHTML(t *testing.T) string {
	url := os.Getenv("GAUTH_UI_URL")
	if url == "" {
		url = "http://localhost:8080/index.html"
	}
	resp, err := http.Get(url)
	if err != nil {
		// If connection refused, provide guidance.
		if strings.Contains(err.Error(), "connection refused") {
			t.Skipf("server not running (%v) - skipping UI smoke test", err)
			return ""
		}
		t.Fatalf("failed GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status %d from %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// TestUISmoke verifies presence of core interactive elements and basic ARIA attributes.
func TestUISmoke(t *testing.T) {
	body := fetchIndexHTML(t)
	checks := map[string]string{
		"theme toggle button":     "id=\"themeToggle\"",
		"mobile nav button":       "id=\"mobileNavButton\"",
		"tablist role":            "role=\"tablist\"",
		"first tab id":            "id=\"tab-token-demo\"",
		"first tab aria-selected": "id=\"tab-token-demo\" aria-selected=\"true\"",
	}
	for label, token := range checks {
		if !strings.Contains(body, token) {
			t.Errorf("missing %s (%s)", label, token)
		}
	}
	// Ensure at least 5 tabs present
	if strings.Count(body, "data-tab=\"") < 5 {
		t.Errorf("expected >=5 tabs, found %d", strings.Count(body, "data-tab=\""))
	}
}
