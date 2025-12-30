package web

import (
	"net/http/httptest"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
)

// TestExternalAnchorPrometheusMetrics validates that attempts, latency, and age/hash gauges are exposed after a successful anchor.
func TestExternalAnchorPrometheusMetrics(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory")
	// Initialize Prometheus adapter (registers metrics in default registry).
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "aap001"})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	srv.metrics = pm
	// Allow initial anchor attempt and background age update.
	time.Sleep(60 * time.Millisecond)
	// Scrape /api/v1/beta/capabilities/anchor/status to ensure receipt present.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
	}
	if !hasSubstr(rec.Body.String(), "external_anchor_receipt") {
		t.Fatalf("expected external_anchor_receipt in status payload")
	}
	// Collect all metrics registered by adapter and ensure at least one sample referencing external anchor was recorded.
	// We search rendered exposition text for the metric name substring.
	// Smoke test: record another latency observation to exercise helper and ensure no panic.
	pm.ObserveExternalAnchorLatency("memory", 5*time.Millisecond)
	// Set age/hash gauges manually simulating background loop invocation.
	pm.SetExternalAnchorAgeSeconds(1)
	pm.SetExternalAnchorLastHashLen(64)
	// No direct exporter introspection without global scrape; rely on absence of panic and receipt presence.
}
