package jurisdiction

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestPrometheusMetricsEndpoint ensures Prometheus exposition contains expected metrics after an enforcement.
func TestPrometheusMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := NewEnforcementEngine()
	integration := &ServerIntegration{engine: eng}
	// Perform a sample enforcement to populate latency + counters.
	claims := map[string]interface{}{"jurisdiction": "UNITED_STATES"}
	_, err := integration.EnforceJurisdiction(context.Background(), "alice", "resource:x", "transfer", claims)
	if err != nil { t.Fatalf("enforce: %v", err) }

	h := NewAPIHandler(integration)
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/jurisdiction/metrics/prometheus", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	for _, metric := range []string{
		"gauth_jurisdiction_enforcements_total",
		"gauth_jurisdiction_enforcements_allowed_total",
		"gauth_jurisdiction_average_latency_ms",
	} {
		if !strings.Contains(body, metric) {
			 t.Fatalf("expected metric %s in body", metric)
		}
	}
	// Ensure no empty content
	if len(body) == 0 { t.Fatalf("empty metrics body") }
	// Latency should be >=0; enforcement sets EMA first sample.
	if !strings.Contains(body, "gauth_jurisdiction_average_latency_ms") {
		 t.Fatalf("latency metric missing")
	}
	// Clear env for isolation (if tests set external file)
	os.Unsetenv("GAUTH_JURISDICTION_RULES_PATH")
}

// testContext provides a simple background context.
// (helper removed; direct context.Background() usage)
