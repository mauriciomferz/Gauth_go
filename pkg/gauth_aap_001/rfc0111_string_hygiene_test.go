package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/rfc"
)

func newHygieneSvc() *Service {
	a := audit.NewMemoryLogger(nil)
	az := authz.NewMemoryAuthorizer()
	az.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	return NewService(a, az)
}

func TestScopeItemEmptyRejected(t *testing.T) {
	svc := newHygieneSvc()
	_, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{""}, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for empty scope item")
	}
}

func TestScopeItemControlCharRejected(t *testing.T) {
	svc := newHygieneSvc()
	bad := "read" + string(rune(0x7))
	_, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{bad}, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for control char in scope")
	}
}

func TestRestrictionEmptyKeyRejected(t *testing.T) {
	svc := newHygieneSvc()
	rs := map[string]string{"": "v"}
	_, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for empty restriction key")
	}
}

func TestRestrictionEmptyValueRejected(t *testing.T) {
	svc := newHygieneSvc()
	rs := map[string]string{"k": ""}
	_, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for empty restriction value")
	}
}

func TestRestrictionControlCharsRejected(t *testing.T) {
	svc := newHygieneSvc()
	rs := map[string]string{"k": "val" + string(rune(0x1F))}
	_, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for control char in restriction value")
	}
}

func TestValidUnicodeAccepted(t *testing.T) {
	svc := newHygieneSvc()
	rs := map[string]string{"note": "café"}
	del, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"léér"}, Restrictions: rs, Duration: time.Minute})
	if err != nil {
		t.Fatalf("expected success for printable unicode: %v", err)
	}
	if del == nil {
		t.Fatalf("missing delegation result")
	}
}

func TestRFCErrorCodeForControlChar(t *testing.T) {
	svc := newHygieneSvc()
	bad := "*" + string(rune(0x00))
	_, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{bad}, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error")
	}
	if rfce, ok := err.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request RFC error got %v", err)
	}
}
