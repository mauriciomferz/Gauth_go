package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestViolationMetricsPrometheus ensures Prometheus exposition contains expected metric lines.
func TestViolationMetricsPrometheus(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Trigger some failures to get non-zero counters.
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/token/validate", nil)
		srv.router.ServeHTTP(w, req)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/prometheus", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	body := w.Body.String()
	// Core header metrics
	if !strings.Contains(body, "AGENTAUTH_validation_total ") {
		t.Fatalf("missing AGENTAUTH_validation_total line")
	}
	// Category lines
	cats := []string{"sig_invalid", "expired", "not_yet_valid", "issuer_mismatch", "replay_detected", "audience_mismatch", "missing_claim", "unknown"}
	for _, c := range cats {
		line := "AGENTAUTH_validation_" + c + "_total"
		if !strings.Contains(body, line) {
			t.Fatalf("missing metric line %s", line)
		}
	}
}
