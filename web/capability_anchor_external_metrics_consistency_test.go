package web

import (
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestExternalAnchorMetricsConsistency ensures unlabeled and provider-labeled counters/histograms stay in sync for manual observations.
func TestExternalAnchorMetricsConsistency(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory")
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem:"AAP-001", Registry: reg})
	// Manual attempts & latency observations (simulate 3 successful anchors)
	for i := 0; i < 3; i++ {
		pm.IncExternalAnchorAttempts("memory")
		pm.ObserveExternalAnchorLatency("memory", time.Duration(5+i)*time.Millisecond)
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
	unlabeledAttempts := attempts.Metric[0].Counter.GetValue()
	attemptsProv := find("gauth_aap001_external_anchor_attempts_provider_total")
	if attemptsProv == nil {
		t.Fatalf("provider attempts vec missing")
	}
	var labeledAttempts float64
	for _, m := range attemptsProv.Metric {
		prov := ""
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider {
				prov = lp.GetValue()
				break
			}
		}
		if prov == memoryProvider {
			labeledAttempts = m.Counter.GetValue()
		}
	}
	if labeledAttempts == 0 {
		t.Fatalf("memory provider attempts labeled value missing")
	}
	if labeledAttempts != unlabeledAttempts {
		t.Fatalf("attempt mismatch unlabeled=%f labeled=%f", unlabeledAttempts, labeledAttempts)
	}
	// Histograms sample count consistency
	latHist := find("gauth_aap001_external_anchor_latency_seconds")
	if latHist == nil || len(latHist.Metric) == 0 || latHist.Metric[0].Histogram == nil {
		t.Fatalf("unlabeled latency histogram missing")
	}
	unlabeledSamples := latHist.Metric[0].Histogram.GetSampleCount()
	latHistProv := find("gauth_aap001_external_anchor_latency_provider_seconds")
	if latHistProv == nil {
		t.Fatalf("provider latency histogram missing")
	}
	var labeledSamples uint64
	for _, m := range latHistProv.Metric {
		prov := ""
		for _, lp := range m.Label {
			if lp.GetName() == labelProvider {
				prov = lp.GetValue()
				break
			}
		}
		if prov == memoryProvider {
			labeledSamples = m.Histogram.GetSampleCount()
		}
	}
	if labeledSamples == 0 {
		t.Fatalf("memory provider histogram samples missing")
	}
	if labeledSamples != unlabeledSamples {
		t.Fatalf("latency histogram sample mismatch unlabeled=%d labeled=%d", unlabeledSamples, labeledSamples)
	}
}
