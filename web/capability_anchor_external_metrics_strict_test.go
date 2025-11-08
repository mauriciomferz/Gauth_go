package web

import (
	"net/http/httptest"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestExternalAnchorMetricsStrict asserts specific numeric values for provider-labeled external anchor metrics.
//
//nolint:gocyclo // External anchor metrics test
func TestExternalAnchorMetricsStrict(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", memoryProvider)
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111", Registry: reg})
	// Use metrics-aware constructor so startup external anchoring attempt records directly into Prometheus.
	srv := NewBetaServerWithMetrics(":0", pm)
	// Allow initial anchor attempt
	time.Sleep(50 * time.Millisecond)
	// Issue an additional latency observation to ensure histogram bucket increments beyond startup attempt.
	pm.ObserveExternalAnchorLatency(memoryProvider, 12*time.Millisecond)
	// Hit status to confirm receipt
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if !hasSubstr(rec.Body.String(), "external_anchor_receipt") {
		t.Fatalf("receipt missing")
	}
	// Gather metrics
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather error: %v", err)
	}
	// Helper to find metric family by name
	find := func(name string) *dto.MetricFamily {
		for _, mf := range mfs {
			if mf.GetName() == name {
				return mf
			}
		}
		return nil
	}
	// Unlabeled counters
	attempts := find("gauth_rfc0111_external_anchor_attempts_total")
	if attempts == nil || len(attempts.Metric) == 0 {
		t.Fatalf("attempts counter missing")
	}
	if attempts.Metric[0].Counter == nil || attempts.Metric[0].Counter.GetValue() < 1 {
		t.Fatalf("attempts counter value <1")
	}
	failures := find("gauth_rfc0111_external_anchor_failures_total")
	if failures == nil || len(failures.Metric) == 0 {
		t.Fatalf("failures counter missing")
	}
	if failures.Metric[0].Counter == nil {
		t.Fatalf("failures counter nil")
	}
	// Provider-labeled attempts (Vec)
	attemptsProv := find("gauth_rfc0111_external_anchor_attempts_provider_total")
	if attemptsProv == nil || len(attemptsProv.Metric) == 0 {
		t.Fatalf("provider attempts vec missing")
	}
	// Expect at least one metric with label provider=memoryProvider
	provSeen := false
	for _, m := range attemptsProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == "provider" && lp.GetValue() == memoryProvider {
				provSeen = true
				if m.Counter.GetValue() < 1 {
					t.Fatalf("provider attempts <1")
				}
				break
			}
		}
	}
	if !provSeen {
		t.Fatalf("provider labeled attempts memory not found")
	}
	// Latency histogram (unlabeled)
	latHist := find("gauth_rfc0111_external_anchor_latency_seconds")
	if latHist == nil || len(latHist.Metric) == 0 {
		t.Fatalf("latency histogram missing")
	}
	if latHist.Metric[0].Histogram == nil || latHist.Metric[0].Histogram.GetSampleCount() == 0 {
		t.Fatalf("latency histogram sample count == 0")
	}
	// Provider-labeled latency histogram
	latHistProv := find("gauth_rfc0111_external_anchor_latency_provider_seconds")
	if latHistProv == nil || len(latHistProv.Metric) == 0 {
		t.Fatalf("provider latency histogram missing")
	}
	provLatSeen := false
	var provSampleCount uint64
	for _, m := range latHistProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == "provider" && lp.GetValue() == memoryProvider {
				provLatSeen = true
				provSampleCount = m.Histogram.GetSampleCount()
				break
			}
		}
	}
	if !provLatSeen {
		t.Fatalf("provider labeled latency histogram memory not found")
	}
	if provSampleCount == 0 {
		t.Fatalf("provider latency histogram sample count == 0")
	}
	// Age gauge
	ageGauge := find("gauth_rfc0111_external_anchor_age_seconds")
	if ageGauge == nil || len(ageGauge.Metric) == 0 {
		t.Fatalf("age gauge missing")
	}
	if ageGauge.Metric[0].Gauge == nil {
		t.Fatalf("age gauge value nil")
	}
	// Hash len gauge
	hashLen := find("gauth_rfc0111_external_anchor_last_hash_len")
	if hashLen == nil || len(hashLen.Metric) == 0 {
		t.Fatalf("hash length gauge missing")
	}
	if v := hashLen.Metric[0].Gauge.GetValue(); v <= 0 {
		t.Fatalf("expected hash length gauge >0 got %f", v)
	}
}
