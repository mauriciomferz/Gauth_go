package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
	"github.com/mauriciomferz/Gauth_go/pkg/rfc"
)

func TestDelegationStatusTransition(t *testing.T) {
	s, aliceCtx := setupTestServiceForStatus(t)
	// Bob is the grantee
	bobCtx := WithSubject(context.Background(), "bob")

	// 1. Create a PoA (Subject: alice)
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"read"},
		Duration: 1 * time.Hour,
	}

	resp, err := s.CreateDelegationCtx(aliceCtx, req)
	if err != nil {
		t.Fatalf("Failed to create delegation: %v", err)
	}
	poaID := resp.POA.ID
	token := resp.AuthToken

	// 2. Verify Token (Initial State: Active, Subject: bob)
	res, err := s.VerifyToken(bobCtx, token)
	if err != nil {
		t.Fatalf("Initial verification failed: %v", err)
	}
	if res.Suspended {
		t.Error("Initial state should not be suspended")
	}

	// 3. Suspend Delegation (Subject: alice)
	err = s.SuspendDelegation(aliceCtx, poaID, "alice", "temporary maintenance")
	if err != nil {
		t.Fatalf("Failed to suspend: %v", err)
	}

	// 4. Verify Token (State: Suspended, Subject: bob) - Expect failure
	_, err = s.VerifyToken(bobCtx, token)
	if err == nil {
		t.Error("Verification should fail for suspended delegation")
	} else {
		rfcErr, ok := err.(rfc.RFCError)
		if !ok || rfcErr.Code != rfc.ErrUnauthorized {
			t.Errorf("Expected ErrUnauthorized, got %v", err)
		}
	}

	// 5. Check Revocation Chain for suspension event
	events := s.revChain.Events()
	found := false
	for _, e := range events {
		if e.DelegationID == poaID && e.Reason == string(delegation.RevocationReasonSuspended) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Suspension event not found in revocation chain")
	}

	// 6. Resume Delegation (Subject: alice)
	err = s.ResumeDelegation(aliceCtx, poaID, "alice")
	if err != nil {
		t.Fatalf("Failed to resume: %v", err)
	}

	// 7. Verify Token (State: Active, Subject: bob) - Expect success
	res, err = s.VerifyToken(bobCtx, token)
	if err != nil {
		t.Fatalf("Verification failed after resumption: %v", err)
	}
	if res.Suspended {
		t.Error("State should no longer be suspended")
	}

	// 8. Check Revocation Chain for activation event
	events = s.revChain.Events()
	found = false
	for _, e := range events {
		if e.DelegationID == poaID && e.Reason == string(delegation.RevocationReasonActivated) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Activation event not found in revocation chain")
	}

	// 9. Negative Case: Suspend by non-grantor
	err = s.SuspendDelegation(aliceCtx, poaID, "bob", "unauthorized")
	if err == nil {
		t.Error("Suspend by non-grantor should fail")
	}

	// 10. Negative Case: Resume when already active
	err = s.ResumeDelegation(aliceCtx, poaID, "alice")
	if err == nil {
		t.Error("Resuming an already active delegation should fail")
	}
}

func setupTestServiceForStatus(t *testing.T) (*Service, context.Context) {
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	// Allow all for testing
	authorizer.AddPolicy(authz.Policy{ID: "allow-all", Subject: "*", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})

	s := NewService(auditLogger, authorizer)

	ctx := WithSubject(context.Background(), "alice")
	return s, ctx
}
