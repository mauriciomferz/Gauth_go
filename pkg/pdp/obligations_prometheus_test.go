package pdp

import (
	"context"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	iobligations "github.com/mauriciomferz/AgentAuth/internal/obligations"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestPrometheusObligationMetricsIntegration ensures the Prometheus adapter records
// obligation latency histogram samples and mandatory failure counter increments.
func TestPrometheusObligationMetricsIntegration(t *testing.T) {
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Registry: reg})

	// Executor: one failing mandatory obligation (must_fail) and one succeeding (ok)
	exec := iobligations.NewSimpleExecutor(1, 2, []string{"must_fail"})
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).
		WithMetrics(pm).
		WithObligations(exec, "").
		WithObligationFailureDenies(true)

	engine.AddPolicy(Policy{ID: "p_prom", Subjects: []string{"zoe"}, Rules: []Rule{{ID: "r_prom", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "must_fail", Mandatory: true}, {ID: "ok", Mandatory: false}}})

	dec, err := engine.Evaluate(context.Background(), Request{Subject: "zoe", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Allow {
		t.Fatalf("expected deny due to mandatory failure")
	}

	// Gather metrics from registry
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather prometheus metrics: %v", err)
	}

	// Helpers to locate metric families
	findMF := func(name string) *dto.MetricFamily {
		for _, mf := range mfs {
			if mf.GetName() == name {
				return mf
			}
		}
		return nil
	}

	// Mandatory failure counter
	mandMF := findMF("agentauth_aap001_mandatory_obligation_failures_total")
	if mandMF == nil || len(mandMF.Metric) == 0 || mandMF.Metric[0].GetCounter().GetValue() < 1 {
		t.Fatalf("expected mandatory_obligation_failures_total >=1, got %#v", mandMF)
	}

	// Obligations failed counter
	failMF := findMF("agentauth_aap001_obligations_failed_total")
	if failMF == nil || len(failMF.Metric) == 0 || failMF.Metric[0].GetCounter().GetValue() < 1 {
		t.Fatalf("expected obligations_failed_total >=1, got %#v", failMF)
	}

	// Obligations executed counter (should be >=1 for the ok obligation)
	execMF := findMF("agentauth_aap001_obligations_executed_total")
	if execMF == nil || len(execMF.Metric) == 0 || execMF.Metric[0].GetCounter().GetValue() < 1 {
		t.Fatalf("expected obligations_executed_total >=1, got %#v", execMF)
	}

	// Latency histogram must have at least one sample (count >= 2 for two obligations)
	latMF := findMF("agentauth_aap001_obligation_latency_seconds")
	if latMF == nil {
		t.Fatalf("missing obligation_latency_seconds metric family")
	}
	var foundCount bool
	for _, m := range latMF.Metric {
		if h := m.GetHistogram(); h != nil {
			if h.GetSampleCount() >= 2 {
				foundCount = true
				break
			}
		}
	}
	if !foundCount {
		t.Fatalf("expected histogram sample count >=2 (two obligations) in obligation_latency_seconds")
	}
}
