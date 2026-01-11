package pdp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPrometheusExportIncludesHistogramAndHelp(t *testing.T) {
	eng := NewInMemoryEngine(DenyOverridesStrategy{})
	// Add a simple policy to generate a few decisions
	eng.AddPolicy(Policy{
		ID:       "p_latency",
		Subjects: []string{"alice"},
		Rules: []Rule{
			{
				ID:        "r1",
				Actions:   []string{"read"},
				Resources: []string{"doc"},
				Effect:    "allow",
			},
		},
	})
	// Generate decisions to populate latency buckets
	for i := 0; i < 5; i++ {
		start := time.Now()
		// Introduce tiny sleep to vary latency
		time.Sleep(time.Duration(50*i) * time.Microsecond)
		_, err := eng.Evaluate(context.Background(), Request{Subject: "alice", Action: "read", Resource: "doc", Time: time.Now()})
		if err != nil {
			t.Fatalf("evaluation error: %v", err)
		}
		_ = start // reserved if we later manually record
	}
	out := eng.ExportPrometheus()
	// Basic checks
	mustContain := []string{
		"# HELP pdp_decisions_total",
		"# TYPE pdp_decisions_total counter",
		"pdp_decisions_total ",
		"# HELP pdp_decision_latency_seconds",
		"# TYPE pdp_decision_latency_seconds histogram",
		"pdp_decision_latency_seconds_bucket{le=\"0.000050\"}",
		"pdp_decision_latency_seconds_bucket{le=\"+Inf\"}",
		"pdp_decision_latency_seconds_sum",
		"pdp_decision_latency_seconds_count",
	}
	for _, token := range mustContain {
		if !strings.Contains(out, token) {
			t.Fatalf("prometheus export missing expected token: %s\nOutput:\n%s", token, out)
		}
	}
}
