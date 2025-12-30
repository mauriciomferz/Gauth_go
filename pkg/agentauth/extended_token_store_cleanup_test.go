package agentauth

import (
	"context"
	"testing"
	"time"
)

func TestMemoryExtendedTokenStore_Cleanup_RevokedWithGracePeriod(t *testing.T) {
	store := NewMemoryExtendedTokenStore()
	ctx := context.Background()

	// 1. Create a token
	token := &ExtendedToken{
		AccessToken: "revoked-token",
		ExpiresIn:   3600, // Valid for 1 hour
		IssuedAt:    time.Now(),
	}
	if err := store.SaveToken(ctx, token); err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// 2. Revoke it immediately
	if err := store.RevokeToken(ctx, "revoked-token"); err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// 3. Run cleanup (simulating immediate cleanup)
	// Currently DeleteExpiredTokens only looks at expiration, so it should NOT remove it yet
	// because it expires in 1 hour.
	// But we want it to be removed if it's been revoked for > 24 hours (grace period).
	// For this test, we'll manually manipulate the revocation time to validiate the logic.

	// Let's first verify it exists and is revoked
	if revoked, _ := store.IsRevoked(ctx, "revoked-token"); !revoked {
		t.Error("Token should be revoked")
	}

	// Manually set revocation time to 25 hours ago
	store.mu.Lock()
	revokedAt := time.Now().Add(-25 * time.Hour)
	store.revokedTokens["revoked-token"] = revokedAt
	if stored, ok := store.tokens["revoked-token"]; ok {
		stored.Metadata.RevokedAt = &revokedAt
	}
	store.mu.Unlock()

	// 4. Run cleanup
	count, err := store.DeleteExpiredTokens(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// 5. Assertions
	// It should be removed because it's been revoked for > 24 hours
	if count != 1 {
		t.Errorf("Expected 1 token cleaned, got %d", count)
	}

	if _, err := store.GetToken(ctx, "revoked-token"); err != ErrTokenNotFound {
		t.Errorf("Token should be removed, got error: %v", err)
	}
}

func TestMemoryExtendedTokenStore_Cleanup_ExpiredWithGracePeriod(t *testing.T) {
	store := NewMemoryExtendedTokenStore()
	ctx := context.Background()

	// 1. Create a token that expired 30 minutes ago
	// Grace period is 1 hour, so it should NOT be removed yet
	token := &ExtendedToken{
		AccessToken: "expired-recent",
		ExpiresIn:   3600,
		IssuedAt:    time.Now().Add(-1*time.Hour - 30*time.Minute), // Expired 30 mins ago
	}
	// Sanity check
	if !isTokenExpired(token) {
		t.Fatal("Token should be expired")
	}

	if err := store.SaveToken(ctx, token); err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// 2. Run cleanup
	count, err := store.DeleteExpiredTokens(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// 3. Verify it is NOT removed (within 1h grace period)
	if count != 0 {
		t.Errorf("Token within grace period should not be removed. Count: %d", count)
	}

	// 4. Update to be expired > 1 hour ago (e.g., 2 hours ago)
	store.mu.Lock()
	store.tokens["expired-recent"].Token.IssuedAt = time.Now().Add(-3 * time.Hour) // Expired 2h ago
	store.mu.Unlock()

	// 5. Run cleanup again
	count, err = store.DeleteExpiredTokens(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// 6. Verify removal
	if count != 1 {
		t.Errorf("Token outside grace period should be removed. Count: %d", count)
	}
}
