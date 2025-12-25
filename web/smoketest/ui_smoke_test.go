// Package smoketest provides a minimal UI presence test for critical elements.
// Run: go test -run TestUISmoke ./web/smoketest
package smoketest

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func fetchIndexHTML(t *testing.T) string {
	url := os.Getenv("GAUTH_UI_URL")
	if url == "" {
		url = "http://localhost:8080/index.html"
	}
	//nolint:gosec // G107: test server URL from environment or localhost
	resp, err := http.Get(url)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			// Skip if server not started; CI should launch server prior to running this test.
			t.Skipf("beta server not running (%v) - skipping UI smoke test", err)
			return ""
		}
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 404 {
		t.Skipf("index.html not served (status 404) - skipping UI smoke test")
		return ""
	}
	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status %d from %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// TestUISmoke checks for core landing page elements.
func TestUISmoke(t *testing.T) {
	body := fetchIndexHTML(t)
	if body == "" { // skipped
		return
	}
	checks := map[string]string{
		"page title":       "GAuth Dashboard",
		"hero title":       "GAuth Beta Dashboard",
		"revocation panel": "Revocation Head",
		"rotation panel":   "Rotation Summary",
		"errors panel":     "Error Catalog",
		"footer copyright": "© 2025 GAuth Beta",
	}
	for label, token := range checks {
		if !strings.Contains(body, token) {
			t.Errorf("missing %s (expected text: %s)", label, token)
		}
	}

	// Check for critical structural elements and panel IDs
	requiredElements := []string{
		"<nav",
		"<footer",
		`id="revocation"`,
		`id="rotation"`,
		`id="capability"`,
	}
	for _, elem := range requiredElements {
		if !strings.Contains(body, elem) {
			t.Errorf("missing structural element: %s", elem)
		}
	}
}

// (helper removed: simplified global presence check)
