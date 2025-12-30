package gauth_aap_001

import (
	"context"
	"errors"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// mockAuthorizer always allows
type mockAuthorizer struct{}

func (m mockAuthorizer) Authorize(_ context.Context, _ authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true}, nil
}

func (m mockAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	return []authz.Permission{{Resource: "*", Actions: []string{"*"}, Granted: true}}, nil
}

// failingAnchorClient returns error for every attempt
type failingAnchorClient struct{}

func (f failingAnchorClient) Anchor(hash string) error { return errors.New("anchor failed") }

// TestNoopAnchorSuccess ensures NoopAnchorClient increments attempts but not failures.
func TestNoopAnchorSuccess(t *testing.T) {
	mem := imetrics.NewMemory()
	svc := NewService(audit.NewMemoryLogger(nil), mockAuthorizer{}, WithMetrics(mem), WithAnchorClient(NoopAnchorClient{}))
	svc.WithClock(func() time.Time { return time.Unix(0, 0) })

	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Minute}
	if _, err := svc.CreateDelegation(req); err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	list, _ := svc.ListDelegations("alice")
	if len(list) == 0 {
		t.Fatalf("expected at least one delegation")
	}
	if err := svc.RevokeDelegation(list[0].ID, "alice"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	// Extract only anchor metrics (positions 15 and 16 from Snapshot)
	var attempts, failures uint64
	_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, attempts, failures, _, _, _, _, _, _ = mem.Snapshot()
	if attempts == 0 {
		t.Fatalf("expected >0 anchor attempts got %d", attempts)
	}
	if failures != 0 {
		t.Fatalf("expected 0 anchor failures got %d", failures)
	}
}

// TestAnchorFailures ensures failing AnchorClient increments both attempts and failures.
func TestAnchorFailures(t *testing.T) {
	mem := imetrics.NewMemory()
	svc := NewService(audit.NewMemoryLogger(nil), mockAuthorizer{}, WithMetrics(mem), WithAnchorClient(failingAnchorClient{}))
	svc.WithClock(func() time.Time { return time.Unix(0, 0) })
	req := DelegationRequest{Grantor: "carol", Grantee: "dave", Scope: []string{"write"}, Duration: time.Minute}
	if _, err := svc.CreateDelegation(req); err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	list, _ := svc.ListDelegations("carol")
	if len(list) == 0 {
		t.Fatalf("expected delegation")
	}
	if err := svc.RevokeDelegation(list[0].ID, "carol"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	// Extract only anchor metrics (positions 15 and 16 from Snapshot)
	var attempts, failures uint64
	_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, attempts, failures, _, _, _, _, _, _ = mem.Snapshot()
	if attempts == 0 {
		t.Fatalf("expected attempts >0 got %d", attempts)
	}
	if failures == 0 {
		t.Fatalf("expected failures >0 got %d", failures)
	}
}
