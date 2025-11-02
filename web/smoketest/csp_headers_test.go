// Package smoketest contains additional header validation tests (CSP).
package smoketest

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestCSPHeaders validates that /index.html responds with a CSP header containing a script nonce and frame-ancestors 'none'
func TestCSPHeaders(t *testing.T) {
	if os.Getenv("GAUTH_SKIP_SMOKETEST") == "1" {
		t.Skip("GAUTH_SKIP_SMOKETEST=1")
	}
	resp, err := http.Get("http://localhost:8080/index.html")
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			t.Skipf("server not running - skipping CSP header test: %v", err)
			return
		}
		t.Fatalf("failed to GET index.html: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("index.html not served (skipping CSP header test)")
		return
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatalf("missing Content-Security-Policy header")
	}
	// Expect script-src containing a nonce attribute like 'nonce-<base64url>'
	// Example: script-src 'self' 'nonce-XYZ' https://cdn...
	nonceRe := regexp.MustCompile(`script-src[^;]*'nonce-[A-Za-z0-9_-]+'`)
	if !nonceRe.MatchString(csp) {
		t.Fatalf("CSP missing script-src nonce: %s", csp)
	}
	if !regexp.MustCompile(`frame-ancestors 'none'`).MatchString(csp) {
		t.Fatalf("CSP missing frame-ancestors 'none': %s", csp)
	}
}
