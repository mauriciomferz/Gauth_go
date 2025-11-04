package rfc0111

import (
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
)

// TestPersistenceDurability verifies PoA entries persist across service restarts when Bolt repository is enabled.
func TestPersistenceDurability(t *testing.T) {
	path := t.TempDir() + "/poa.db"
	os.Setenv("GAUTH_PERSIST_PATH", path)
	defer os.Unsetenv("GAUTH_PERSIST_PATH")

	// First service instance (create)
	memLogger := audit.NewMemoryLogger(nil)
	svc1 := NewService(memLogger, &allowAllAuthorizer{})
	resp, err := svc1.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	id := resp.POA.ID

	// Close underlying bolt repo if type matches
	if br, ok := svc1.repo.(*BoltRepository); ok {
		if err := br.Close(); err != nil {
			t.Fatalf("close repo failed: %v", err)
		}
	}

	// Simulate restart (new service instance)
	svc2 := NewService(audit.NewMemoryLogger(nil), &allowAllAuthorizer{})
	poa2, ok := svc2.repo.Get(id)
	if !ok || poa2 == nil {
		t.Fatalf("poa not found after restart")
	}
	if poa2.Grantor != "alice" || poa2.Grantee != "bob" {
		t.Fatalf("poa data mismatch after restart: %+v", poa2)
	}

	// Ensure creating another delegation reuses same DB file (incremental persistence)
	resp2, err := svc2.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"write"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if resp2.POA.ID == id {
		t.Fatalf("expected distinct ID for second POA")
	}

	// Reopen again to verify both entries remain
	if br2, ok := svc2.repo.(*BoltRepository); ok {
		if err := br2.Close(); err != nil {
			t.Fatalf("close repo2 failed: %v", err)
		}
	}
	svc3 := NewService(audit.NewMemoryLogger(nil), &allowAllAuthorizer{})
	found1, ok1 := svc3.repo.Get(id)
	found2, ok2 := svc3.repo.Get(resp2.POA.ID)
	if !ok1 || !ok2 || found1 == nil || found2 == nil {
		t.Fatalf("expected both POAs present after second restart")
	}
}
