package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func TestServiceKeyRotationAuto(t *testing.T) {
	mem := metrics.NewMemory()
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})

	// Create service with 50ms rotation interval
	svc := NewService(audit.NewMemoryLogger(nil), ma, WithMetrics(mem), WithKeyRotation(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start lifecycle (scheduler)
	svc.StartLifecycle(ctx)

	// Issue Token 1
	req := DelegationRequest{Grantor: "p@example.com", Grantee: "q@example.com", Scope: []string{"x"}, Duration: time.Minute}
	res1, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("issue 1 failed: %v", err)
	}

	// Parse Token 1 to get kid/signature (assuming V1 for simplicity or V2 if default)
	// We can check the active key ID directly on the keyring too for robustness
	kid1 := svc.keyRing.Active().ID

	// Wait for > interval (plus buffer)
	time.Sleep(100 * time.Millisecond)

	// Issue Token 2
	res2, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("issue 2 failed: %v", err)
	}

	kid2 := svc.keyRing.Active().ID

	if kid1 == kid2 {
		t.Fatalf("expected key rotation, but active key ID remained %s", kid1)
	}

	// Double check that res2 is actually signed by kid2
	// (Simulate validation with service key ring which should have both)
	if _, err := svc.VerifyToken(WithSubject(ctx, "q@example.com"), res2.AuthToken); err != nil {
		t.Errorf("failed to verify token signed by new key: %v", err)
	}

	// Token 1 should still be valid (in grace period previous keys)
	if _, err := svc.VerifyToken(WithSubject(ctx, "q@example.com"), res1.AuthToken); err != nil {
		t.Errorf("failed to verify token signed by old key (grace period): %v", err)
	}
}

// TestWithKeyRotationOption verifies that option correctly installs scheduler
func TestWithKeyRotationOption(t *testing.T) {
	ma := authz.NewMemoryAuthorizer()
	svc := NewService(audit.NewMemoryLogger(nil), ma, WithKeyRotation(time.Hour))
	if svc.keyRotationScheduler == nil {
		t.Fatal("scheduler not initialized with option")
	}
}
