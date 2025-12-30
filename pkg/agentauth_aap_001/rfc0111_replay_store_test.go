package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// TestReplayStorePrecedence ensures that when a ReplayStore is configured it is used for detection.
func TestReplayStorePrecedence(t *testing.T) {
	memMetrics := metrics.NewMemory()
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	store := newMockReplayStore(time.Minute)
	svc := NewService(auditLogger, authorizer, WithMetrics(memMetrics), WithReplayProtection(100, time.Minute), WithReplayStore(store))

	req := DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"read"}, Duration: time.Minute}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}

	// First verify should succeed (store miss)
	ctx := WithSubject(context.Background(), "bob@example.com")
	if _, err := svc.VerifyToken(ctx, resp.AuthToken); err != nil {
		t.Fatalf("first verify unexpected error: %v", err)
	}
	// Second verify should be replay via store.
	_, err2 := svc.VerifyToken(ctx, resp.AuthToken)
	if err2 == nil {
		t.Fatalf("expected replay error on second verify")
	}
	if e, ok := err2.(aap.RFCError); !ok || e.Code != aap.ErrReplay {
		t.Fatalf("expected replay error got %v", err2)
	}
	snap := memMetrics.SnapshotEx()
	if snap.ReplayHits != 1 || snap.ReplayMisses != 1 {
		t.Fatalf("unexpected metrics hits=%d misses=%d", snap.ReplayHits, snap.ReplayMisses)
	}
}
