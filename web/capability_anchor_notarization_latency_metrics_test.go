package web

import (
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestCapabilityAnchorNotarizationLatencyHistogram ensures the latency histogram line appears in custom exposition
// when GAUTH_CAP_ANCHOR_NOTARIZE=1 and at least one observation is recorded.
func TestCapabilityAnchorNotarizationLatencyHistogram(t *testing.T) {
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	// We don't rely on emission path; directly observe a latency via adapter method.
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth"})
	srv := NewBetaServer(":0")
	srv.metrics = pm
	// Simulate latency observation (e.g., 12ms) using type assertion.
	if obs, ok := srv.metrics.(interface{ ObserveCapabilityAnchorNotarizationLatency(time.Duration) }); ok {
		obs.ObserveCapabilityAnchorNotarizationLatency(12 * time.Millisecond)
	} else {
		t.Fatalf("prometheus adapter missing latency observer")
	}
	// Exposition endpoint should include HELP/TYPE and at least one bucket or advisory note line.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("metrics endpoint code=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !regexp.MustCompile(`capability_anchor_notarization_latency_seconds`).MatchString(body) {
		t.Fatalf("expected latency histogram identifier in body:\n%s", body)
	}
}
