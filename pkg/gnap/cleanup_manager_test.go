package gnap

import (
	"context"
	"testing"
	"time"
)

func TestCleanupManager_StartStop(t *testing.T) {
	grantStore := NewMemoryGrantStore()
	tokenStore := NewMemoryTokenStore()
	manager := NewCleanupManager(grantStore, tokenStore, 100*time.Millisecond)

	ctx := context.Background()

	// Start
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	stats := manager.Stats()
	if !stats.Running {
		t.Error("Manager should be running")
	}

	// Start again should be idempotent
	err = manager.Start(ctx)
	if err != nil {
		t.Error("Second Start should not error")
	}

	// Wait for at least one cleanup
	time.Sleep(150 * time.Millisecond)

	stats = manager.Stats()
	if stats.LastCleanup.IsZero() {
		t.Error("Cleanup should have run")
	}

	// Stop
	manager.Stop()

	stats = manager.Stats()
	if stats.Running {
		t.Error("Manager should be stopped")
	}

	// Stop again should be idempotent
	manager.Stop()
}

func TestCleanupManager_RunOnce(t *testing.T) {
	grantStore := NewMemoryGrantStore()
	tokenStore := NewMemoryTokenStore()

	now := time.Now()

	// Create expired grant
	expiredGrant, _ := grantStore.Create(&GrantRequest{})
	expiredGrant.ExpiresAt = now.Add(-1 * time.Hour)
	_ = grantStore.Update(expiredGrant)

	// Create expired token
	expiredToken := &IssuedToken{
		Value:     "expired",
		IssuedAt:  now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	_ = tokenStore.Store(expiredToken)

	manager := NewCleanupManager(grantStore, tokenStore, time.Hour)

	// Run cleanup
	grantsRemoved, tokensRemoved := manager.RunOnce()

	if grantsRemoved != 1 {
		t.Errorf("Expected 1 grant removed, got %d", grantsRemoved)
	}

	if tokensRemoved != 1 {
		t.Errorf("Expected 1 token removed, got %d", tokensRemoved)
	}

	stats := manager.Stats()
	if stats.TotalGrantsCleaned != 1 {
		t.Errorf("Expected total grants cleaned = 1, got %d", stats.TotalGrantsCleaned)
	}

	if stats.TotalTokensCleaned != 1 {
		t.Errorf("Expected total tokens cleaned = 1, got %d", stats.TotalTokensCleaned)
	}
}

func TestCleanupManager_PeriodicCleanup(t *testing.T) {
	grantStore := NewMemoryGrantStore()
	tokenStore := NewMemoryTokenStore()

	now := time.Now()

	// Create some expired items
	for i := 0; i < 3; i++ {
		grant, _ := grantStore.Create(&GrantRequest{})
		grant.ExpiresAt = now.Add(-1 * time.Hour)
		_ = grantStore.Update(grant)
	}

	manager := NewCleanupManager(grantStore, tokenStore, 50*time.Millisecond)

	ctx := context.Background()
	_ = manager.Start(ctx)
	defer manager.Stop()

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	stats := manager.Stats()
	if stats.TotalGrantsCleaned < 3 {
		t.Errorf("Expected at least 3 grants cleaned, got %d", stats.TotalGrantsCleaned)
	}
}

func TestCleanupManager_ContextCancellation(t *testing.T) {
	grantStore := NewMemoryGrantStore()
	tokenStore := NewMemoryTokenStore()
	manager := NewCleanupManager(grantStore, tokenStore, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	_ = manager.Start(ctx)

	// Cancel context
	cancel()

	// Give it time to shut down
	time.Sleep(100 * time.Millisecond)

	// Should have stopped gracefully
	stats := manager.Stats()
	if stats.Running {
		t.Error("Manager should have stopped after context cancellation")
	}
}

func TestCleanupManager_NilStores(t *testing.T) {
	// Test with nil stores (should not panic)
	manager := NewCleanupManager(nil, nil, time.Minute)

	grants, tokens := manager.RunOnce()

	if grants != 0 {
		t.Errorf("Expected 0 grants removed with nil store, got %d", grants)
	}

	if tokens != 0 {
		t.Errorf("Expected 0 tokens removed with nil store, got %d", tokens)
	}
}
