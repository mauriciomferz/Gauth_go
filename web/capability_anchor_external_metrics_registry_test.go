package web

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
)

// TestExternalAnchorMetricsIsolatedRegistry ensures external anchor metrics collectors register and emit in a custom registry.
func TestExternalAnchorMetricsIsolatedRegistry(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory")
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111", Registry: reg})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	srv.metrics = pm
	// Allow initial anchor attempt.
	time.Sleep(40 * time.Millisecond)
	// Verify status includes receipt.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
	}
	if !hasSubstr(rec.Body.String(), "external_anchor_receipt") {
		t.Fatalf("receipt missing in status payload")
	}
	// Trigger another latency observation to ensure histogram bucket increments.
	pm.ObserveExternalAnchorLatency("memory", 7*time.Millisecond)
	// Collect metrics from isolated registry and assert key metric families exist.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather error: %v", err)
	}
	want := map[string]bool{
		"gauth_rfc0111_external_anchor_latency_seconds": false,
		"gauth_rfc0111_external_anchor_attempts_total":  false,
		"gauth_rfc0111_external_anchor_failures_total":  false,
		"gauth_rfc0111_external_anchor_age_seconds":     false,
		"gauth_rfc0111_external_anchor_last_hash_len":   false,
	}
	for _, mf := range mfs {
		name := mf.GetName()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	// Export exposition text snapshot for debugging if CI_DEBUG=1.
	if !allTrue(want) {
		var buf bytes.Buffer
		for _, mf := range mfs {
			buf.WriteString(mf.GetName() + "\n")
		}
		t.Log("metric families gathered:\n" + buf.String())
	}
	for n, seen := range want {
		if !seen {
			t.Fatalf("expected metric family %s registered", n)
		}
	}
}

func allTrue(m map[string]bool) bool {
	for _, v := range m {
		if !v {
			return false
		}
	}
	return true
}
