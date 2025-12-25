package gauth

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryPIPPolicyStore_Cleanup(t *testing.T) {
	store := NewInMemoryPIPPolicyStore()
	ctx := context.Background()

	// 1. Set a policy that expires quickly
	policy1 := &PowerOfAttorneyPolicy{
		PolicyID: "policy1",
		// ... other fields irrelevant for store test
	}
	if err := store.Set(ctx, "policy1", policy1, 1*time.Millisecond); err != nil {
		t.Fatalf("Failed to set policy1: %v", err)
	}

	// 2. Set a policy that expires later
	policy2 := &PowerOfAttorneyPolicy{
		PolicyID: "policy2",
	}
	if err := store.Set(ctx, "policy2", policy2, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set policy2: %v", err)
	}

	// Wait for policy1 to expire
	time.Sleep(2 * time.Millisecond)

	// 3. Run Cleanup
	count, err := store.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// 4. Verify results
	if count != 1 {
		t.Errorf("Expected 1 policy cleaned, got %d", count)
	}

	// Policy1 should be gone
	if _, err := store.Get(ctx, "policy1"); err != ErrPolicyNotFound {
		t.Errorf("Policy1 should be removed, got error: %v", err)
	}

	// Policy2 should remain
	if p, err := store.Get(ctx, "policy2"); err != nil || p == nil {
		t.Errorf("Policy2 should remain, got error: %v", err)
	}
}
