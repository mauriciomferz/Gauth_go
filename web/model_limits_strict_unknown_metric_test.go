package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	pm "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestModelLimitsStrictUnknownMetric ensures the dedicated unknown model counter increments when strict mode denies a request.
func TestModelLimitsStrictUnknownMetric(t *testing.T) {
	os.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")
	defer os.Unsetenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN")
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	// Swap in Prometheus adapter so counter is exposed via exposition endpoint.
	bs.metrics = pm.NewPrometheusMetrics(pm.PrometheusAdapterOptions{})

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/model/validate", bytes.NewBufferString(`{"model_id":"nonexistent","input_tokens":10}`))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "model_unknown") {
		t.Fatalf("expected strict unknown rejection got code=%d body=%s", rec.Code, rec.Body.String())
	}

	// Scrape generic Prometheus metrics endpoint and assert counter presence.
	recM := httptest.NewRecorder()
	reqM, _ := http.NewRequest("GET", "/api/v1/beta/metrics/prometheus", nil)
	bs.router.ServeHTTP(recM, reqM)
	if recM.Code != 200 {
		// Do not hard fail; but this should normally be present when Prometheus adapter wired.
		// For robustness still fail to ensure visibility in CI.
		t.Fatalf("prometheus metrics endpoint code=%d body=%s", recM.Code, recM.Body.String())
	}
	body := recM.Body.String()
	if !strings.Contains(body, "gauth_rfc0111_model_unknown_total") {
		// Provide small snippet for debugging.
		if len(body) > 1000 {
			body = body[:1000]
		}
		t.Fatalf("missing model_unknown_total counter in metrics output snippet=%s", body)
	}
}
