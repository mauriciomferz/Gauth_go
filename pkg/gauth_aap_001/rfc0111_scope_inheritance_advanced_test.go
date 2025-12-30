package gauth_aap_001

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
)

// TestScopeInheritanceAdvanced verifies regex pattern coverage when GAUTH_ENABLE_ADVANCED_SCOPE=1.
func TestScopeInheritanceAdvanced(t *testing.T) {
	os.Setenv("GAUTH_ENABLE_ADVANCED_SCOPE", "1")
	defer os.Unsetenv("GAUTH_ENABLE_ADVANCED_SCOPE")
	path := t.TempDir() + "/poa.db"
	os.Setenv("GAUTH_PERSIST_PATH", path)
	defer os.Unsetenv("GAUTH_PERSIST_PATH")
	memLogger := audit.NewMemoryLogger(nil)
	svc := NewService(memLogger, &allowAllAuthorizer{})
	root, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"finance.read", "re:^audit\\.[a-z]+\\.write$"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("root create failed: %v", err)
	}
	// Regex covers audit.log.write
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"audit.log.write"}, Duration: time.Hour, ParentPOAID: root.POA.ID}); err != nil {
		t.Fatalf("expected regex covered scope allowed: %v", err)
	}
	// Regex should not cover audit.log.delete (pattern expects write)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "dave", Scope: []string{"audit.log.delete"}, Duration: time.Hour, ParentPOAID: root.POA.ID}); err == nil {
		t.Fatalf("expected audit.log.delete to be rejected (not covered by regex pattern)")
	}
}
