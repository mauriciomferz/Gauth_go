package agentauth_aap_001

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// Test custom ValidationLimits override behavior.
func TestCustomValidationLimits(t *testing.T) {
	// Custom limits (very small) to force failures quickly.
	limits := ValidationLimits{
		MaxScopeItems:        2,
		MaxScopeLen:          5,
		MaxRestrictions:      1,
		MaxRestrictionKeyLen: 4,
		MaxRestrictionValLen: 4,
		MaxDuration:          time.Hour, // smaller than default 1 year
	}
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithValidationLimits(limits))
	// Authorize creation so authz does not block.
	if ma, ok := svc.authz.(*authz.MemoryAuthorizer); ok {
		ma.AddPolicy(authz.Policy{ID: "allow", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	}

	// 1. Scope item count exceed
	scope := []string{"a", "b", "c"} // 3 > MaxScopeItems(2)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: scope, Duration: time.Minute}); err == nil {
		t.Fatalf("expected scope count error")
	}

	// 2. Scope item length exceed
	scope = []string{"abcdef"} // len 6 > MaxScopeLen(5)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: scope, Duration: time.Minute}); err == nil {
		t.Fatalf("expected scope length error")
	}

	// 3. Restrictions count exceed
	rs := map[string]string{"k1": "v1", "k2": "v2"} // 2 > MaxRestrictions(1)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"a"}, Restrictions: rs, Duration: time.Minute}); err == nil {
		t.Fatalf("expected restrictions count error")
	}

	// 4. Restriction key length exceed
	rs = map[string]string{"longk": "v"} // key len 5 > MaxRestrictionKeyLen(4)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"a"}, Restrictions: rs, Duration: time.Minute}); err == nil {
		t.Fatalf("expected restriction key length error")
	}

	// 5. Restriction value length exceed
	rs = map[string]string{"k": "longv"} // val len 5 > MaxRestrictionValLen(4)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"a"}, Restrictions: rs, Duration: time.Minute}); err == nil {
		t.Fatalf("expected restriction value length error")
	}

	// 6. Duration exceed
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"a"}, Duration: limits.MaxDuration + time.Minute}); err == nil {
		t.Fatalf("expected duration exceed error")
	}

	// 7. Happy path within limits
	rs = map[string]string{"k": "v"}
	scope = []string{"ok"}
	del, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: scope, Restrictions: rs, Duration: time.Minute})
	if err != nil {
		t.Fatalf("expected success within limits, got error: %v", err)
	}
	if del == nil {
		t.Fatalf("expected delegation object")
	}
}

// Ensure defaults are applied when zero-value limits passed.
func TestValidationLimitsDefaultApplication(t *testing.T) {
	// Provide empty struct -> defaults should apply (we rely on ability to create within default bounds).
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithValidationLimits(ValidationLimits{}))
	if ma, ok := svc.authz.(*authz.MemoryAuthorizer); ok {
		ma.AddPolicy(authz.Policy{ID: "allow", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	}
	// Use values near documented defaults (assuming applyDefaults sets 32,128,32,64,256,365d).
	rs := map[string]string{"key": "val"}
	scope := []string{"read"}
	del, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: scope, Restrictions: rs, Duration: time.Hour})
	if err != nil {
		t.Fatalf("expected success with default limits: %v", err)
	}
	if del == nil {
		t.Fatalf("expected delegation object")
	}
}

func TestValidationLimitsRFCErrorCodes(t *testing.T) {
	limits := ValidationLimits{MaxScopeItems: 1}
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithValidationLimits(limits))
	if ma, ok := svc.authz.(*authz.MemoryAuthorizer); ok {
		ma.AddPolicy(authz.Policy{ID: "allow", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	}
	// Exceed scope items
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"a", "b"}, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error")
	}
	if e, ok := err.(aap.RFCError); !ok || e.Code != aap.ErrInvalidRequest {
		t.Fatalf("expected RFC invalid_request error, got %v", err)
	}
}
