package jurisdiction

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/compliance"
)

// TestValidatorMetricsPrometheus ensures validator metrics Prometheus exposition
// contains expected core metrics after sample validations.
func TestValidatorMetricsPrometheus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := NewEnforcementEngine()
	integration := &ServerIntegration{engine: eng}
	validator := eng.validator

	// Perform sample jurisdiction validation (EU board approval path) to populate latency + approval counters.
	if err := validator.ValidateJurisdiction(context.Background(), compliance.JurisdictionEU, "high_value_transaction"); err != nil {
		t.Fatalf("ValidateJurisdiction board approval path: %v", err)
	}
	// Entity validation (success + failure)
	if err := validator.ValidateEntityType(compliance.JurisdictionUS, compliance.EntityTypeCorporation); err != nil {
		t.Fatalf("entity validation success unexpected error: %v", err)
	}
	if err := validator.ValidateEntityType(compliance.JurisdictionUS, compliance.EntityType("nonexistent")); err == nil {
		t.Fatalf("expected failure for unsupported entity type")
	}
	// Value limit checks (one normal, one violation via custom requirements)
	reqUS, err := validator.GetJurisdictionRules(compliance.JurisdictionUS)
	if err != nil {
		t.Fatalf("get US rules: %v", err)
	}
	if err := validator.ValidateJurisdictionRequirements(context.Background(), reqUS, "trade_execution"); err != nil {
		t.Fatalf("expected normal value limit check success: %v", err)
	}
	// Create a violating requirements set
	badReq := &compliance.JurisdictionRequirements{
		Jurisdiction: compliance.JurisdictionUS, ValueLimits: map[string]float64{"bad_action": 0},
		RequiredApprovals: map[string]compliance.ApprovalLevel{"bad_action": ""},
	}
	if err := validator.ValidateJurisdictionRequirements(context.Background(), badReq, "bad_action"); err == nil {
		t.Fatalf("expected failure for bad_action invalid limit & approval")
	}

	h := NewAPIHandler(integration)
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/jurisdiction/validator/metrics/prometheus", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	// Core metrics to assert presence
	for _, metric := range []string{
		"agentauth_validator_validation_attempts_total",
		"agentauth_validator_validation_successes_total",
		"agentauth_validator_validation_failures_total",
		"agentauth_validator_entity_validation_attempts_total",
		"agentauth_validator_entity_validation_failures_total",
		"agentauth_validator_value_limit_checks_total",
		"agentauth_validator_value_limit_violations_total",
		"agentauth_validator_approval_checks_total",
		"agentauth_validator_approval_failures_total",
		"agentauth_validator_board_approval_checks_total",
		"agentauth_validator_total_validation_latency_ms",
		"agentauth_validator_last_validation_latency_ms",
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("expected metric %s in body", metric)
		}
	}
	if len(body) == 0 {
		t.Fatalf("empty validator metrics body")
	}
}
