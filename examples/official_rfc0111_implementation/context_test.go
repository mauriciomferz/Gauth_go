package main

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"
	"github.com/mauriciomferz/Gauth_go/pkg/testutil"
)

func TestCreateDelegationCanceled(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := gauth_rfc_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_rfc_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: time.Hour}
	if _, err := svc.CreateDelegationCtx(ctx, req); err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}
}

func TestValidateDelegationDeadline(t *testing.T) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice@example.com", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	svc := gauth_rfc_001.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
	req := gauth_rfc_001.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()
	if err := svc.ValidateDelegationCtx(ctx, resp.POA.ID, req.Grantee, "transaction:execute"); err == nil {
		t.Fatalf("expected deadline exceeded error")
	}
}
