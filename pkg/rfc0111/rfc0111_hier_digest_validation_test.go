package rfc0111

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
)

// TestHierDigestParentValidation ensures parent digest mismatch causes validation failure.
func TestHierDigestParentValidation(t *testing.T) {
	os.Setenv("GAUTH_ENABLE_HIER_DIGEST", "1")
	defer os.Unsetenv("GAUTH_ENABLE_HIER_DIGEST")
	path := t.TempDir() + "/poa.db"
	os.Setenv("GAUTH_PERSIST_PATH", path)
	defer os.Unsetenv("GAUTH_PERSIST_PATH")
	memLogger := audit.NewMemoryLogger(nil)
	svc := NewService(memLogger, &allowAllAuthorizer{})
	rootResp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"finance.read"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("root create failed: %v", err)
	}
	childResp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"finance.read"}, Duration: time.Hour, ParentPOAID: rootResp.POA.ID})
	if err != nil {
		t.Fatalf("child create failed: %v", err)
	}
	// Sanity: validation succeeds before tamper
	if vErr := svc.ValidateDelegationCtx(context.Background(), childResp.POA.ID, childResp.POA.Grantee, "finance.read"); vErr != nil {
		t.Fatalf("expected validation succeed pre-tamper: %v", vErr)
	}
	// Tamper parent body (e.g., change scope) -> recomputed digest differs
	parent := rootResp.POA
	parent.Scope = []string{"finance.read", "finance.write"} // broaden parent
	// Persist tampered parent to repo to simulate unauthorized mutation
	_ = svc.repo.Update(&parent)
	// Validation should now fail due to parent digest mismatch
	if vErr := svc.ValidateDelegationCtx(context.Background(), childResp.POA.ID, childResp.POA.Grantee, "finance.read"); vErr == nil {
		t.Fatalf("expected validation failure after parent tamper")
	}
}
