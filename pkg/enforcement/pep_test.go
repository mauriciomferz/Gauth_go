package enforcement

import (
	"context"
	"testing"
)

// MockPDP implements PDPClient for testing
type MockPDP struct {
	decision string
	reason   string
	err      error
}

func (m *MockPDP) Decide(ctx context.Context, req *EnforcementRequest) (*PDPDecision, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &PDPDecision{
		Decision: m.decision,
		Reason:   m.reason,
		Policies: []string{"test-policy"},
	}, nil
}

func TestSupplySidePEP_EnforceClientAction_PDPPermit(t *testing.T) {
	// Supply-side: Client enforcing its own authorization
	pdp := &MockPDP{decision: "permit", reason: "allowed"}
	pep := NewSupplySidePEP("ai:agent-123", pdp)

	err := pep.EnforceClientAction(context.Background(), "document:456", "read", nil)
	if err != nil {
		t.Errorf("Expected nil error for permitted action, got: %v", err)
	}
}

func TestSupplySidePEP_EnforceClientAction_PDPDeny(t *testing.T) {
	// Supply-side: Client must not proceed when PDP denies
	pdp := &MockPDP{decision: "deny", reason: "insufficient permissions"}
	pep := NewSupplySidePEP("ai:agent-123", pdp)

	err := pep.EnforceClientAction(context.Background(), "secret:data", "read", nil)
	if err == nil {
		t.Error("Expected error for denied action, got nil")
	}

	if err != nil && err.Error() != "supply-side PEP: action denied by PDP: insufficient permissions" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestSupplySidePEP_EnforceClientAction_WithRule(t *testing.T) {
	// Supply-side: Client with local enforcement rules
	pdp := &MockPDP{decision: "permit", reason: "allowed"}
	pep := NewSupplySidePEP("ai:agent-123", pdp)

	// Add a deny rule for sensitive resources
	if err := pep.AddRule(&Rule{
		ID:   "deny-sensitive",
		Name: "Deny Sensitive Resources",
		Condition: func(ctx context.Context, req *EnforcementRequest) bool {
			return req.Resource == "sensitive:data"
		},
		Action:  "deny",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Should be denied by rule even though PDP permits
	err := pep.EnforceClientAction(context.Background(), "sensitive:data", "read", nil)
	if err == nil {
		t.Error("Expected error for rule-denied action, got nil")
	}
}

func TestDemandSidePEP_ValidateClientCompliance_PDPPermit(t *testing.T) {
	// Demand-side: Resource server validating client action
	pdp := &MockPDP{decision: "permit", reason: "client authorized"}
	pep := NewDemandSidePEP("server:rs-1", "owner:alice", pdp)

	err := pep.ValidateClientCompliance(
		context.Background(),
		"ai:agent-123",
		"document:456",
		"read",
		"token-xyz",
		nil,
	)

	if err != nil {
		t.Errorf("Expected nil error for permitted client action, got: %v", err)
	}
}

func TestDemandSidePEP_ValidateClientCompliance_PDPDeny(t *testing.T) {
	// Demand-side: Resource server must reject unauthorized client
	pdp := &MockPDP{decision: "deny", reason: "client not authorized for this resource"}
	pep := NewDemandSidePEP("server:rs-1", "owner:alice", pdp)

	err := pep.ValidateClientCompliance(
		context.Background(),
		"ai:agent-456",
		"document:789",
		"write",
		"token-abc",
		nil,
	)

	if err == nil {
		t.Error("Expected error for denied client action, got nil")
	}

	if err != nil && err.Error() != "demand-side PEP: client action denied by PDP: client not authorized for this resource" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestDemandSidePEP_ValidateClientCompliance_WithRule(t *testing.T) {
	// Demand-side: Resource server with local enforcement rules
	pdp := &MockPDP{decision: "permit", reason: "allowed"}
	pep := NewDemandSidePEP("server:rs-1", "owner:alice", pdp)

	// Add a rule to deny write operations from specific client
	if err := pep.AddRule(&Rule{
		ID:   "deny-client-write",
		Name: "Deny Specific Client Writes",
		Condition: func(ctx context.Context, req *EnforcementRequest) bool {
			return req.Subject == "ai:agent-456" && req.Action == "write"
		},
		Action:  "deny",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Should be denied by rule
	err := pep.ValidateClientCompliance(
		context.Background(),
		"ai:agent-456",
		"document:789",
		"write",
		"token-abc",
		nil,
	)

	if err == nil {
		t.Error("Expected error for rule-denied client action, got nil")
	}
}

func TestPEPSides_Distinction(t *testing.T) {
	// Verify that supply-side and demand-side PEPs are distinct
	pdp := &MockPDP{decision: "permit", reason: "allowed"}

	supplySide := NewSupplySidePEP("ai:agent-123", pdp)
	demandSide := NewDemandSidePEP("server:rs-1", "owner:alice", pdp)

	// Verify they are different types
	if supplySide.Enforcer == demandSide.Enforcer {
		t.Error("Supply-side and demand-side PEPs should have distinct enforcers")
	}

	// Verify client ID is set correctly
	if supplySide.clientID != "ai:agent-123" {
		t.Errorf("Expected clientID 'ai:agent-123', got '%s'", supplySide.clientID)
	}

	// Verify server/owner IDs are set correctly
	if demandSide.serverID != "server:rs-1" {
		t.Errorf("Expected serverID 'server:rs-1', got '%s'", demandSide.serverID)
	}
	if demandSide.ownerID != "owner:alice" {
		t.Errorf("Expected ownerID 'owner:alice', got '%s'", demandSide.ownerID)
	}
}

func TestSupplySidePEP_NoPDP(t *testing.T) {
	// Supply-side PEP without PDP (only local rules)
	pep := NewSupplySidePEP("ai:agent-123", nil)

	// Should allow by default (no rules)
	err := pep.EnforceClientAction(context.Background(), "document:123", "read", nil)
	if err != nil {
		t.Errorf("Expected nil error without PDP and rules, got: %v", err)
	}
}

func TestDemandSidePEP_NoPDP(t *testing.T) {
	// Demand-side PEP without PDP (only local rules)
	pep := NewDemandSidePEP("server:rs-1", "owner:alice", nil)

	// Should allow by default (no rules)
	err := pep.ValidateClientCompliance(
		context.Background(),
		"ai:agent-123",
		"document:123",
		"read",
		"token-xyz",
		nil,
	)

	if err != nil {
		t.Errorf("Expected nil error without PDP and rules, got: %v", err)
	}
}

func TestSupplySidePEP_ContextPropagation(t *testing.T) {
	// Verify context is properly propagated through enforcement
	pdp := &MockPDP{decision: "permit", reason: "allowed"}
	pep := NewSupplySidePEP("ai:agent-123", pdp)

	ctx := make(map[string]interface{})
	ctx["request_id"] = "req-789"
	ctx["timestamp"] = "2025-11-07T10:00:00Z"

	err := pep.EnforceClientAction(context.Background(), "document:456", "read", ctx)
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

func TestDemandSidePEP_TokenValidation(t *testing.T) {
	// Demand-side PEP validates client token
	pdp := &MockPDP{decision: "permit", reason: "token valid"}
	pep := NewDemandSidePEP("server:rs-1", "owner:alice", pdp)

	// Add rule to validate token presence
	if err := pep.AddRule(&Rule{
		ID:   "require-token",
		Name: "Require Valid Token",
		Condition: func(ctx context.Context, req *EnforcementRequest) bool {
			token, ok := req.Context["client_token"].(string)
			return !ok || token == ""
		},
		Action:  "deny",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Should be denied without token
	err := pep.ValidateClientCompliance(
		context.Background(),
		"ai:agent-123",
		"document:456",
		"read",
		"", // empty token
		nil,
	)

	if err == nil {
		t.Error("Expected error for missing token, got nil")
	}
}
