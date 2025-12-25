package gnap

import (
	"testing"
	"time"
)

// TestMemoryTokenStore_Cleanup verifies expired and revoked token cleanup
func TestMemoryTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryTokenStore()

	now := time.Now()

	// Create expired token
	expiredToken := &IssuedToken{
		Value:     "expired-token",
		GrantID:   "grant-1",
		IssuedAt:  now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour), // Expired 1 hour ago
	}
	_ = store.Store(expiredToken)

	// Create recently revoked token (should NOT be cleaned)
	recentRevokedToken := &IssuedToken{
		Value:     "recent-revoked",
		GrantID:   "grant-1",
		IssuedAt:  now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(1 * time.Hour),
		Revoked:   true,
		RevokedAt: now.Add(-1 * time.Hour), // Revoked 1 hour ago
	}
	_ = store.Store(recentRevokedToken)

	// Create old revoked token (should be cleaned)
	oldRevokedToken := &IssuedToken{
		Value:     "old-revoked",
		GrantID:   "grant-2",
		IssuedAt:  now.Add(-48 * time.Hour),
		ExpiresAt: now.Add(1 * time.Hour),
		Revoked:   true,
		RevokedAt: now.Add(-25 * time.Hour), // Revoked 25 hours ago
	}
	_ = store.Store(oldRevokedToken)

	// Create valid token
	validToken := &IssuedToken{
		Value:     "valid-token",
		GrantID:   "grant-1",
		IssuedAt:  now,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	_ = store.Store(validToken)

	// Create token with no expiration
	noExpireToken := &IssuedToken{
		Value:    "no-expire",
		GrantID:  "grant-3",
		IssuedAt: now,
	}
	_ = store.Store(noExpireToken)

	// Verify all tokens exist
	if len(store.tokens) != 5 {
		t.Fatalf("Expected 5 tokens before cleanup, got %d", len(store.tokens))
	}

	// Run cleanup
	removed := store.Cleanup()

	// Should remove 2 tokens: expired + old revoked
	if removed != 2 {
		t.Errorf("Expected 2 tokens removed, got %d", removed)
	}

	// Verify only 3 tokens remain
	if len(store.tokens) != 3 {
		t.Errorf("Expected 3 tokens after cleanup, got %d", len(store.tokens))
	}

	// Verify the right tokens remain
	_, err := store.Get(validToken.Value)
	if err != nil {
		t.Error("Valid token should still exist")
	}

	_, err = store.Get(noExpireToken.Value)
	if err != nil {
		t.Error("Non-expiring token should still exist")
	}

	// Recent revoked should remain (within grace period)
	if _, ok := store.tokens[recentRevokedToken.Value]; !ok {
		t.Error("Recently revoked token should remain during grace period")
	}

	// Verify cleaned tokens are gone
	if _, ok := store.tokens[expiredToken.Value]; ok {
		t.Error("Expired token should be removed")
	}

	if _, ok := store.tokens[oldRevokedToken.Value]; ok {
		t.Error("Old revoked token should be removed")
	}
}

// TestMemoryTokenStore_CleanupGrantIndex verifies grant index is maintained
func TestMemoryTokenStore_CleanupGrantIndex(t *testing.T) {
	store := NewMemoryTokenStore()

	now := time.Now()
	grantID := "grant-456"

	// Create 2 expired and 1 valid token for the same grant
	expired1 := &IssuedToken{
		Value:     "expired1",
		GrantID:   grantID,
		IssuedAt:  now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	_ = store.Store(expired1)

	expired2 := &IssuedToken{
		Value:     "expired2",
		GrantID:   grantID,
		IssuedAt:  now.Add(-3 * time.Hour),
		ExpiresAt: now.Add(-2 * time.Hour),
	}
	_ = store.Store(expired2)

	validToken := &IssuedToken{
		Value:     "valid",
		GrantID:   grantID,
		IssuedAt:  now,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	_ = store.Store(validToken)

	// Verify grant has 3 tokens
	tokens, _ := store.ListByGrant(grantID)
	if len(tokens) != 3 {
		t.Fatalf("Expected 3 tokens for grant before cleanup, got %d", len(tokens))
	}

	// Cleanup
	store.Cleanup()

	// Verify grant index has only 1 token
	tokens, _ = store.ListByGrant(grantID)
	if len(tokens) != 1 {
		t.Errorf("Expected 1 token for grant after cleanup, got %d", len(tokens))
	}

	if tokens[0].Value != validToken.Value {
		t.Error("Wrong token remained in grant index")
	}
}

// TestMemoryTokenStore_CleanupEmptyGrants verifies empty grant entries are removed
func TestMemoryTokenStore_CleanupEmptyGrants(t *testing.T) {
	store := NewMemoryTokenStore()

	now := time.Now()
	grantID := "grant-789"

	// Create only expired tokens for a grant
	expired1 := &IssuedToken{
		Value:     "expired1",
		GrantID:   grantID,
		IssuedAt:  now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	_ = store.Store(expired1)

	expired2 := &IssuedToken{
		Value:     "expired2",
		GrantID:   grantID,
		IssuedAt:  now.Add(-3 * time.Hour),
		ExpiresAt: now.Add(-2 * time.Hour),
	}
	_ = store.Store(expired2)

	// Verify grant exists in index
	if _, ok := store.byGrant[grantID]; !ok {
		t.Fatal("Grant should exist in index before cleanup")
	}

	// Cleanup
	store.Cleanup()

	// Verify empty grant entry is removed from index
	if _, ok := store.byGrant[grantID]; ok {
		t.Error("Empty grant entry should be removed from index")
	}
}
