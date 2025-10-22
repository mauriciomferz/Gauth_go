package test

import (
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/policy"
)

// TestPolicyBundleSubstitutionDetection simulates a substitution of a middle bundle in the chain.
// Current VerifyChain implementation recalculates each bundle hash and checks previous hash linkage.
// Substitution is detected because the modified bundle's stored Hash no longer matches its recomputed hash.
func TestPolicyBundleSubstitutionDetection(t *testing.T) {
	reg := policy.NewRegistry()
	// Create three bundles with minimal required policy content to satisfy validation.
	for i := 0; i < 3; i++ {
		b := policy.Bundle{ID: "b" + string(rune('1'+i)), Policies: []policy.Policy{{ID: "p" + string(rune('1'+i)), Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"res"}, Effect: policy.Allow}}}}}
		if _, err := reg.AddBundle(b); err != nil {
			t.Fatalf("add bundle %d: %v", i, err)
		}
	}
	if err := reg.VerifyChain(); err != nil {
		t.Fatalf("expected valid chain: %v", err)
	}

	// Simulate substitution of middle bundle (index 1): change policy list without recomputing hash.
	// (unused variable removed)
	// Access middle bundle directly
	// (registry.bundles is private; we indirectly tamper by capturing pointer after AddBundle sequence via FindByHash)
	// For test simplicity, iterate to get middle reference.
	hashes := reg.ChainHashes()
	if len(hashes) != 3 {
		t.Fatalf("expected 3 hashes")
	}
	midBundle := reg.FindByHash(hashes[1])
	if midBundle == nil {
		t.Fatalf("mid bundle not found")
	}
	// Tamper: replace policies with different content (policy IDs differ) and DO NOT recompute hash.
	midBundle.Policies = []policy.Policy{{ID: "tampered", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"other"}, Effect: policy.Allow}}}}

	// Verification should now fail due to hash mismatch at index 1.
	if err := reg.VerifyChain(); err == nil {
		t.Fatalf("expected substitution detection failure")
	}
}

// TestPolicyBundleNoSubstitution ensures unmodified chain passes verification.
func TestPolicyBundleNoSubstitution(t *testing.T) {
	reg := policy.NewRegistry()
	for i := 0; i < 2; i++ {
		b := policy.Bundle{ID: "B" + string(rune('A'+i)), Policies: []policy.Policy{{ID: "P" + string(rune('A'+i)), Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"op"}, Resources: []string{"res"}, Effect: policy.Allow}}}}}
		if _, err := reg.AddBundle(b); err != nil {
			t.Fatalf("add bundle: %v", err)
		}
	}
	if err := reg.VerifyChain(); err != nil {
		t.Fatalf("unexpected verify failure: %v", err)
	}
}
