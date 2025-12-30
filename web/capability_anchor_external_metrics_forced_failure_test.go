package web

import (
	"net/http/httptest"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestExternalAnchorMetricsForcedFailure verifies that forced failure counters increment distinctly
// from general failures and attempts when AGENTAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS is set.
func TestExternalAnchorMetricsForcedFailure(t *testing.T) {
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "tsa_stub")
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS", "5")
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS", "10")
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB", "0.0") // eliminate probabilistic failures; rely purely on forced path
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_RETRIES", "2")     // allow some retries though success expected after forced failures consumed
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_RETRY_BASE_MS", "10")
	// Deterministic seed to remove randomness from latency only; probability zero.
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED", "1700011")
	// Force exactly one initial failure.
	t.Setenv("AGENTAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS", "1")
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem:"AAP-001", Registry: reg})
	srv := NewBetaServerWithMetrics(":0", pm)
	t.Cleanup(func() { srv.Shutdown() })
	// Wait for initial attempt & possible retry.
	time.Sleep(300 * time.Millisecond)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
	}
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
	forced := find("gauth_aap001_external_anchor_forced_failures_total")
	if attempts == nil || failures == nil || forced == nil {
		t.Fatalf("missing one of attempts/failures/forced counters")
	}
	aCount := attempts.Metric[0].Counter.GetValue()
	fCount := failures.Metric[0].Counter.GetValue()
	ffCount := forced.Metric[0].Counter.GetValue()
	if ffCount != 1 {
		t.Fatalf("expected forced failures == 1 got %f", ffCount)
	}
	if fCount < ffCount {
		t.Fatalf("expected general failures >= forced failures got failures=%f forced=%f", fCount, ffCount)
	}
	// Provider-labeled forced failures
	forcedProv := find("gauth_aap001_external_anchor_forced_failures_provider_total")
	if forcedProv == nil {
		t.Fatalf("provider forced failures vec missing")
	}
	var providerForced float64
	for _, m := range forcedProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
				providerForced = m.Counter.GetValue()
				break
			}
		}
	}
	if providerForced != ffCount {
		t.Fatalf("expected provider forced failures == %f got %f", ffCount, providerForced)
	}
	// Ensure at least one success occurred after forced failure (latency histogram sample present)
	latProv := find("gauth_aap001_external_anchor_latency_provider_seconds")
	if latProv == nil {
		t.Fatalf("latency provider histogram missing")
	}
	var sampleCount uint64
	for _, m := range latProv.Metric {
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider && lp.GetValue() == tsaStubProvider {
				sampleCount = m.Histogram.GetSampleCount()
				break
			}
		}
	}
	if sampleCount == 0 {
		t.Fatalf("expected latency sample after success; sampleCount=0 attempts=%f failures=%f forced=%f", aCount, fCount, ffCount)
	}
}
