package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/rfc"
)

// TestReplayProtection verifies that a second validation of the same token is rejected.
func TestReplayProtection(t *testing.T) {
	memMetrics := metrics.NewMemory()
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	// Allow alice to create delegations so test can proceed past authz layer.
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithMetrics(memMetrics), WithReplayProtection(100, time.Minute))

	req := DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"read"}, Duration: time.Minute}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}

	// First verification should succeed.
	ctx := WithSubject(context.Background(), "bob@example.com")
	vres, verr := svc.VerifyToken(ctx, resp.AuthToken)
	if verr != nil {
		t.Fatalf("first verify unexpected error: %v", verr)
	}
	if vres == nil || vres.DelegationID != resp.POA.ID {
		t.Fatalf("unexpected verification result")
	}

	// Second verification should be replay rejected.
	vres2, verr2 := svc.VerifyToken(ctx, resp.AuthToken)
	if verr2 == nil {
		t.Fatalf("expected error on replay but got none")
	}
	if rfce, ok := verr2.(rfc.RFCError); !ok || rfce.Code != rfc.ErrReplay {
		t.Fatalf("expected replay error got: %v", verr2)
	}
	if vres2 != nil {
		t.Fatalf("expected nil result on replay")
	}

	snap := memMetrics.SnapshotEx()
	if snap.ReplayHits != 1 {
		t.Fatalf("expected 1 replay hit got %d", snap.ReplayHits)
	}
	if snap.ReplayMisses != 1 {
		t.Fatalf("expected 1 replay miss got %d", snap.ReplayMisses)
	}
}
