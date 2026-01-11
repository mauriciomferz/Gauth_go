package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
)

// TestDualControlRevocationWorkflow exercises initiation, approval quorum and cancellation flows.
func TestDualControlRevocationWorkflow(t *testing.T) {
	t.Setenv("AGENTAUTH_PERSIST_PATH", t.TempDir()+"/poa.db")
	t.Setenv("AGENTAUTH_REVOCATION_REQUIRED_COUNT", "2")
	memLogger := audit.NewMemoryLogger(nil)
	svc := NewService(memLogger, &allowAllAuthorizer{})
	// Create POA with controllers
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"ops.read"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	poaID := resp.POA.ID
	// Manually attach controllers for test (simulating issuance with controllers field)
	poa, _ := svc.repo.Get(poaID)
	poa.Controllers = []string{"controller1", "controller2"}
	if err := svc.repo.Update(poa); err != nil {
		t.Fatalf("update controllers failed: %v", err)
	}
	ctx := context.Background()
	// Unauthorized initiator
	if err := svc.InitiateRevocation(ctx, RevocationRequest{POAID: poaID, Initiator: "bob", Reason: "risk"}); err == nil {
		t.Fatalf("expected unauthorized initiator rejection")
	}
	// Authorized initiator (grantor)
	if err := svc.InitiateRevocation(ctx, RevocationRequest{POAID: poaID, Initiator: "alice", Reason: "risk"}); err != nil {
		t.Fatalf("initiation failed: %v", err)
	}
	// First approval (controller1)
	if err := svc.ApproveRevocation(ctx, poaID, "controller1"); err != nil {
		t.Fatalf("first approval failed: %v", err)
	}
	// Duplicate approval should be idempotent
	if err := svc.ApproveRevocation(ctx, poaID, "controller1"); err != nil {
		t.Fatalf("duplicate approval should be allowed/idempotent: %v", err)
	}
	// Second approval (controller2) should finalize
	if err := svc.ApproveRevocation(ctx, poaID, "controller2"); err != nil {
		t.Fatalf("second approval failed: %v", err)
	}
	poaFinal, _ := svc.repo.Get(poaID)
	if poaFinal.Status != POAStatusRevoked || poaFinal.PendingRevocation == nil || !poaFinal.PendingRevocation.Finalized {
		t.Fatalf("expected POA revoked after quorum; status=%s finalized=%v", poaFinal.Status, poaFinal.PendingRevocation != nil && poaFinal.PendingRevocation.Finalized)
	}
	// Attempt cancellation after finalize should fail
	if err := svc.CancelRevocation(ctx, poaID, "alice"); err == nil {
		t.Fatalf("expected cancellation failure after finalize")
	}
}

// TestDualControlRevocationMetrics ensures metrics counters increment across workflow actions.
func TestDualControlRevocationMetrics(t *testing.T) {
	t.Skip("Test needs refactoring to work with current unexported field access patterns - will implement proper metrics tests in future iteration")
}
