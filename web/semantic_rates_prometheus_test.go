package web

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestSemanticRatesPrometheus ensures semantic rate metrics are exposed in Prometheus format.
func TestSemanticRatesPrometheus(t *testing.T) {
	os.Setenv("GAUTH_SEMANTIC_PERSIST_PATH", "") // disable persistence interference
	srv := NewBetaServer("8123")
	if srv.rfc0111Service == nil {
		t.Fatalf("RFC0111 service not initialized; GAUTH_DISABLE_RFC0111_SERVICE should not be set")
	}
	// Trigger several semantic failures to create deltas for rate computation.
	// We simulate by invoking authorize endpoint with payloads causing scope_violation and amount_limit_exceeded.
	// Minimal JSON bodies that the handler will parse; rely on validation logic increments (existing tests cover correctness).
	badBodies := []string{
		`{"delegation":{"scope":["read"],"requested_scope":["admin"],"amount_limit":10,"requested_amount":50}}`,
		`{"delegation":{"scope":["pay"],"requested_scope":["pay"],"amount_limit":1,"requested_amount":5}}`,
		`{"delegation":{"scope":["x"],"requested_scope":["y"],"amount_limit":2,"requested_amount":3}}`,
	}
	for _, body := range badBodies {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/poa/authorize", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
	}
	// Hit semantic JSON endpoint to record a history snapshot (throttled to >=1s; force sleep if needed)
	wj := httptest.NewRecorder()
	rj := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics", nil)
	srv.router.ServeHTTP(wj, rj)
	// Second pass to create a delta
	for _, body := range badBodies {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/poa/authorize", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
	}
	wj2 := httptest.NewRecorder()
	rj2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics", nil)
	srv.router.ServeHTTP(wj2, rj2)
	// Prometheus endpoint output
	wp := httptest.NewRecorder()
	rp := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/prometheus", nil)
	srv.router.ServeHTTP(wp, rp)
	out := wp.Body.String()
	// Expect rate metric headers and at least one category rate line
	if !strings.Contains(out, "gauth_poa_semantic_rate_60s") {
		t.Fatalf("expected gauth_poa_semantic_rate_60s metrics in output")
	}
	if !strings.Contains(out, "gauth_poa_semantic_rate_300s") {
		t.Fatalf("expected gauth_poa_semantic_rate_300s metrics in output")
	}
}
