package rfc0111

import (
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
)

// TestSubDelegationDepth verifies parent linkage and depth derivation plus max depth enforcement.
func TestSubDelegationDepth(t *testing.T) {
	// Use temporary BoltDB path for persistence testing
	path := t.TempDir() + "/poa.db"
	os.Setenv("GAUTH_PERSIST_PATH", path)
	os.Setenv("GAUTH_MAX_DELEGATION_DEPTH", "3") // cap depth
	defer os.Unsetenv("GAUTH_PERSIST_PATH")
	defer os.Unsetenv("GAUTH_MAX_DELEGATION_DEPTH")

	// Provide a non-nil audit logger
	memLogger := audit.NewMemoryLogger(nil)
	svc := NewService(memLogger, &allowAllAuthorizer{})
	// Root delegation (depth 0)
	rootResp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("root create failed: %v", err)
	}
	if rootResp.POA.ParentPOAID != "" || rootResp.POA.Depth != 0 {
		t.Fatalf("expected root depth=0 no parent, got depth=%d parent=%s", rootResp.POA.Depth, rootResp.POA.ParentPOAID)
	}

	// Child depth 1
	child1, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"read"}, Duration: time.Hour, ParentPOAID: rootResp.POA.ID})
	if err != nil {
		t.Fatalf("child1 create failed: %v", err)
	}
	if child1.POA.Depth != 1 || child1.POA.ParentPOAID != rootResp.POA.ID {
		t.Fatalf("expected child1 depth=1 parent=%s got depth=%d parent=%s", rootResp.POA.ID, child1.POA.Depth, child1.POA.ParentPOAID)
	}

	// Child2 depth 2
	child2, err := svc.CreateDelegation(DelegationRequest{Grantor: "carol", Grantee: "dave", Scope: []string{"read"}, Duration: time.Hour, ParentPOAID: child1.POA.ID})
	if err != nil {
		t.Fatalf("child2 create failed: %v", err)
	}
	if child2.POA.Depth != 2 {
		t.Fatalf("expected child2 depth=2 got %d", child2.POA.Depth)
	}

	// Child3 depth 3 (allowed since max depth=3)
	child3, err := svc.CreateDelegation(DelegationRequest{Grantor: "dave", Grantee: "erin", Scope: []string{"read"}, Duration: time.Hour, ParentPOAID: child2.POA.ID})
	if err != nil {
		t.Fatalf("child3 create failed: %v", err)
	}
	if child3.POA.Depth != 3 {
		t.Fatalf("expected child3 depth=3 got %d", child3.POA.Depth)
	}

	// Child4 depth 4 (should exceed and fail)
	_, err = svc.CreateDelegation(DelegationRequest{Grantor: "erin", Grantee: "frank", Scope: []string{"read"}, Duration: time.Hour, ParentPOAID: child3.POA.ID})
	if err == nil {
		t.Fatalf("expected depth exceed error creating child4")
	}
}
