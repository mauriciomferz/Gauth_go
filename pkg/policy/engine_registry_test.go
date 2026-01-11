package policy

import (
	"context"
	"testing"
)

// TestRegistryRollback verifies Rollback correctly sets headOverride to specified version.
func TestRegistryRollback(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Add three bundles
	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b2 := Bundle{ID: "bundle-2", Version: 2, Policies: []Policy{{ID: "p2", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b3 := Bundle{ID: "bundle-3", Version: 3, Policies: []Policy{{ID: "p3", Subjects: []string{"*"}, Rules: []Rule{}}}}

	if _, err := store.AppendBundle(ctx, b1); err != nil {
		t.Fatalf("failed to add bundle 1: %v", err)
	}
	if _, err := store.AppendBundle(ctx, b2); err != nil {
		t.Fatalf("failed to add bundle 2: %v", err)
	}
	if _, err := store.AppendBundle(ctx, b3); err != nil {
		t.Fatalf("failed to add bundle 3: %v", err)
	}

	// Rollback to version 2
	if err := store.Rollback(ctx, 2); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify head is version 2
	head, err := store.Head(ctx)
	if err != nil {
		t.Fatalf("Head error: %v", err)
	}
	if head == nil {
		t.Fatal("Head returned nil after rollback")
	}
	if head.Version != 2 {
		t.Errorf("expected version 2 after rollback, got %d", head.Version)
	}
}

// TestRegistryRollback_NotFound verifies Rollback returns error for non-existent version.
func TestRegistryRollback_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	if _, err := store.AppendBundle(ctx, b1); err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	// Try to rollback to non-existent version
	if err := store.Rollback(ctx, 99); err == nil {
		t.Error("expected error for rollback to non-existent version, got nil")
	}
}

// TestRegistryActiveVersion verifies ActiveVersion returns correct version after rollback.
func TestRegistryActiveVersion(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Empty registry should return 0
	if av, _ := store.ActiveVersion(ctx); av != 0 {
		t.Errorf("expected ActiveVersion=0 for empty registry, got %d", av)
	}

	// Add bundles
	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b2 := Bundle{ID: "bundle-2", Version: 2, Policies: []Policy{{ID: "p2", Subjects: []string{"*"}, Rules: []Rule{}}}}

	if _, err := store.AppendBundle(ctx, b1); err != nil {
		t.Fatalf("failed to add bundle 1: %v", err)
	}
	if _, err := store.AppendBundle(ctx, b2); err != nil {
		t.Fatalf("failed to add bundle 2: %v", err)
	}

	// Should return latest version (2)
	if av, _ := store.ActiveVersion(ctx); av != 2 {
		t.Errorf("expected ActiveVersion=2, got %d", av)
	}

	// Rollback to version 1
	if err := store.Rollback(ctx, 1); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Should now return 1
	if av, _ := store.ActiveVersion(ctx); av != 1 {
		t.Errorf("expected ActiveVersion=1 after rollback, got %d", av)
	}
}

// TestRegistryChainWithVersions verifies ChainWithVersions returns correct version/hash pairs.
func TestRegistryChainWithVersions(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Empty registry should return empty slice
	// Note: accessing internal registry for this test as ChainWithVersions isn't on Store interface
	chain := store.Registry().ChainWithVersions()
	if len(chain) != 0 {
		t.Errorf("expected empty chain for empty registry, got %d items", len(chain))
	}

	// Add bundles
	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b2 := Bundle{ID: "bundle-2", Version: 2, Policies: []Policy{{ID: "p2", Subjects: []string{"*"}, Rules: []Rule{}}}}

	b1Added, err := store.AppendBundle(ctx, b1)
	if err != nil {
		t.Fatalf("failed to add bundle 1: %v", err)
	}
	b2Added, err := store.AppendBundle(ctx, b2)
	if err != nil {
		t.Fatalf("failed to add bundle 2: %v", err)
	}

	// Verify chain
	chain = store.Registry().ChainWithVersions()
	if len(chain) != 2 {
		t.Fatalf("expected 2 items in chain, got %d", len(chain))
	}

	if chain[0].Version != 1 || chain[0].Hash != b1Added.Hash {
		t.Errorf("chain[0] mismatch: expected version=1 hash=%s, got version=%d hash=%s",
			b1Added.Hash, chain[0].Version, chain[0].Hash)
	}
	if chain[1].Version != 2 || chain[1].Hash != b2Added.Hash {
		t.Errorf("chain[1] mismatch: expected version=2 hash=%s, got version=%d hash=%s",
			b2Added.Hash, chain[1].Version, chain[1].Hash)
	}
}

// TestRegistryDiff_BasicChanges verifies Diff correctly identifies added, removed, changed policies.
func TestRegistryDiff_BasicChanges(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Version 1: policy-a and policy-b
	b1 := Bundle{
		ID:      "bundle-1",
		Version: 1,
		Policies: []Policy{
			{
				ID:       "policy-a",
				Subjects: []string{"user:alice"},
				Rules: []Rule{{
					Actions:   []string{"read"},
					Resources: []string{"doc:*"},
					Effect:    Allow,
				}},
			},
			{
				ID:       "policy-b",
				Subjects: []string{"user:bob"},
				Rules: []Rule{{
					Actions:   []string{"write"},
					Resources: []string{"doc:*"},
					Effect:    Allow,
				}},
			},
		},
	}

	// Version 2: policy-a (modified), policy-c (new), policy-b removed
	b2 := Bundle{
		ID:      "bundle-2",
		Version: 2,
		Policies: []Policy{
			{
				ID:       "policy-a",
				Subjects: []string{"user:alice", "user:admin"},
				Rules: []Rule{{
					Actions:   []string{"read"},
					Resources: []string{"doc:*"},
					Effect:    Allow,
				}},
			},
			{
				ID:       "policy-c",
				Subjects: []string{"user:charlie"},
				Rules: []Rule{{
					Actions:   []string{"delete"},
					Resources: []string{"doc:*"},
					Effect:    Allow,
				}},
			},
		},
	}

	if _, err := store.AppendBundle(ctx, b1); err != nil {
		t.Fatalf("failed to add bundle 1: %v", err)
	}
	if _, err := store.AppendBundle(ctx, b2); err != nil {
		t.Fatalf("failed to add bundle 2: %v", err)
	}

	// Compute diff from version 1 to 2 using standalone Diff
	diff, err := Diff(ctx, store, 1, 2)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Verify added: policy-c
	if len(diff.Added) != 1 || diff.Added[0].ID != "policy-c" {
		t.Errorf("expected 1 added policy (policy-c), got %d: %v", len(diff.Added), diff.Added)
	}

	// Verify removed: policy-b
	if len(diff.Removed) != 1 || diff.Removed[0].ID != "policy-b" {
		t.Errorf("expected 1 removed policy (policy-b), got %d: %v", len(diff.Removed), diff.Removed)
	}

	// Verify changed: policy-a (subjects changed)
	if len(diff.Changed) != 1 || diff.Changed[0].ID != "policy-a" {
		t.Errorf("expected 1 changed policy (policy-a), got %d: %v", len(diff.Changed), diff.Changed)
	}
}

// TestRegistryDiff_EmptyChain verifies Diff returns error for empty registry.
func TestRegistryDiff_EmptyChain(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()
	// Should fail because versions 1 and 2 don't exist
	if _, err := Diff(ctx, store, 1, 2); err == nil {
		t.Error("expected error for Diff on empty registry, got nil")
	}
}

// TestRegistryDiff_SameVersion verifies Diff returns error when versions are identical.
func TestRegistryDiff_SameVersion(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	if _, err := store.AppendBundle(ctx, b1); err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	if _, err := Diff(ctx, store, 1, 1); err == nil {
		t.Error("expected error for Diff with identical versions, got nil")
	}
}

// TestRegistryDiff_DefaultVersions verifies Diff uses ActiveVersion and head when versions are 0.
func TestRegistryDiff_DefaultVersions(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b2 := Bundle{ID: "bundle-2", Version: 2, Policies: []Policy{{ID: "p2", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b3 := Bundle{ID: "bundle-3", Version: 3, Policies: []Policy{{ID: "p3", Subjects: []string{"*"}, Rules: []Rule{}}}}

	if _, err := store.AppendBundle(ctx, b1); err != nil {
		t.Fatalf("failed to add bundle 1: %v", err)
	}
	if _, err := store.AppendBundle(ctx, b2); err != nil {
		t.Fatalf("failed to add bundle 2: %v", err)
	}
	if _, err := store.AppendBundle(ctx, b3); err != nil {
		t.Fatalf("failed to add bundle 3: %v", err)
	}

	// Diff with fromVersion=0 (should use ActiveVersion=3), toVersion=0 (should use head=3)
	// This should fail because they're the same
	if _, err := Diff(ctx, store, 0, 0); err == nil {
		t.Error("expected error for Diff(0,0) when both resolve to same version, got nil")
	}

	// Rollback to version 1
	if err := store.Rollback(ctx, 1); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Now Diff(0,0) should compare ActiveVersion=1 (from rollback) to Head=1?
	// Note: logic in Diff wrapper: from=0 -> ActiveVersion. to=0 -> Head.
	// ActiveVersion is 1. Head is 3 (unless Head() respects override? InMemoryStore.Head calls reg.Head().
	// reg.Head() respects override!)
	// Wait, reg.Head() respects override? YES (line 114 engine.go: returns headOverride if not nil)
	// So `Head` returns 1.
	// So both become 1.
	// It should fail.
	if _, err := Diff(ctx, store, 0, 0); err == nil {
		t.Error("expected error for Diff(0,0) after rollback to same version, got nil")
	}

	// Test explicit version diff: (0=Active=1) to 3
	diff, err := Diff(ctx, store, 0, 3)
	if err != nil {
		t.Fatalf("Diff(0,3) after rollback failed: %v", err)
	}

	if diff.FromVersion != 1 || diff.ToVersion != 3 {
		t.Errorf("expected diff from version 1 to 3, got from %d to %d", diff.FromVersion, diff.ToVersion)
	}
}

// TestRegistryFindByHash verifies FindByHash returns correct bundle or nil.
func TestRegistryFindByHash(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	b1 := Bundle{ID: "bundle-1", Version: 1, Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}}}
	b1Added, err := store.AppendBundle(ctx, b1)
	if err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	// Find by hash should return the bundle
	found, err := store.GetByHash(ctx, b1Added.Hash)
	if err != nil {
		t.Fatalf("GetByHash error: %v", err)
	}
	if found == nil {
		t.Fatal("GetByHash returned nil for existing hash")
	}
	if found.Version != 1 {
		t.Errorf("expected version 1, got %d", found.Version)
	}

	// Non-existent hash should return nil
	notFound, _ := store.GetByHash(ctx, "nonexistent-hash-12345")
	if notFound != nil {
		t.Error("expected nil for non-existent hash, got non-nil")
	}
}

// TestValidateBundle_ValidBundle verifies ValidateBundle accepts valid bundles.
func TestValidateBundle_ValidBundle(t *testing.T) {
	bundle := Bundle{
		ID:      "valid-bundle",
		Version: 1,
		Policies: []Policy{
			{
				ID:       "valid-policy",
				Subjects: []string{"user:alice"},
				Rules: []Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"doc:*"},
						Effect:    Allow,
					},
				},
			},
		},
	}

	err := ValidateBundle(bundle)
	if err != nil {
		t.Errorf("ValidateBundle failed for valid bundle: %v", err)
	}
}

// TestValidateBundle_EmptyBundleID verifies ValidateBundle rejects bundles with empty ID.
func TestValidateBundle_EmptyBundleID(t *testing.T) {
	bundle := Bundle{
		ID:       "",
		Version:  1,
		Policies: []Policy{{ID: "p1", Subjects: []string{"*"}, Rules: []Rule{}}},
	}

	err := ValidateBundle(bundle)
	if err == nil {
		t.Error("expected error for bundle with empty ID, got nil")
	}
}

// TestValidateBundle_NoPolicies verifies ValidateBundle rejects bundles with no policies.
func TestValidateBundle_NoPolicies(t *testing.T) {
	bundle := Bundle{
		ID:       "bundle-no-policies",
		Version:  1,
		Policies: []Policy{},
	}

	err := ValidateBundle(bundle)
	if err == nil {
		t.Error("expected error for bundle with no policies, got nil")
	}
}

// TestValidateBundle_EmptyPolicyID verifies ValidateBundle rejects policies with empty ID.
func TestValidateBundle_EmptyPolicyID(t *testing.T) {
	bundle := Bundle{
		ID:      "bundle-bad-policy",
		Version: 1,
		Policies: []Policy{
			{
				ID:       "",
				Subjects: []string{"*"},
				Rules:    []Rule{},
			},
		},
	}

	err := ValidateBundle(bundle)
	if err == nil {
		t.Error("expected error for policy with empty ID, got nil")
	}
}

// TestValidateBundle_NoSubjects verifies ValidateBundle rejects policies with no subjects.
func TestValidateBundle_NoSubjects(t *testing.T) {
	bundle := Bundle{
		ID:      "bundle-no-subjects",
		Version: 1,
		Policies: []Policy{
			{
				ID:       "policy-no-subjects",
				Subjects: []string{},
				Rules:    []Rule{},
			},
		},
	}

	err := ValidateBundle(bundle)
	if err == nil {
		t.Error("expected error for policy with no subjects, got nil")
	}
}
