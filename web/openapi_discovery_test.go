package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenAPIDiscoveryEndpoints ensures the discovery document exposes openapi urls and spec endpoints serve content.
func TestOpenAPIDiscoveryEndpoints(t *testing.T) {
	bs := NewTestServerNoSeed(t)
	// Discovery
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	bs.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("discovery status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "openapi_url") {
		t.Fatalf("expected openapi_url in discovery: %s", body)
	}
	if !strings.Contains(body, "/openapi.yaml") {
		t.Fatalf("expected /openapi.yaml reference in discovery: %s", body)
	}

	// YAML spec
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/openapi.yaml", nil)
	bs.router.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("openapi.yaml status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), "openapi: 3.1.0") {
		t.Fatalf("expected openapi version header in yaml: %s", rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), "/api/v1/token/create") {
		t.Fatalf("expected token create path in yaml")
	}
	if !strings.Contains(rr2.Body.String(), "/api/v1/token/revoke") {
		t.Fatalf("expected token revoke path in yaml")
	}

	// JSON spec
	rr3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/v1/openapi", nil)
	bs.router.ServeHTTP(rr3, req3)
	if rr3.Code != 200 {
		t.Fatalf("openapi json status=%d", rr3.Code)
	}
	if !strings.Contains(rr3.Body.String(), "\"paths\"") {
		t.Fatalf("expected paths object in json spec: %s", rr3.Body.String())
	}
}

// TestOpenAPISpecCriticalPaths ensures all critical paths appear in YAML.
func TestOpenAPISpecCriticalPaths(t *testing.T) {
	bs := NewTestServerNoSeed(t)
	// Fetch YAML
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/openapi.yaml", nil)
	bs.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("openapi.yaml status=%d", rr.Code)
	}
	y := rr.Body.String()
	critical := []string{
		"/api/v1/token/create",
		"/api/v1/token/validate",
		"/api/v1/token/revoke",
		"/api/v1/token/status/update",
		"/api/v1/beta/policy/evaluate",
		"/api/v1/audit/record",
	}
	for _, p := range critical {
		if !strings.Contains(y, p) {
			t.Fatalf("missing critical path %s in spec", p)
		}
	}
}
