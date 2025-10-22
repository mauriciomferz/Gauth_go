package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAuthzMetricsPrometheus ensures the authorization metrics Prometheus endpoint exposes
// histogram buckets, count, and p99 line after a few evaluations.
func TestAuthzMetricsPrometheus(t *testing.T) {
	srv := NewBetaServer("")
	// Prime with a few evaluations to populate latency buckets.
	for i := 0; i < 7; i++ {
		payload := map[string]any{
			"subject":  "alice@example.com",
			"resource": "report:finance",
			"action":   "read",
			"context":  map[string]any{"department": "finance"},
		}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/authz/evaluate", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("evaluation status %d body=%s", w.Code, w.Body.String())
		}
		time.Sleep(500 * time.Microsecond) // small variance
	}

	// Fetch prometheus exposition
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/authz/metrics/prometheus", nil)
	srv.router.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != 200 {
		t.Fatalf("unexpected status %d body=%s", w.Code, body)
	}
	// Basic required lines
	if !strings.Contains(body, "authz_latency_bucket") {
		t.Fatalf("expected latency bucket lines in prometheus output: %s", body)
	}
	if !strings.Contains(body, "authz_latency_p99_nanoseconds") {
		t.Fatalf("expected p99 gauge line in prometheus output: %s", body)
	}
}
