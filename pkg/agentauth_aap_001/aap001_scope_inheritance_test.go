package agentauth_aap_001

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
)

// TestScopeInheritance verifies conservative subset enforcement for sub-delegation scopes.
func TestScopeInheritance(t *testing.T) {
	path := t.TempDir() + "/poa.db"
	t.Setenv("AGENTAUTH_PERSIST_PATH", path)

	memLogger := audit.NewMemoryLogger(nil)
	svc := NewService(memLogger, &allowAllAuthorizer{})
	root, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"finance.read", "audit.*"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("root create failed: %v", err)
	}
	// Allowed exact subset
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"finance.read"}, Duration: time.Hour, ParentPOAID: root.POA.ID}); err != nil {
		t.Fatalf("expected exact subset allowed: %v", err)
	}
	// Allowed prefix match (audit.* parent covers audit.log.write)
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "dave", Scope: []string{"audit.log.write"}, Duration: time.Hour, ParentPOAID: root.POA.ID}); err != nil {
		t.Fatalf("expected prefix subset allowed: %v", err)
	}
	// Rejected broaden: child adds finance.write not present and not covered by wildcard
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "erin", Scope: []string{"finance.write"}, Duration: time.Hour, ParentPOAID: root.POA.ID}); err == nil {
		t.Fatalf("expected finance.write to be rejected (not covered)")
	}
	// Rejected broaden: unrelated domain 'systems.reboot'
	if _, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "frank", Scope: []string{"systems.reboot"}, Duration: time.Hour, ParentPOAID: root.POA.ID}); err == nil {
		t.Fatalf("expected systems.reboot to be rejected (not covered)")
	}
}
