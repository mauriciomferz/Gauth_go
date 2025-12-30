package gauth_aap_001

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// TestHierarchicalDigestRootAndChild verifies Version bump to 4 and parent digest binding.
func TestHierarchicalDigestRootAndChild(t *testing.T) {
	os.Setenv("AGENTAUTH_ENABLE_HIER_DIGEST", "1")
	defer os.Unsetenv("AGENTAUTH_ENABLE_HIER_DIGEST")
	path := t.TempDir() + "/poa.db"
	os.Setenv("AGENTAUTH_PERSIST_PATH", path)
	defer os.Unsetenv("AGENTAUTH_PERSIST_PATH")
	memLogger := audit.NewMemoryLogger(nil)
	kms, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("kms init: %v", err)
	}
	svc := NewService(memLogger, &allowAllAuthorizer{}, WithKMS(kms))
	rootResp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"finance.read"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("root create failed: %v", err)
	}
	if rootResp.POA.Version != 4 {
		// Taxonomy absence ensures bump only due to hier flag; root should still be v4
		// (Version 4 signals inclusion of hierarchy object even for depth=0)
		if rootResp.POA.Version != 4 {
			t.Fatalf("expected root Version=4 got %d", rootResp.POA.Version)
		}
	}
	childResp, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"finance.read"}, Duration: time.Hour, ParentPOAID: rootResp.POA.ID})
	if err != nil {
		t.Fatalf("child create failed: %v", err)
	}
	if childResp.POA.Version != 4 {
		t.Fatalf("expected child Version=4 got %d", childResp.POA.Version)
	}
	if childResp.POA.ParentDigest == "" {
		t.Fatalf("expected parent digest captured")
	}
	// Recompute canonical digests to ensure deterministic difference between root and child
	rootDig, _, derr := CanonicalPOADigest(&rootResp.POA)
	if derr != nil {
		t.Fatalf("canonical root err: %v", derr)
	}
	childDig, _, derr2 := CanonicalPOADigest(&childResp.POA)
	if derr2 != nil {
		t.Fatalf("canonical child err: %v", derr2)
	}
	if rootDig == childDig {
		t.Fatalf("expected distinct digests root vs child")
	}
}

// TestHierDigestDisabled ensures Version <4 when hier digest flag not enabled.
func TestHierDigestDisabled(t *testing.T) {
	path := t.TempDir() + "/poa.db"
	os.Setenv("AGENTAUTH_PERSIST_PATH", path)
	defer os.Unsetenv("AGENTAUTH_PERSIST_PATH")
	memLogger := audit.NewMemoryLogger(nil)
	kms, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("kms init: %v", err)
	}
	svc := NewService(memLogger, &allowAllAuthorizer{}, WithKMS(kms))
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"finance.read"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if resp.POA.Version >= 4 {
		t.Fatalf("expected Version<4 when hier digest disabled got %d", resp.POA.Version)
	}
}

// TestHierDigestTamperParent verifies tampering parent digest causes child signature mismatch after re-sign attempt.
func TestHierDigestTamperParent(t *testing.T) {
	os.Setenv("AGENTAUTH_ENABLE_HIER_DIGEST", "1")
	defer os.Unsetenv("AGENTAUTH_ENABLE_HIER_DIGEST")
	path := t.TempDir() + "/poa.db"
	os.Setenv("AGENTAUTH_PERSIST_PATH", path)
	defer os.Unsetenv("AGENTAUTH_PERSIST_PATH")
	memLogger := audit.NewMemoryLogger(nil)
	kms, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("kms init: %v", err)
	}
	svc := NewService(memLogger, &allowAllAuthorizer{}, WithKMS(kms))
	rootResp, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"finance.read"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("root create failed: %v", err)
	}
	childResp, err := svc.CreateDelegation(DelegationRequest{Grantor: "bob", Grantee: "carol", Scope: []string{"finance.read"}, Duration: time.Hour, ParentPOAID: rootResp.POA.ID})
	if err != nil {
		t.Fatalf("child create failed: %v", err)
	}
	// Simulate tamper: modify child's ParentDigest and recompute canonical digest; signature digest should mismatch.
	origSig := childResp.POA.Signature
	if origSig == nil {
		t.Fatalf("child signature missing")
	}
	childResp.POA.ParentDigest = "deadbeef" // invalid digest
	newDig, _, derr := CanonicalPOADigest(&childResp.POA)
	if derr != nil {
		t.Fatalf("canonical after tamper err: %v", derr)
	}
	if newDig == origSig.DigestHex {
		t.Fatalf("expected digest change after tamper")
	}
}
