package gauth_aap_001

import (
	"fmt"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/rfc"
)

// NOTE: These tests exercise default limits (hard-coded defaults applied via ValidationLimits.applyDefaults).
// Configurable override tests live in aap001_validation_limits_test.go.

func newSvcForLimits() *Service {
	return NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer())
}

func TestScopeTooManyEntries(t *testing.T) {
	svc := newSvcForLimits()
	// Construct scope exceeding assumed limit (e.g., 32). We build 40 entries.
	scope := make([]string, 40)
	for i := range scope {
		scope[i] = "r"
	}
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: scope, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for too many scope entries")
	}
	if rfce, ok := err.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request got %v", err)
	}
}

func TestScopeItemTooLong(t *testing.T) {
	svc := newSvcForLimits()
	// Single scope string exceeding assumed max length (e.g., 64). Use 200 chars.
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{string(long)}, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for long scope item")
	}
	if rfce, ok := err.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request got %v", err)
	}
}

func TestRestrictionsTooMany(t *testing.T) {
	svc := newSvcForLimits()
	// Allow creation so we test validation path not authz denial.
	if ma, ok := svc.authz.(*authz.MemoryAuthorizer); ok {
		ma.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	}
	// Build restrictions map exceeding actual limit (32). Use 33 entries.
	rs := make(map[string]string)
	for i := 0; i < 33; i++ {
		rs["k"+string(rune('a'+(i%26)))+fmt.Sprintf("%d", i)] = "v"
	}
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for too many restrictions")
	}
	if rfce, ok := err.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request got %v", err)
	}
}

func TestRestrictionKeyTooLong(t *testing.T) {
	svc := newSvcForLimits()
	// Key length > actual limit (64). Use 80 chars.
	key := make([]byte, 80)
	for i := range key {
		key[i] = 'k'
	}
	rs := map[string]string{string(key): "v"}
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for long restriction key")
	}
	if rfce, ok := err.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request got %v", err)
	}
}

func TestRestrictionValueTooLong(t *testing.T) {
	svc := newSvcForLimits()
	// Value length > actual limit (256). Use 300 chars.
	val := make([]byte, 300)
	for i := range val {
		val[i] = 'v'
	}
	rs := map[string]string{"k": string(val)}
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for long restriction value")
	}
	if rfce, ok := err.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request got %v", err)
	}
}
