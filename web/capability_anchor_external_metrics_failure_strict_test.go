package web

import (
	"net/http/httptest"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestExternalAnchorMetricsFailureStrict asserts failure counters increment when provider always fails.
//
//nolint:gocyclo // External anchor failure test
func TestExternalAnchorMetricsFailureStrict(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "tsa_stub")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB", "1") // force failure
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS", "5")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS", "10")
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111", Registry: reg})
	// Construct server with provided metrics adapter so initial attempted (and failing) anchor uses Prometheus.
	srv := NewBetaServerWithMetrics(":0", pm)
	t.Cleanup(func() { srv.Shutdown() })
	// Sleep briefly to allow initial attempted (and failing) anchor using normalized provider label.
	time.Sleep(40 * time.Millisecond)
	// Status endpoint should NOT include external_anchor_receipt since provider always fails.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
	}
	if hasSubstr(rec.Body.String(), "external_anchor_receipt") {
		t.Fatalf("unexpected receipt present under forced failure")
	}
	// Gather metrics
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather error: %v", err)
	}
	find := func(name string) *dto.MetricFamily {
		for _, mf := range mfs {
			if mf.GetName() == name {
				return mf
			}
		}
		return nil
	}
	attempts := find("gauth_rfc0111_external_anchor_attempts_total")
	if attempts == nil || len(attempts.Metric) == 0 || attempts.Metric[0].Counter == nil {
		t.Fatalf("attempts counter missing")
	}
	if attempts.Metric[0].Counter.GetValue() < 1 {
		t.Fatalf("attempts counter expected >=1")
	}
	failures := find("gauth_rfc0111_external_anchor_failures_total")
	if failures == nil || len(failures.Metric) == 0 || failures.Metric[0].Counter == nil {
		t.Fatalf("failures counter missing")
	}
	if failures.Metric[0].Counter.GetValue() < 1 {
		t.Fatalf("failures counter expected >=1")
	}
	// Provider-labeled failures
	failuresProv := find("gauth_rfc0111_external_anchor_failures_provider_total")
	if failuresProv == nil {
		t.Fatalf("provider failures vec missing")
	}
	seen := false
	for _, m := range failuresProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
				if m.Counter.GetValue() < 1 {
					t.Fatalf("provider labeled failures <1")
				}
				seen = true
			}
		}
	}
	if !seen {
		t.Fatalf("provider tsa-stub failures metric not found")
	}
	// Age gauge should be zero (no success). Hash len gauge should be zero.
	age := find("gauth_rfc0111_external_anchor_age_seconds")
	if age == nil || len(age.Metric) == 0 || age.Metric[0].Gauge == nil {
		t.Fatalf("age gauge missing")
	}
	if age.Metric[0].Gauge.GetValue() != 0 {
		t.Fatalf("expected age gauge 0 for no success got %f", age.Metric[0].Gauge.GetValue())
	}
	hlen := find("gauth_rfc0111_external_anchor_last_hash_len")
	if hlen == nil || len(hlen.Metric) == 0 || hlen.Metric[0].Gauge == nil {
		t.Fatalf("hash len gauge missing")
	}
	if hlen.Metric[0].Gauge.GetValue() != 0 {
		t.Fatalf("expected hash len gauge 0 for no success got %f", hlen.Metric[0].Gauge.GetValue())
	}
}
