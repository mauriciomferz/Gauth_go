package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// TestSemanticRestore ensures SetSemanticSnapshot overwrites counters correctly.
func TestSemanticRestore(t *testing.T) {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-all-alice", Subject: "alice", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	// Generate some increments
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Restrictions: map[string]string{"currency": "USD", "max_amount": "50.0", "jurisdiction": "US"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	// Trigger currency mismatch
	err = svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", ValidationContext{Action: "transaction:execute", Metadata: map[string]string{"currency": "EUR"}})
	if err == nil {
		t.Fatalf("expected currency mismatch error")
	}
	// Trigger restriction mismatch (jurisdiction)
	err = svc.ValidateDelegationRich(context.Background(), resp.POA.ID, "bob", ValidationContext{Action: "transaction:execute", Metadata: map[string]string{"currency": "USD", "jurisdiction": "CA"}})
	if err == nil {
		t.Fatalf("expected restriction mismatch error")
	}
	snap := svc.SemanticSnapshot()
	if snap["currency_mismatch"] != 1 || snap["restriction_mismatch"] != 1 {
		t.Fatalf("expected initial counters 1/1 got %+v", snap)
	}
	// Now restore to a custom snapshot (simulate persisted state)
	restore := map[string]uint64{"amount_limit_exceeded": 7, "currency_mismatch": 5, "scope_violation": 3, "restriction_mismatch": 9}
	svc.SetSemanticSnapshot(restore)
	after := svc.SemanticSnapshot()
	for k, v := range restore {
		if after[k] != v {
			t.Fatalf("expected %s=%d got %d", k, v, after[k])
		}
	}
}
