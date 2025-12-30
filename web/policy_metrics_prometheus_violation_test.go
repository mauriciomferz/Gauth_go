package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
)

// TestPolicyMetricsPrometheusViolationCounters ensures violation counters lines are present.
func TestPolicyMetricsPrometheusViolationCounters(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Trigger some increments: evaluate unsupported policy action to produce unauthorized
	// For simplicity we directly touch the metrics if available
	if mm, ok := srv.metrics.(*imetrics.Memory); ok {
		mm.IncScopeViolations()
		mm.IncRestrictionViolations()
		mm.IncUnauthorized()
		mm.IncExpired()
		mm.IncRevoked()
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/policy/metrics/prometheus", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	out := w.Body.String()
	expected := []string{
		"AGENTAUTH_scope_violations_total",
		"AGENTAUTH_restriction_violations_total",
		"AGENTAUTH_unauthorized_decisions_total",
		"AGENTAUTH_expired_delegations_total",
		"AGENTAUTH_revoked_delegations_total",
	}
	for _, token := range expected {
		if !strings.Contains(out, token) {
			t.Fatalf("missing counter %s in prometheus output", token)
		}
	}
}
