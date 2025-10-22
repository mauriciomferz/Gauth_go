package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// minimal struct to decode generic JSON
type generic map[string]interface{}

func setupTestServer() *BetaServer { return NewBetaServer(":0") }

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRootServesHTML(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected html content type, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), "GAuth Beta Demo") {
		t.Fatalf("expected body to contain title text")
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/api/v1/beta/health")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body generic
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if success, ok := body["success"].(bool); !ok || !success {
		t.Fatalf("expected success true")
	}
}

func TestInfoEndpoint(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/api/v1/beta/info")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body generic
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["disclaimer"] == "" {
		t.Fatalf("expected disclaimer present")
	}
	if _, ok := body["features"].([]interface{}); !ok {
		t.Fatalf("expected features array")
	}
}

func TestPingEndpoint(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/api/v1/beta/ping")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body generic
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if pong, ok := body["pong"].(bool); !ok || !pong {
		t.Fatalf("expected pong true")
	}
}

func TestCSPHeaderPresent(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/")
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatalf("expected CSP header")
	}
	if !strings.Contains(csp, "default-src 'self'") {
		t.Fatalf("missing default-src in CSP")
	}
}

func TestPOAMetrics(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/api/v1/poa/metrics")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestPOAAuthorizeError(t *testing.T) {
	srv := setupTestServer()
	// Missing required fields should trigger 400
	req := httptest.NewRequest(http.MethodPost, "/api/v1/poa/authorize", strings.NewReader(`{"client_id":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("expected failure for invalid request")
	}
}

func TestBetaVersionHeader(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/api/v1/beta/health")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if hv := w.Header().Get("X-API-Version"); hv != "beta" {
		t.Fatalf("expected X-API-Version=beta, got %q", hv)
	}
}

func TestEducationalDeprecationHeaders(t *testing.T) {
	srv := setupTestServer()
	w := performRequest(srv.router, http.MethodGet, "/api/v1/beta/health")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if d := w.Header().Get("X-API-Deprecated"); d != "true" {
		t.Fatalf("expected X-API-Deprecated=true, got %q", d)
	}
	if rep := w.Header().Get("X-API-Replacement"); rep != "/api/v1/beta" {
		t.Fatalf("expected X-API-Replacement=/api/v1/beta, got %q", rep)
	}
}
