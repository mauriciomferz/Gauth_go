package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/aap"
)

// TestSuspendDelegation_Success validates successful suspension of an active delegation.
func TestSuspendDelegation_Success(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create delegation
	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write"})
	if poa.Status != POAStatusActive {
		t.Fatalf("expected active status, got %s", poa.Status)
	}

	// Suspend
	err := svc.SuspendDelegation(ctx, poa.ID, "alice", "temporarily paused for security review")
	if err != nil {
		t.Fatalf("suspend failed: %v", err)
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify status changed
	updated, ok := svc.repo.Get(poa.ID)
	if !ok {
		t.Fatal("delegation not found after suspension")
	}
	if updated.Status != POAStatusSuspended {
		t.Errorf("expected suspended status, got %s", updated.Status)
	}

	// Verify audit event
	events := svc.AuditEvents()
	found := false
	for _, ev := range events {
		if ev.Action == "suspend_delegation" && ev.Object == poa.ID {
			found = true
			if ev.Subject != "alice" {
				t.Errorf("expected subject alice, got %s", ev.Subject)
			}
			if ev.Metadata["reason"] != "temporarily paused for security review" {
				t.Errorf("expected reason in metadata, got %v", ev.Metadata)
			}
			if ev.Metadata["prev_status"] != string(POAStatusActive) {
				t.Errorf("expected prev_status active, got %v", ev.Metadata["prev_status"])
			}
			break
		}
	}
	if !found {
		t.Error("suspend_delegation audit event not found")
	}
}

// TestSuspendDelegation_InvalidStatus validates rejection of suspension for non-active delegations.
func TestSuspendDelegation_InvalidStatus(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	tests := []struct {
		name   string
		status POAStatus
	}{
		{"suspended", POAStatusSuspended},
		{"revoked", POAStatusRevoked},
		{"terminated", POAStatusTerminated},
		{"expired", POAStatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poa := createTestDelegation(svc, "alice", "bob", []string{"read"})
			poa.Status = tt.status
			_ = svc.repo.Update(poa)

			err := svc.SuspendDelegation(ctx, poa.ID, "alice", "test")
			if err == nil {
				t.Fatal("expected error for invalid status, got nil")
			}
		/aapErr, ok := err. aap.RFCError)
			if !ok ||/aapErr.Code != aap.ErrInvalidRequest {
				t.Errorf("expected ErrInvalidRequest, got %v", err)
			}
		})
	}
}

// TestSuspendDelegation_Unauthorized validates only grantor can suspend.
func TestSuspendDelegation_Unauthorized(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	poa := createTestDelegation(svc, "alice", "bob", []string{"read"})

	// Try suspending as non-grantor
	err := svc.SuspendDelegation(ctx, poa.ID, "charlie", "unauthorized attempt")
	if err == nil {
		t.Fatal("expected error for non-grantor, got nil")
	}
/aapErr, ok := err. aap.RFCError)
	if !ok ||/aapErr.Code != aap.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	// Verify status unchanged
	updated, _ := svc.repo.Get(poa.ID)
	if updated.Status != POAStatusActive {
		t.Errorf("status should remain active, got %s", updated.Status)
	}
}

// TestResumeDelegation_Success validates successful resumption of a suspended delegation.
func TestResumeDelegation_Success(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create and suspend delegation
	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write"})
	_ = svc.SuspendDelegation(ctx, poa.ID, "alice", "test")

	// Verify suspended
	suspended, _ := svc.repo.Get(poa.ID)
	if suspended.Status != POAStatusSuspended {
		t.Fatalf("expected suspended, got %s", suspended.Status)
	}

	// Resume
	err := svc.ResumeDelegation(ctx, poa.ID, "alice")
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify status changed back to active
	resumed, ok := svc.repo.Get(poa.ID)
	if !ok {
		t.Fatal("delegation not found after resumption")
	}
	if resumed.Status != POAStatusActive {
		t.Errorf("expected active status, got %s", resumed.Status)
	}

	// Verify audit event
	events := svc.AuditEvents()
	found := false
	for _, ev := range events {
		if ev.Action == "resume_delegation" && ev.Object == poa.ID {
			found = true
			if ev.Subject != "alice" {
				t.Errorf("expected subject alice, got %s", ev.Subject)
			}
			if ev.Metadata["prev_status"] != string(POAStatusSuspended) {
				t.Errorf("expected prev_status suspended, got %v", ev.Metadata["prev_status"])
			}
			break
		}
	}
	if !found {
		t.Error("resume_delegation audit event not found")
	}
}

// TestResumeDelegation_InvalidStatus validates rejection of resumption for non-suspended delegations.
func TestResumeDelegation_InvalidStatus(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	tests := []struct {
		name   string
		status POAStatus
	}{
		{"active", POAStatusActive},
		{"revoked", POAStatusRevoked},
		{"terminated", POAStatusTerminated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poa := createTestDelegation(svc, "alice", "bob", []string{"read"})
			poa.Status = tt.status
			_ = svc.repo.Update(poa)

			err := svc.ResumeDelegation(ctx, poa.ID, "alice")
			if err == nil {
				t.Fatal("expected error for invalid status, got nil")
			}
		/aapErr, ok := err. aap.RFCError)
			if !ok ||/aapErr.Code != aap.ErrInvalidRequest {
				t.Errorf("expected ErrInvalidRequest, got %v", err)
			}
		})
	}
}

// TestUpdateDelegationScope_Success validates successful scope reduction.
func TestUpdateDelegationScope_Success(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create delegation with broad scope
	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write", "delete", "admin"})

	// Reduce scope
	newScope := []string{"read", "write"}
	err := svc.UpdateDelegationScope(ctx, poa.ID, "alice", newScope, "security policy changed")
	if err != nil {
		t.Fatalf("update scope failed: %v", err)
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify scope changed
	updated, ok := svc.repo.Get(poa.ID)
	if !ok {
		t.Fatal("delegation not found after scope update")
	}
	if len(updated.Scope) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(updated.Scope))
	}
	scopeSet := make(map[string]bool)
	for _, s := range updated.Scope {
		scopeSet[s] = true
	}
	if !scopeSet["read"] || !scopeSet["write"] {
		t.Errorf("expected read and write permissions, got %v", updated.Scope)
	}

	// Verify scope history
	if historyJSON, ok := updated.Restrictions["__scope_history"]; ok {
		t.Logf("Scope history recorded: %s", historyJSON)
	} else {
		t.Error("scope history not recorded")
	}

	// Verify audit event
	events := svc.AuditEvents()
	found := false
	for _, ev := range events {
		if ev.Action == "update_delegation_scope" && ev.Object == poa.ID {
			found = true
			if ev.Metadata["reason"] != "security policy changed" {
				t.Errorf("expected reason in metadata, got %v", ev.Metadata)
			}
			break
		}
	}
	if !found {
		t.Error("update_delegation_scope audit event not found")
	}
}

// TestUpdateDelegationScope_InvalidSubset validates rejection when new scope is not a subset.
func TestUpdateDelegationScope_InvalidSubset(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write"})

	// Try expanding scope (should fail)
	newScope := []string{"read", "write", "delete"}
	err := svc.UpdateDelegationScope(ctx, poa.ID, "alice", newScope, "attempt to expand")
	if err == nil {
		t.Fatal("expected error for scope expansion, got nil")
	}
/aapErr, ok := err. aap.RFCError)
	if !ok ||/aapErr.Code != aap.ErrInvalidRequest {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}

	// Verify scope unchanged
	unchanged, _ := svc.repo.Get(poa.ID)
	if len(unchanged.Scope) != 2 {
		t.Errorf("scope should remain unchanged with 2 permissions, got %d", len(unchanged.Scope))
	}
}

// TestUpdateDelegationScope_EmptyScope validates rejection of empty scope.
func TestUpdateDelegationScope_EmptyScope(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write"})

	// Try empty scope (should fail - use revocation instead)
	err := svc.UpdateDelegationScope(ctx, poa.ID, "alice", []string{}, "invalid empty")
	if err == nil {
		t.Fatal("expected error for empty scope, got nil")
	}
/aapErr, ok := err. aap.RFCError)
	if !ok ||/aapErr.Code != aap.ErrInvalidRequest {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}
}

// TestUpdateDelegationScope_NoChange validates rejection when scope is identical.
func TestUpdateDelegationScope_NoChange(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write"})

	// Try same scope (should fail)
	err := svc.UpdateDelegationScope(ctx, poa.ID, "alice", []string{"read", "write"}, "no change")
	if err == nil {
		t.Fatal("expected error for identical scope, got nil")
	}
/aapErr, ok := err. aap.RFCError)
	if !ok ||/aapErr.Code != aap.ErrInvalidRequest {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}
}

// TestUpdateDelegationScope_SuspendedDelegation validates scope update on suspended delegation.
func TestUpdateDelegationScope_SuspendedDelegation(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create, then suspend
	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write", "delete"})
	_ = svc.SuspendDelegation(ctx, poa.ID, "alice", "test")

	// Update scope while suspended (should succeed)
	newScope := []string{"read"}
	err := svc.UpdateDelegationScope(ctx, poa.ID, "alice", newScope, "reduce during suspension")
	if err != nil {
		t.Fatalf("update scope on suspended delegation failed: %v", err)
	}

	// Verify scope changed
	updated, _ := svc.repo.Get(poa.ID)
	if len(updated.Scope) != 1 || updated.Scope[0] != "read" {
		t.Errorf("expected scope [read], got %v", updated.Scope)
	}

	// Verify still suspended
	if updated.Status != POAStatusSuspended {
		t.Errorf("status should remain suspended, got %s", updated.Status)
	}
}

// TestVerifyToken_SuspendedDelegation validates token verification rejects suspended delegations.
func TestVerifyToken_SuspendedDelegation(t *testing.T) {
	t.Skip("Skipping token generation test - requires full PASETO setup")
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create active delegation and generate token
	poa := createTestDelegation(svc, "alice", "bob", []string{"read", "write"})

	// Generate real token using CreateDelegation
	req := DelegationRequest{
		Grantor:      poa.Grantor,
		Grantee:      poa.Grantee,
		Scope:        poa.Scope,
		Restrictions: poa.Restrictions,
		Duration:     24 * time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}
	token := resp.AuthToken

	// Verify token works when active
	result, err := svc.VerifyToken(ctx, token)
	if err != nil {
		t.Fatalf("verify active token failed: %v", err)
	}
	if result.Suspended {
		t.Error("active delegation should not be marked suspended")
	}

	// Suspend delegation
	_ = svc.SuspendDelegation(ctx, resp.POA.ID, "alice", "test suspension")

	// Verify token now rejected
	result, err = svc.VerifyToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for suspended delegation, got nil")
	}
/aapErr, ok := err. aap.RFCError)
	if !ok ||/aapErr.Code != aap.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if result != nil && !result.Suspended {
		t.Error("result should indicate suspension")
	}
}

// TestSuspensionResumptionCycle validates full lifecycle.
func TestSuspensionResumptionCycle(t *testing.T) {
	t.Skip("Skipping token generation test - requires full PASETO setup")
	ctx := WithSubject(context.Background(), "bob")
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create delegation with token
	req := DelegationRequest{
		Grantor:      "alice",
		Grantee:      "bob",
		Scope:        []string{"read", "write", "delete"},
		Restrictions: make(map[string]string),
		Duration:     24 * time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}
	token := resp.AuthToken
	poa := resp.POA

	// 1. Verify active token works
	_, err = svc.VerifyToken(ctx, token)
	if err != nil {
		t.Fatalf("active token verification failed: %v", err)
	}

	// 2. Suspend
	_ = svc.SuspendDelegation(ctx, poa.ID, "alice", "incident response")

	// 3. Verify token rejected
	_, err = svc.VerifyToken(ctx, token)
	if err == nil {
		t.Fatal("suspended token should be rejected")
	}

	// 4. Reduce scope while suspended
	_ = svc.UpdateDelegationScope(ctx, poa.ID, "alice", []string{"read"}, "minimize permissions")

	// 5. Resume
	_ = svc.ResumeDelegation(ctx, poa.ID, "alice")

	// 6. Verify token works again with reduced scope
	result, err := svc.VerifyToken(ctx, token)
	if err != nil {
		t.Fatalf("resumed token verification failed: %v", err)
	}

	// Check scope was reduced in POA (token still has old scope in envelope)
	updated, _ := svc.repo.Get(poa.ID)
	if len(updated.Scope) != 1 || updated.Scope[0] != "read" {
		t.Errorf("expected reduced scope [read], got %v", updated.Scope)
	}
	if result.Suspended {
		t.Error("resumed delegation should not be marked suspended")
	}
}

// Helper functions

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	// Create permissive authorizer for testing
	authzMem := authz.NewMemoryAuthorizer()
	authzMem.AddPolicy(authz.Policy{ID: "allow-all", Subject: "*", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})

	// Create service with memory-backed components
	svc := NewService(audit.NewMemoryLogger(nil), authzMem)

	cleanup := func() {
		// Cleanup if needed
	}

	return svc, cleanup
}

func createTestDelegation(svc *Service, grantor, grantee string, scope []string) *PowerOfAttorney {
	now := svc.nowFn()
	poa := &PowerOfAttorney{
		ID:           "test-" + grantor + "-" + grantee + "-" + time.Now().Format("20060102150405"),
		Version:      1,
		Grantor:      grantor,
		Grantee:      grantee,
		Scope:        scope,
		Restrictions: make(map[string]string),
		ValidFrom:    now,
		ValidUntil:   now.Add(24 * time.Hour),
		Status:       POAStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = svc.repo.Create(poa)
	return poa
}
