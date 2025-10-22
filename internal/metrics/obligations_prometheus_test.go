package metrics

import (
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
)

// TestPrometheusObligationsCounters ensures obligations executed/failed counters register and increment.
func TestPrometheusObligationsCounters(t *testing.T) {
	reg := prom.NewRegistry()
	pm := NewPrometheusMetrics(PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111", Registry: reg})
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
		case "gauth_rfc0111_obligations_executed_total":
			if len(mf.Metric) == 0 {
				t.Fatalf("executed metric empty")
			}
			exec = mf.Metric[0].Counter.GetValue()
		case "gauth_rfc0111_obligations_failed_total":
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
