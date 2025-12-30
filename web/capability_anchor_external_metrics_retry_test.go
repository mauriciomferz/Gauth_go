package web

import (
	"net/http/httptest"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestExternalAnchorMetricsRetrySuccess forces a transient failure then success via fail probability and retries.
//
//nolint:gocyclo // External anchor retry test
func TestExternalAnchorMetricsRetrySuccess(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "tsa_stub")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS", "5")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS", "10")
	// Medium fail prob to likely hit at least one retry; allow up to 4 retries.
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB", "0.6")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES", "4")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RETRY_BASE_MS", "10")
	// Deterministic RNG seed to ensure we observe at least one failure then a success across attempts.
	// Seed chosen empirically so sequence of rand.Float64() with failProb=0.6 yields failure early and success later.
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED", "1700001")
	// Force exactly one initial failure irrespective of probability to stabilize test expectations.
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS", "1")
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem:"AAP-001", Registry: reg})
	srv := NewBetaServerWithMetrics(":0", pm) // server needed for startup attempt
	t.Cleanup(func() { srv.Shutdown() })
	// Wait for initial attempt + potential retries to finish.
	time.Sleep(400 * time.Millisecond)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
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
	attempts := find("gauth_aap001_external_anchor_attempts_total")
	if attempts == nil || len(attempts.Metric) == 0 {
		t.Fatalf("attempts counter missing")
	}
	attemptCount := attempts.Metric[0].Counter.GetValue()
	if attemptCount < 1 {
		t.Fatalf("attempts <1")
	}
	// Provider-labeled attempts
	attemptsProv := find("gauth_aap001_external_anchor_attempts_provider_total")
	if attemptsProv == nil {
		t.Fatalf("provider attempts vec missing")
	}
	var providerAttempts float64
	for _, m := range attemptsProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
				providerAttempts = m.Counter.GetValue()
				break
			}
		}
	}
	if providerAttempts == 0 {
		t.Fatalf("provider labeled attempts tsa-stub missing")
	}
	// We no longer enforce a failure occurrence: deterministic seeding surfaced that some seeds produce
	// immediate success sequences even with moderate failProb. Observability of success path (latency sample)
	// is sufficient; failure scenarios are covered by TestExternalAnchorMetricsRetryAllFail.
	failuresProv := find("gauth_aap001_external_anchor_failures_provider_total")
	var providerFailures float64
	if failuresProv != nil {
		for _, m := range failuresProv.Metric {
			for _, lp := range m.Label {
				if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
					providerFailures = m.Counter.GetValue()
					break
				}
			}
		}
	}
	if providerFailures < 1 {
		t.Fatalf("expected at least one forced failure before success, got %f", providerFailures)
	}
	latProv := find("gauth_aap001_external_anchor_latency_provider_seconds")
	var sampleCount uint64
	if latProv != nil {
		for _, m := range latProv.Metric {
			for _, lp := range m.Label {
				if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
					sampleCount = m.Histogram.GetSampleCount()
					break
				}
			}
		}
	}
	if sampleCount == 0 {
		t.Fatalf("expected latency sample count >0 after success")
	}
}

// TestExternalAnchorMetricsRetryAllFail forces all failures via prob=1 and checks attempts == retries+1 and failures == attempts.
func TestExternalAnchorMetricsRetryAllFail(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "tsa_stub")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS", "5")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS", "10")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB", "1")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES", "3")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RETRY_BASE_MS", "5")
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem:"AAP-001", Registry: reg})
	_ = NewBetaServerWithMetrics(":0", pm)
	time.Sleep(300 * time.Millisecond)
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
	attempts := find("gauth_aap001_external_anchor_attempts_total")
	failures := find("gauth_aap001_external_anchor_failures_total")
	if attempts == nil || failures == nil {
		t.Fatalf("missing attempts or failures counters")
	}
	aCount := attempts.Metric[0].Counter.GetValue()
	fCount := failures.Metric[0].Counter.GetValue()
	if aCount != 4 {
		t.Fatalf("expected 4 attempts (initial+3 retries) got %f", aCount)
	}
	if fCount != aCount {
		t.Fatalf("expected failures == attempts got failures=%f attempts=%f", fCount, aCount)
	}
	// Provider-labeled failures count equality
	failuresProv := find("gauth_aap001_external_anchor_failures_provider_total")
	if failuresProv == nil {
		t.Fatalf("provider failures vec missing")
	}
	var providerFailures float64
	for _, m := range failuresProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
				providerFailures = m.Counter.GetValue()
				break
			}
		}
	}
	if providerFailures != aCount {
		t.Fatalf("expected provider failures == attempts got %f vs %f", providerFailures, aCount)
	}
	// Age & hash len gauges should remain zero
	age := find("gauth_aap001_external_anchor_age_seconds")
	hlen := find("gauth_aap001_external_anchor_last_hash_len")
	if age == nil || hlen == nil {
		t.Fatalf("missing age or hash len gauges")
	}
	if age.Metric[0].Gauge.GetValue() != 0 {
		t.Fatalf("expected age 0 got %f", age.Metric[0].Gauge.GetValue())
	}
	if hlen.Metric[0].Gauge.GetValue() != 0 {
		t.Fatalf("expected hash len 0 got %f", hlen.Metric[0].Gauge.GetValue())
	}
}
