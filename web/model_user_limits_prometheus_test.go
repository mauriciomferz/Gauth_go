package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	pm "github.com/mauriciomferz/AgentAuth/internal/metrics"
)

// TestModelUserLimitsPrometheusExposition ensures per-user exceed counters are exposed after triggering violations.
func TestModelUserLimitsPrometheusExposition(t *testing.T) {
	// Prepare limits with per-user overrides to force exceeds.
	f, err := os.CreateTemp(t.TempDir(), "model_user_prom_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":200,"max_output_tokens":150,"max_requests_per_minute":10}},"user_limits":{"demo-model":{"alice":{"max_input_tokens":100,"max_output_tokens":80,"max_requests_per_minute":1}}}}`
	if _, err := f.Write([]byte(limitsJSON)); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	// Set path env before server init so loader picks it up.
	// Use short rate limit for alice to guarantee a rate exceed quickly.
	t.Setenv("GAUTH_MODEL_LIMITS_CONFIG_PATH", f.Name())
	// Enable Prometheus exposition endpoint (already active by default on /api/v1/beta/metrics/prometheus)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Replace memory metrics with Prometheus adapter to ensure counters are exposed via exposition endpoints.
	regMetrics := pm.NewPrometheusMetrics(pm.PrometheusAdapterOptions{})
	srv.metrics = regMetrics
	// Trigger user input exceed (expects 400)
	doMV(t, srv, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 120, "output_tokens": 10})
	// Trigger user output exceed (expects 400)
	doMV(t, srv, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 10, "output_tokens": 120})
	// Trigger user rate exceed (limit=1: first ok, second 429)
	doMV(t, srv, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 10, "output_tokens": 10})
	doMV(t, srv, map[string]any{"model_id": "demo-model", "user_id": "alice", "input_tokens": 10, "output_tokens": 10})
	// Fetch metrics exposition
	// Use new generic Prometheus exposition endpoint
	w := performRequest(srv.router, "GET", "/api/v1/beta/metrics/prometheus")
	if w.Code != 200 {
		t.Fatalf("metrics exposition status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	want := []string{
		"gauth_aap001_model_user_input_limit_exceeded_total",
		"gauth_aap001_model_user_output_limit_exceeded_total",
		"gauth_aap001_model_user_rate_limit_exceeded_total",
	}
	for _, m := range want {
		if !regexp.MustCompile(m + " ").MatchString(body) {
			// Body includes many metrics; print truncated prefix for debug
			if len(body) > 800 {
				body = body[:800]
			}
			t.Fatalf("expected metric %s in exposition body prefix:\n%s", m, body)
		}
	}
}

func doMV(t *testing.T, srv *BetaServer, body map[string]any) {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 && w.Code != 400 && w.Code != 429 {
		t.Fatalf("unexpected status %d body=%s", w.Code, w.Body.String())
	}
}
