package metrics

import (
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
)

// TestPrometheusObligationsCounters ensures obligations executed/failed counters register and increment.
func TestPrometheusObligationsCounters(t *testing.T) {
	reg := prom.NewRegistry()
	pm := NewPrometheusMetrics(PrometheusAdapterOptions{Namespace: "AGENTAUTH", Subsystem: "aap001", Registry: reg})
	// Increment executed twice, failed once.
	pm.IncObligationsExecuted()
	pm.IncObligationsExecuted()
	pm.IncObligationsFailed()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var exec, fail float64
	for _, mf := range mfs {
		switch mf.GetName() {
		case "AGENTAUTH_aap001_obligations_executed_total":
			if len(mf.Metric) == 0 {
				t.Fatalf("executed metric empty")
			}
			exec = mf.Metric[0].Counter.GetValue()
		case "AGENTAUTH_aap001_obligations_failed_total":
			if len(mf.Metric) == 0 {
				t.Fatalf("failed metric empty")
			}
			fail = mf.Metric[0].Counter.GetValue()
		}
	}
	if exec != 2 {
		t.Fatalf("expected executed=2 got %f", exec)
	}
	if fail != 1 {
		t.Fatalf("expected failed=1 got %f", fail)
	}
}
