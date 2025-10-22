package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestPrometheusAdapterBasic verifies counters increment & exposition contains expected names.
func TestPrometheusAdapterBasic(t *testing.T) {
	reg := prom.NewRegistry()
	pm := NewPrometheusMetrics(PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "test", Registry: reg})
	pm.IncDelegationsCreated()
	pm.IncDelegationsCreated()
	pm.IncSignaturesIssued()
	pm.IncSignatureIssueFailures()
	pm.IncSignatureVerifications()
	pm.IncSignatureVerificationFailures()
	pm.IncRevocationIntegrityFailures()
	pm.IncSignaturePublicKeyMissing()
	pm.ObserveValidationLatency(1234) // nanoseconds interpreted later as seconds; fine for test

	// Scrape
	rr := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body := rr.Body.String()
	// Ensure a few metric names exist
	expected := []string{
		"gauth_test_delegations_created_total",
		"gauth_test_signatures_issued_total",
		"gauth_test_signature_issue_failures_total",
		"gauth_test_validation_latency_seconds_bucket", // histogram family
		"gauth_test_signature_public_key_missing_total",
	}
	for _, name := range expected {
		if !strings.Contains(body, name) {
			t.Fatalf("expected to find metric %s in exposition:\n%s", name, body)
		}
	}
}
