package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/rfc"
)

// helper to build service with allow policy
func testService() *Service {
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	ma.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice", Resource: "poa", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	return NewService(audit.NewMemoryLogger(nil), ma)
}

func newDelegation(s *Service, dur time.Duration) (*DelegationResponse, error) {
	return s.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:pay", "read"}, Duration: dur, Restrictions: map[string]string{"max_amount": "50"}})
}

func expectCode(t *testing.T, err error, code rfc.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %s got nil", code)
	}
	rerr, ok := err.(rfc.RFCError)
	if !ok {
		t.Fatalf("expected RFCError got %T", err)
	}
	if rerr.Code != code {
		t.Fatalf("expected code %s got %s", code, rerr.Code)
	}
}

// TestErrorMapping covers revoked, expired, scope_violation, restriction_exceeded.
func TestErrorMapping(t *testing.T) {
	svc := testService()
	// 1. Scope violation
	dr, err := newDelegation(svc, 5*time.Minute)
	if err != nil {
		t.Fatalf("delegation create: %v", err)
	}
	err = svc.ValidateDelegationCtx(context.Background(), dr.POA.ID, "bob", "admin:delete")
	expectCode(t, err, rfc.ErrScopeViolation)

	// 2. Restriction exceeded
	ctx := WithRequestedAmount(context.Background(), "75")
	err = svc.ValidateDelegationCtx(ctx, dr.POA.ID, "bob", "transaction:pay")
	expectCode(t, err, rfc.ErrRestrictionExceeded)

	// 3. Expired
	svc.WithClock(func() time.Time { return time.Now().Add(10 * time.Minute) })
	err = svc.ValidateDelegationCtx(context.Background(), dr.POA.ID, "bob", "read")
	expectCode(t, err, rfc.ErrExpired)

	// 4. Revoked
	// Need fresh delegation (not expired) then revoke.
	svc.WithClock(time.Now) // reset clock
	dr2, err := newDelegation(svc, 2*time.Minute)
	if err != nil {
		t.Fatalf("delegation create2: %v", err)
	}
	// Add explicit policy for this specific delegation resource revoke (resource=delegation ID)
	ma := svc.authz.(*authz.MemoryAuthorizer)
	ma.AddPolicy(authz.Policy{ID: "allow-revoke-specific", Subject: "alice", Resource: dr2.POA.ID, Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	if err2 := svc.RevokeDelegation(dr2.POA.ID, "alice"); err2 != nil {
		t.Fatalf("revoke: %v", err2)
	}
	err = svc.ValidateDelegationCtx(context.Background(), dr2.POA.ID, "bob", "read")
	expectCode(t, err, rfc.ErrRevoked)
}
