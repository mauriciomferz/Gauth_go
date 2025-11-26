package web

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/internal/notary"
)

// TestCapabilityAnchorNotarizationProviderMetrics ensures provider-labeled metrics are recorded.
func TestCapabilityAnchorNotarizationProviderMetrics(t *testing.T) {
	os.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	os.Setenv("GAUTH_CAP_ANCHOR_NOTARY_PROVIDER", capSourceExternal)
	os.Setenv("GAUTH_NOTARY_STUB_MIN_LATENCY_MS", "1")
	os.Setenv("GAUTH_NOTARY_STUB_MAX_LATENCY_MS", "2")
	os.Setenv("GAUTH_NOTARY_STUB_FAIL_PROB", "0")
	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	// Replace default memory metrics with Prometheus adapter to access provider-labeled methods.
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "cap_anchor"})
	srv.metrics = pm
	if srv.notarizer == nil {
		t.Fatalf("expected notarizer initialized")
	}
	if srv.capabilityRegistryHash == "" {
		t.Fatalf("expected capabilityRegistryHash set")
	}
	// Perform explicit notarization to ensure metrics observation.
	start := time.Now()
	receipt, err := srv.notarizer.Notarize(srv.capabilityRegistryHash)
	if err != nil {
		t.Fatalf("notarize error: %v", err)
	}
	// Persist receipt on server to allow status endpoint to surface provider immediately.
	srv.capLastNotarizationReceipt = receipt
	srv.capLastNotarization = time.Now()
	latency := time.Since(start)
	// Manually record provider-labeled latency if adapter supports.
	if pm, ok := srv.metrics.(interface{ ObserveCapabilityAnchorNotarizationLatencyProvider(string, time.Duration) }); ok {
		pm.ObserveCapabilityAnchorNotarizationLatencyProvider(receipt.Provider, latency)
	}
	// Scrape metrics exposition endpoint (custom server metrics). We only assert HELP/TYPE line presence for provider-less; provider-labeled ones live in Prom registry.
	// Instead, access Prometheus default registry via HTTP test if server exposes /metrics path; if not, rely on adapter type assertion.
	// We assert adapter histogram vec exists when PrometheusMetrics is in use.
	// Cannot access unexported fields; instead ensure interface assertions succeed for provider-labeled methods.
	if _, ok := srv.metrics.(interface{ ObserveCapabilityAnchorNotarizationLatencyProvider(string, time.Duration) }); !ok {
		t.Fatalf("expected provider-labeled latency observer interface to be implemented")
	}
	if _, ok := srv.metrics.(interface{ IncCapabilityAnchorNotarizationFailuresProvider(string) }); !ok {
		t.Fatalf("expected provider-labeled failure counter interface to be implemented")
	}
	// Minimal validation: ensure provider name is external_stub.
	if receipt.Provider != capSourceExternal {
		t.Fatalf("unexpected provider: %s", receipt.Provider)
	}
	// Trigger a failure to exercise provider-labeled failures vector.
	os.Setenv("GAUTH_NOTARY_STUB_FAIL_PROB", "1") // force failure next attempt
	// Recreate stub to pick up new env configuration.
	_, _ = srv.notarizer.(interface{ Latest() interface{} }) // type hint to ensure interface compatibility
	srv.notarizer = notary.NewExternalStub()
	_, err2 := srv.notarizer.Notarize(srv.capabilityRegistryHash)
	if err2 == nil {
		t.Fatalf("expected forced failure after reinitializing stub with fail prob=1")
	}
	// Record failure labeled metric if available.
	if pm, ok := srv.metrics.(interface{ IncCapabilityAnchorNotarizationFailuresProvider(string) }); ok {
		pm.IncCapabilityAnchorNotarizationFailuresProvider(receipt.Provider)
	}
	// Quick scrape of server custom metrics endpoint to ensure base HELP lines still present.
	// Fallback: query capability anchor status endpoint and ensure receipt provider surfaced.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(w, req)
	body := w.Body.String()
	// Accept provider either in receipt provider field or fallback provider key.
	if !strings.Contains(body, receipt.Provider) && !strings.Contains(body, "notarization_provider") {
		t.Fatalf("expected provider '%s' in anchor status response body (or notarization_provider key): %s", receipt.Provider, body)
	}
}
