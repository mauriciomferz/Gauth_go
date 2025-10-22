package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestPolicyMetricsPrometheusViolationCounters ensures violation counters lines are present.
func TestPolicyMetricsPrometheusViolationCounters(t *testing.T) {
	srv := NewBetaServer("")
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
	req := httptest.NewRequest("GET", "/api/v1/beta/policy/metrics/prometheus", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	out := w.Body.String()
	expected := []string{
		"gauth_scope_violations_total",
		"gauth_restriction_violations_total",
		"gauth_unauthorized_decisions_total",
		"gauth_expired_delegations_total",
		"gauth_revoked_delegations_total",
	}
	for _, token := range expected {
		if !strings.Contains(out, token) {
			t.Fatalf("missing counter %s in prometheus output", token)
		}
	}
}
