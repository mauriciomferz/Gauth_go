package authz

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusLabels(t *testing.T) {
	// Note: Generic promauto registration is global.
	// To test specifically, we rely on the fact this test process is isolated or we just check if value > 0.

	p := NewPrometheusMetricsProvider("authz_test")
	ma := NewMemoryAuthorizer()
	ma.SetMetricsProvider(p)

	// Add policy
	ma.AddPolicy(Policy{
		ID:       "p1",
		Effect:   Allow,
		Subject:  "alice",
		Resource: "file1",
		Actions:  []string{"read"},
	})

	req := Request{
		Subject:  "alice",
		Resource: "file1",
		Action:   "read",
	}

	// Decision: Allow
	_, _ = ma.Authorize(context.Background(), req)

	// Verify decision metric
	// We expect authz_test_decisions_total{action="read", outcome="allow", resource_type="resource"} == 1

	// Since we can't easily access the internal vector from 'p' (it's unexported fields in my impl),
	// I should have made them exported or added a getter.
	// Or I can use testutil.GatherAndCount() if I register it to a specific registry.
	// But NewPrometheusMetricsProvider uses global registry.

	// However, I can read the vector if I change the struct field to be exported or add a getter.
	// Or I can just blindly check the DefaultGatherer output.

	// Wait, I can't read 'p.decisions' because I didn't export it in metrics_export.go.
	// I will update metrics_export.go to export fields or provide access for testing?
	// Or I can rely on 'prometheus.DefaultGatherer'.

	ms, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather failed: %v", err)
	}

	found := false
	for _, m := range ms {
		if m.GetName() == "authz_test_decisions_total" {
			for _, metric := range m.GetMetric() {
				// Check labels
				labels := metric.GetLabel()
				matchAction := false
				matchOutcome := false
				for _, l := range labels {
					if l.GetName() == "action" && l.GetValue() == "read" {
						matchAction = true
					}
					if l.GetName() == "outcome" && l.GetValue() == "allow" {
						matchOutcome = true
					}
				}
				if matchAction && matchOutcome {
					found = true
					if metric.GetCounter().GetValue() < 1 {
						t.Errorf("Expected value >= 1, got %v", metric.GetCounter().GetValue())
					}
				}
			}
		}
	}

	if !found {
		t.Errorf("Did not find expected metric authz_test_decisions_total with labels")
	}
}

// Helper to check if Gather contains what we want.
