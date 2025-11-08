package enforcement

import (
	"context"
	"testing"
	"time"
)

func TestEnforcer_RuleBasedEnforcement(t *testing.T) {
	enforcer := NewEnforcer()

	// Add a rule that denies access to sensitive resources
	rule := &Rule{
		ID:          "deny-sensitive",
		Name:        "Deny Sensitive Resources",
		Description: "Blocks access to sensitive resources",
		Condition: func(ctx context.Context, req *EnforcementRequest) bool {
			return req.Resource == "sensitive:data"
		},
		Action:   "deny",
		Priority: 1,
		Enabled:  true,
	}

	err := enforcer.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	req := &EnforcementRequest{
		Subject:  "user:alice",
		Resource: "sensitive:data",
		Action:   "read",
		Context:  map[string]interface{}{},
	}

	decision, err := enforcer.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if decision.Decision != "deny" {
		t.Errorf("Expected deny, got %s", decision.Decision)
	}

	if len(decision.AppliedRules) != 1 {
		t.Errorf("Expected 1 applied rule, got %d", len(decision.AppliedRules))
	}
}

func TestEnforcer_DisclosureEnforcement(t *testing.T) {
	enforcer := NewEnforcer()

	// Add disclosure requirement
	enforcer.disclosureManager.AddDisclosure(DisclosureRequirement{
		Type:        "data-usage",
		Description: "Data will be used for analytics",
		Required:    true,
	})

	// Request without acknowledged disclosure
	req := &EnforcementRequest{
		Subject:  "user:bob",
		Resource: "document:123",
		Action:   "read",
		Context:  map[string]interface{}{},
	}

	decision, err := enforcer.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if decision.Decision != "deny" {
		t.Errorf("Expected deny for missing disclosure, got %s", decision.Decision)
	}

	if len(decision.Disclosures) == 0 {
		t.Error("Expected disclosure requirements in decision")
	}
}

func TestEnforcer_DisclosureAcknowledged(t *testing.T) {
	enforcer := NewEnforcer()

	enforcer.disclosureManager.AddDisclosure(DisclosureRequirement{
		Type:        "data-usage",
		Description: "Data will be used for analytics",
		Required:    true,
	})

	// Request with acknowledged disclosure
	req := &EnforcementRequest{
		Subject:  "user:bob",
		Resource: "document:123",
		Action:   "read",
		Context:  map[string]interface{}{},
		Disclosures: []DisclosureRequirement{
			{
				Type:         "data-usage",
				Acknowledged: true,
			},
		},
	}

	decision, err := enforcer.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if decision.Decision == "deny" {
		t.Errorf("Expected allow with acknowledged disclosure, got %s", decision.Decision)
	}
}

func TestEnforcer_MultipleRules(t *testing.T) {
	enforcer := NewEnforcer()

	// Add allow rule
	allowRule := &Rule{
		ID:          "allow-public",
		Name:        "Allow Public Resources",
		Description: "Allows access to public resources",
		Condition: func(ctx context.Context, req *EnforcementRequest) bool {
			return req.Resource == "public:data"
		},
		Action:   "allow",
		Priority: 2,
		Enabled:  true,
	}

	// Add deny rule with higher priority
	denyRule := &Rule{
		ID:          "deny-all",
		Name:        "Deny All",
		Description: "Denies all access",
		Condition: func(ctx context.Context, req *EnforcementRequest) bool {
			return true
		},
		Action:   "deny",
		Priority: 1,
		Enabled:  false, // Disabled
	}

	enforcer.AddRule(allowRule)
	enforcer.AddRule(denyRule)

	req := &EnforcementRequest{
		Subject:  "user:charlie",
		Resource: "public:data",
		Action:   "read",
		Context:  map[string]interface{}{},
	}

	decision, err := enforcer.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if decision.Decision != "allow" {
		t.Errorf("Expected allow, got %s", decision.Decision)
	}
}

func TestEnforcer_AuditCallback(t *testing.T) {
	enforcer := NewEnforcer()

	auditCalled := false
	enforcer.SetAuditCallback(func(decision EnforcementDecision) {
		auditCalled = true
		if decision.Subject != "user:test" {
			t.Errorf("Expected subject user:test, got %v", decision.Metadata)
		}
	})

	req := &EnforcementRequest{
		Subject:  "user:test",
		Resource: "resource:test",
		Action:   "read",
		Context:  map[string]interface{}{},
	}

	_, err := enforcer.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(10 * time.Millisecond)

	if !auditCalled {
		t.Error("Audit callback was not called")
	}
}

func TestEnforcer_GetMetrics(t *testing.T) {
	enforcer := NewEnforcer()

	enforcer.AddRule(&Rule{
		ID:      "rule1",
		Enabled: true,
	})

	enforcer.AddRule(&Rule{
		ID:      "rule2",
		Enabled: false,
	})

	metrics := enforcer.GetMetrics()

	if metrics["total_rules"].(int) != 2 {
		t.Errorf("Expected 2 total rules, got %v", metrics["total_rules"])
	}

	if metrics["enabled_rules"].(int) != 1 {
		t.Errorf("Expected 1 enabled rule, got %v", metrics["enabled_rules"])
	}

	if metrics["ai_enabled"].(bool) {
		t.Error("Expected AI not enabled")
	}
}

// Mock AI Integration for testing
type MockAIIntegration struct {
	agentID string
}

func (m *MockAIIntegration) EvaluateEnforcement(ctx context.Context, req *EnforcementRequest) (*AIRecommendation, error) {
	return &AIRecommendation{
		Confidence: 0.95,
		Suggestion: "allow",
		Reasoning:  "Low risk access pattern",
		AgentID:    m.agentID,
	}, nil
}

func (m *MockAIIntegration) GetAgentID() string {
	return m.agentID
}

func TestEnforcer_AIIntegration(t *testing.T) {
	enforcer := NewEnforcer()

	mockAI := &MockAIIntegration{agentID: "g-agent-001"}
	enforcer.SetAIIntegration(mockAI)

	req := &EnforcementRequest{
		Subject:  "user:david",
		Resource: "document:456",
		Action:   "read",
		Context:  map[string]interface{}{},
	}

	decision, err := enforcer.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if decision.AIRecommendation == nil {
		t.Fatal("Expected AI recommendation")
	}

	if decision.AIRecommendation.AgentID != "g-agent-001" {
		t.Errorf("Expected agent ID g-agent-001, got %s", decision.AIRecommendation.AgentID)
	}

	if decision.AIRecommendation.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", decision.AIRecommendation.Confidence)
	}

	if decision.EnforcementType != EnforcementHybrid {
		t.Errorf("Expected hybrid enforcement type, got %s", decision.EnforcementType)
	}
}
