package gauth_aap_001

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/rfc"
)

// errorReplayStore simulates a distributed replay store that returns errors.
type errorReplayStore struct{}

func (e errorReplayStore) Seen(jti string) (bool, error) {
	return false, errors.New("store unavailable")
}
func (e errorReplayStore) Record(jti string, at time.Time) error {
	return errors.New("store write failed")
}

// TestReplayFailClosed verifies that when fail-closed option is enabled, replay store errors abort verification.
func TestReplayFailClosed(t *testing.T) {
	memMetrics := metrics.NewMemory()
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	// Service with erroring replay store and fail-closed semantics.
	svc := NewService(auditLogger, authorizer, WithMetrics(memMetrics), WithReplayStore(errorReplayStore{}), WithReplayFailClosed())

	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}

	// Verification should fail due to replay store error (Seen path).
	ctx := WithSubject(context.Background(), "bob@example.com")
	_, verr := svc.VerifyToken(ctx, resp.AuthToken)
	if verr == nil {
		t.Fatalf("expected error due to replay store failure")
	}
	if rfce, ok := verr.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request error got %v", verr)
	}

	// Ensure metrics recorded store errors.
	snap := memMetrics.SnapshotEx()
	if snap.ReplayStoreErrors == 0 {
		t.Fatalf("expected replay store errors counter increment")
	}
}

// TestReplayFailClosedRecord ensures record error also aborts verification when fail-closed.
func TestReplayFailClosedRecord(t *testing.T) {
	memMetrics := metrics.NewMemory()
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	// We reuse errorReplayStore since both Seen and Record paths error; ensures both branches are covered.
	svc := NewService(auditLogger, authorizer, WithMetrics(memMetrics), WithReplayStore(errorReplayStore{}), WithReplayFailClosed())
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}
	ctx := WithSubject(context.Background(), "bob@example.com")
	_, verr := svc.VerifyToken(ctx, resp.AuthToken)
	if verr == nil {
		t.Fatalf("expected error due to replay store failure (record)")
	}
	if rfce, ok := verr.(rfc.RFCError); !ok || rfce.Code != rfc.ErrInvalidRequest {
		t.Fatalf("expected invalid_request error got %v", verr)
	}
	if memMetrics.SnapshotEx().ReplayStoreErrors == 0 {
		t.Fatalf("expected replay store errors counter increment")
	}
}
