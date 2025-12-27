package gauth_rfc_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

type MockAuthorizer struct{}

func (m *MockAuthorizer) Authorize(ctx context.Context, request authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true}, nil
}

func (m *MockAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	return nil, nil
}

func TestGranularRevocation(t *testing.T) {
	// Setup service
	svc := setupService()
	grantor := "alice"
	grantee := "bob"

	// 1. Create PoA with multiple scopes
	poaReq := DelegationRequest{
		Grantor:  grantor,
		Grantee:  grantee,
		Scope:    []string{"payment", "audit", "read"},
		Duration: 1 * time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(context.Background(), poaReq)
	if err != nil {
		t.Fatalf("failed to create delegation: %v", err)
	}
	poa := &resp.POA

	// 2. Issue tokens for different scopes
	tokPayment := issueTokenForScope(svc, poa, []string{"payment"})
	tokAudit := issueTokenForScope(svc, poa, []string{"audit"})

	// 3. Verify tokens work initially
	ctx := context.WithValue(context.Background(), ctxKeySubject, grantee)
	_, err = svc.VerifyToken(ctx, tokPayment)
	if err != nil {
		t.Errorf("payment token verify failed: %v", err)
	}
	_, err = svc.VerifyToken(ctx, tokAudit)
	if err != nil {
		t.Errorf("audit token verify failed: %v", err)
	}

	// 4. Revoke "payment" authority
	err = svc.RevokeAuthorityCtx(context.Background(), poa.ID, grantor, []string{"payment"})
	if err != nil {
		t.Fatalf("failed to revoke payment authority: %v", err)
	}

	// 5. Verify payment token is now rejected
	_, err = svc.VerifyToken(ctx, tokPayment)
	if err == nil {
		t.Error("expected payment token to be rejected but it passed")
	} else if err.Error() != "unauthorized: scope payment has been revoked for this delegation" {
		t.Errorf("unexpected error for revoked payment token: %v", err)
	}

	// 6. Verify audit token still works
	_, err = svc.VerifyToken(ctx, tokAudit)
	if err != nil {
		t.Errorf("audit token verify failed after partial revocation: %v", err)
	}

	// 7. Attempt to revoke non-existent scope
	err = svc.RevokeAuthorityCtx(context.Background(), poa.ID, grantor, []string{"ghost"})
	if err == nil {
		t.Error("expected error revoking non-existent scope")
	}

	// 8. Attempt unauthorized revocation (by grantee)
	err = svc.RevokeAuthorityCtx(context.Background(), poa.ID, grantee, []string{"read"})
	if err == nil {
		t.Error("expected error for unauthorized revocation")
	}
}

func TestGranularRevocation_MultiScopeToken(t *testing.T) {
	// Setup service
	svc := setupService()
	grantor := "alice"
	grantee := "bob"

	// 1. Create PoA
	poaReq := DelegationRequest{
		Grantor:  grantor,
		Grantee:  grantee,
		Scope:    []string{"a", "b", "c"},
		Duration: 1 * time.Hour,
	}
	resp, _ := svc.CreateDelegationCtx(context.Background(), poaReq)
	poa := &resp.POA

	// 2. Issue token with multiple scopes
	tokMulti := issueTokenForScope(svc, poa, []string{"a", "b"})

	// 3. Revoke one of them
	svc.RevokeAuthorityCtx(context.Background(), poa.ID, grantor, []string{"a"})

	// 4. Verify whole token is rejected
	ctx := context.WithValue(context.Background(), ctxKeySubject, grantee)
	_, err := svc.VerifyToken(ctx, tokMulti)
	if err == nil {
		t.Error("expected multi-scope token to be rejected if one scope is revoked")
	}
}

func setupService() *Service {
	return NewService(audit.NewMemoryLogger(nil), &MockAuthorizer{})
}

func issueTokenForScope(svc *Service, poa *PowerOfAttorney, scope []string) string {
	origScope := poa.Scope
	poa.Scope = scope
	defer func() { poa.Scope = origScope }()
	return generateAuthToken(svc, poa)
}
