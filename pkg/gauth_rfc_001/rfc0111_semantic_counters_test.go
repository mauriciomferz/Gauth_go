package gauth_rfc_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// newTestService constructs a fresh RFC0111 service with memory audit/authorizer for isolation.
func newTestService() *Service {
	memAuthz := authz.NewMemoryAuthorizer()
	// Seed broad allow policy for test convenience (simplifies delegation creation authorization).
	memAuthz.AddPolicy(authz.Policy{ID: "allow-all-alice", Subject: "alice", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})
	return NewService(audit.NewMemoryLogger(nil), memAuthz)
}

// TestSemanticCountersScopeViolation ensures scope_violation counter increments on action mismatch.
func TestSemanticCountersScopeViolation(t *testing.T) {
	svc := newTestService()
	// Create delegation with specific scope
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	// Validate with different action causing scope violation
	vctx := ValidationContext{Action: "transaction:refund", Metadata: map[string]string{"currency": "USD"}}
	err = svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", vctx)
	if err == nil {
		t.Fatalf("expected scope violation error")
	}
	snap := svc.SemanticSnapshot()
	if snap["scope_violation"] != 1 {
		t.Fatalf("expected scope_violation=1 got %d", snap["scope_violation"])
	}
	if snap["amount_limit_exceeded"] != 0 || snap["currency_mismatch"] != 0 || snap["restriction_mismatch"] != 0 {
		t.Fatalf("unexpected other counter increments: %+v", snap)
	}
}

// TestSemanticCountersAmountLimitExceeded ensures amount_limit_exceeded counter increments when requested amount exceeds max_amount.
func TestSemanticCountersAmountLimitExceeded(t *testing.T) {
	svc := newTestService()
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD", "max_amount": "100.00"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	amt := 150.00
	vctx := ValidationContext{Action: "transaction:execute", RequestedAmount: &amt, Metadata: map[string]string{"currency": "USD"}}
	err = svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", vctx)
	if err == nil {
		t.Fatalf("expected amount limit exceeded error")
	}
	snap := svc.SemanticSnapshot()
	if snap["amount_limit_exceeded"] != 1 {
		t.Fatalf("expected amount_limit_exceeded=1 got %d", snap["amount_limit_exceeded"])
	}
	if snap["scope_violation"] != 0 || snap["currency_mismatch"] != 0 || snap["restriction_mismatch"] != 0 {
		t.Fatalf("unexpected other counter increments: %+v", snap)
	}
}

// TestSemanticCountersCurrencyMismatch ensures currency_mismatch counter increments when metadata currency differs from restriction.
func TestSemanticCountersCurrencyMismatch(t *testing.T) {
	svc := newTestService()
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	vctx := ValidationContext{Action: "transaction:execute", Metadata: map[string]string{"currency": "EUR"}}
	err = svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", vctx)
	if err == nil {
		t.Fatalf("expected currency mismatch error")
	}
	snap := svc.SemanticSnapshot()
	if snap["currency_mismatch"] != 1 {
		t.Fatalf("expected currency_mismatch=1 got %d", snap["currency_mismatch"])
	}
	if snap["scope_violation"] != 0 || snap["amount_limit_exceeded"] != 0 || snap["restriction_mismatch"] != 0 {
		t.Fatalf("unexpected other counter increments: %+v", snap)
	}
}

// TestSemanticCountersRestrictionMismatch ensures restriction_mismatch increments for non-special restriction key mismatch.
func TestSemanticCountersRestrictionMismatch(t *testing.T) {
	svc := newTestService()
	// regulatory scope requires jurisdiction restriction; include currency to avoid triggering currency logic
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"regulatory:report"}, Restrictions: map[string]string{"jurisdiction": "US"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	vctx := ValidationContext{Action: "regulatory:report", Metadata: map[string]string{"jurisdiction": "CA"}}
	err = svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", vctx)
	if err == nil {
		t.Fatalf("expected restriction mismatch error")
	}
	snap := svc.SemanticSnapshot()
	if snap["restriction_mismatch"] != 1 {
		t.Fatalf("expected restriction_mismatch=1 got %d", snap["restriction_mismatch"])
	}
	if snap["scope_violation"] != 0 || snap["amount_limit_exceeded"] != 0 || snap["currency_mismatch"] != 0 {
		t.Fatalf("unexpected other counter increments: %+v", snap)
	}
}

// TestSemanticCountersDailyAmountLimitExceeded ensures daily_amount_limit_exceeded increments on cumulative breach.
func TestSemanticCountersDailyAmountLimitExceeded(t *testing.T) {
	svc := newTestService()
	// Create delegation with both per-transaction and daily cumulative limits
	// Set max_amount higher than each individual request so daily cumulative breach is isolated.
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD", "max_amount": "70.00", "max_daily_amount": "100.00"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	amt1 := 60.0
	if err := svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", ValidationContext{Action: "transaction:execute", RequestedAmount: &amt1, Metadata: map[string]string{"currency": "USD"}}); err != nil {
		t.Fatalf("first validation should pass: %v", err)
	}
	amt2 := 50.0 // cumulative 110 > 100 triggers daily limit exceeded without per-tx breach
	err2 := svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", ValidationContext{Action: "transaction:execute", RequestedAmount: &amt2, Metadata: map[string]string{"currency": "USD"}})
	if err2 == nil {
		t.Fatalf("expected daily limit restriction error")
	}
	snap := svc.SemanticSnapshot()
	if snap["daily_amount_limit_exceeded"] != 1 {
		t.Fatalf("expected daily_amount_limit_exceeded=1 got %d", snap["daily_amount_limit_exceeded"])
	}
	// Ensure other counters untouched
	if snap["amount_limit_exceeded"] != 0 || snap["scope_violation"] != 0 || snap["currency_mismatch"] != 0 || snap["restriction_mismatch"] != 0 {
		t.Fatalf("unexpected other counter increments: %+v", snap)
	}
}
