package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// TestDailyAmountLimitExceeded verifies cumulative daily amount enforcement for transaction actions.
func TestDailyAmountLimitExceeded(t *testing.T) {
	mem := metrics.NewMemory()
	aud := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(aud, authorizer, WithMetrics(mem))

	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD", "max_amount": "50.00", "max_daily_amount": "100.00"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// First validation (40) should pass.
	amt1 := 40.0
	// Provide requested amount via context key that auditActionAmount() uses (prototype path).
	if err := svc.ValidateDelegationRich(WithRequestedAmount(context.Background(), "40"), resp.POA.ID, "bob", ValidationContext{Action: "transaction:execute", RequestedAmount: &amt1, Metadata: map[string]string{"currency": "USD"}}); err != nil {
		t.Fatalf("first validation failed: %v", err)
	}
	// Second validation (55) exceeds daily limit cumulative (95 -> 150 > 100) should fail with restriction error.
	amt2 := 55.0
	err2 := svc.ValidateDelegationRich(WithRequestedAmount(context.Background(), "55"), resp.POA.ID, "bob", ValidationContext{Action: "transaction:execute", RequestedAmount: &amt2, Metadata: map[string]string{"currency": "USD"}})
	if err2 == nil {
		t.Fatalf("expected daily limit restriction error")
	}
	if e, ok := err2.(aap.RFCError); !ok || e.Code != aap.ErrRestrictionExceeded {
		t.Fatalf("expected restriction_exceeded got %v", err2)
	}
}
