package oidc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTokenRevocationService tests the token revocation service.
func TestTokenRevocationService(t *testing.T) {
	service := NewTokenRevocationService()
	ctx := context.Background()

	t.Run("revoke_and_check", func(t *testing.T) {
		err := service.RevokeToken(ctx, "token1", "test_reason", "user123", time.Now().Add(1*time.Hour))
		require.NoError(t, err)

		revoked, err := service.IsRevoked(ctx, "token1")
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("not_revoked", func(t *testing.T) {
		revoked, err := service.IsRevoked(ctx, "token2")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("get_revocation_info", func(t *testing.T) {
		err := service.RevokeToken(ctx, "token3", "security_breach", "admin", time.Now().Add(1*time.Hour))
		require.NoError(t, err)

		info, err := service.GetRevocationInfo(ctx, "token3")
		require.NoError(t, err)
		assert.Equal(t, "token3", info.TokenID)
		assert.Equal(t, "security_breach", info.Reason)
		assert.Equal(t, "admin", info.RevokedBy)
	})

	t.Run("batch_revoke", func(t *testing.T) {
		tokens := []string{"batch1", "batch2", "batch3"}
		expiresAt := time.Now().Add(1 * time.Hour)
		err := service.RevokeTokensBatch(ctx, tokens, "batch_revocation", "system", expiresAt)
		require.NoError(t, err)

		for _, token := range tokens {
			revoked, err := service.IsRevoked(ctx, token)
			require.NoError(t, err)
			assert.True(t, revoked)
		}
	})

	t.Run("cleanup_expired", func(t *testing.T) {
		// Add a token that's already expired
		pastTime := time.Now().Add(-1 * time.Hour)
		err := service.RevokeToken(ctx, "expired_token", "test", "user", pastTime)
		require.NoError(t, err)

		// Should be revoked now
		revoked, err := service.IsRevoked(ctx, "expired_token")
		require.NoError(t, err)
		assert.True(t, revoked)

		// Cleanup
		service.CleanupExpired()

		// Should not be revoked after cleanup
		revoked, err = service.IsRevoked(ctx, "expired_token")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("concurrent_operations", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				tokenID := "concurrent_token"
				_ = service.RevokeToken(ctx, tokenID, "concurrent_test", "user", time.Now().Add(1*time.Hour))
				_, _ = service.IsRevoked(ctx, tokenID)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestRefreshTokenService tests the refresh token service.
func TestRefreshTokenService(t *testing.T) {
	service := NewRefreshTokenService()
	ctx := context.Background()

	t.Run("store_and_retrieve", func(t *testing.T) {
		entry := &RefreshTokenEntry{
			RefreshToken: "refresh_token_1",
			ProviderID:   "google",
			Subject:      "user123",
			Audience:     "client1",
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		err := service.StoreRefreshToken(ctx, "refresh_token_1", entry)
		require.NoError(t, err)

		retrieved, err := service.GetRefreshToken(ctx, "refresh_token_1")
		require.NoError(t, err)
		assert.Equal(t, "google", retrieved.ProviderID)
		assert.Equal(t, "user123", retrieved.Subject)
		assert.Equal(t, "client1", retrieved.Audience)
	})

	t.Run("retrieve_nonexistent", func(t *testing.T) {
		_, err := service.GetRefreshToken(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("retrieve_expired", func(t *testing.T) {
		entry := &RefreshTokenEntry{
			RefreshToken: "expired_token",
			ProviderID:   "google",
			Subject:      "user123",
			IssuedAt:     time.Now().Add(-25 * time.Hour),
			ExpiresAt:    time.Now().Add(-1 * time.Hour),
		}

		err := service.StoreRefreshToken(ctx, "expired_token", entry)
		require.NoError(t, err)

		_, err = service.GetRefreshToken(ctx, "expired_token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("update_usage", func(t *testing.T) {
		entry := &RefreshTokenEntry{
			RefreshToken: "usage_token",
			ProviderID:   "okta",
			Subject:      "user456",
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		err := service.StoreRefreshToken(ctx, "usage_token", entry)
		require.NoError(t, err)

		// Use it once
		err = service.UpdateRefreshTokenUsage(ctx, "usage_token")
		require.NoError(t, err)

		// Use it again
		err = service.UpdateRefreshTokenUsage(ctx, "usage_token")
		require.NoError(t, err)

		// Check usage count
		retrieved, err := service.GetRefreshToken(ctx, "usage_token")
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.UseCount)
		assert.True(t, retrieved.LastUsed.After(entry.IssuedAt))
	})

	t.Run("revoke_refresh_token", func(t *testing.T) {
		entry := &RefreshTokenEntry{
			RefreshToken: "revoke_token",
			ProviderID:   "azure",
			Subject:      "user789",
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		err := service.StoreRefreshToken(ctx, "revoke_token", entry)
		require.NoError(t, err)

		// Revoke it
		err = service.RevokeRefreshToken(ctx, "revoke_token")
		require.NoError(t, err)

		// Should not be found
		_, err = service.GetRefreshToken(ctx, "revoke_token")
		assert.Error(t, err)
	})

	t.Run("cleanup_expired", func(t *testing.T) {
		// Add an expired token
		expiredEntry := &RefreshTokenEntry{
			RefreshToken: "cleanup_expired_token",
			ProviderID:   "google",
			Subject:      "user_cleanup",
			IssuedAt:     time.Now().Add(-25 * time.Hour),
			ExpiresAt:    time.Now().Add(-1 * time.Hour),
		}

		err := service.StoreRefreshToken(ctx, "cleanup_expired_token", expiredEntry)
		require.NoError(t, err)

		// Cleanup
		service.CleanupExpired()

		// Should not be found
		_, err = service.GetRefreshToken(ctx, "cleanup_expired_token")
		assert.Error(t, err)
	})

	t.Run("concurrent_operations", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				entry := &RefreshTokenEntry{
					RefreshToken: "concurrent_token",
					ProviderID:   "google",
					Subject:      "concurrent_user",
					IssuedAt:     time.Now(),
					ExpiresAt:    time.Now().Add(24 * time.Hour),
				}
				_ = service.StoreRefreshToken(ctx, "concurrent_token", entry)
				_, _ = service.GetRefreshToken(ctx, "concurrent_token")
				_ = service.UpdateRefreshTokenUsage(ctx, "concurrent_token")
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestTokenIntrospectionService tests the token introspection service.
func TestTokenIntrospectionService(t *testing.T) {
	// Simplified test without real JWKS dependency
	// Full integration tests with real tokens are in test/integration
	t.Run("empty_token", func(t *testing.T) {
		// This test shows the expected behavior without full setup
		// Real testing requires integration tests with valid JWTs
		t.Skip("Requires full JWKS setup - see integration tests")
	})
}

// TestTokenRevocationServiceCleanupLoop tests the cleanup background process.
func TestTokenRevocationServiceCleanupLoop(t *testing.T) {
	service := NewTokenRevocationService()
	ctx := context.Background()

	// Add an expired token
	pastTime := time.Now().Add(-1 * time.Hour)
	err := service.RevokeToken(ctx, "cleanup_test", "test", "user", pastTime)
	require.NoError(t, err)

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)
	service.CleanupExpired()

	// Should be cleaned up
	revoked, err := service.IsRevoked(ctx, "cleanup_test")
	require.NoError(t, err)
	assert.False(t, revoked)
}

// TestRefreshTokenServiceCleanupLoop tests the cleanup background process.
func TestRefreshTokenServiceCleanupLoop(t *testing.T) {
	service := NewRefreshTokenService()
	ctx := context.Background()

	// Add an expired token
	entry := &RefreshTokenEntry{
		RefreshToken: "cleanup_test_refresh",
		ProviderID:   "google",
		Subject:      "user_cleanup",
		IssuedAt:     time.Now().Add(-25 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}

	err := service.StoreRefreshToken(ctx, "cleanup_test_refresh", entry)
	require.NoError(t, err)

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)
	service.CleanupExpired()

	// Should be cleaned up
	_, err = service.GetRefreshToken(ctx, "cleanup_test_refresh")
	assert.Error(t, err)
}

// TestTokenRevocationInfoFields tests revocation info data structure.
func TestTokenRevocationInfoFields(t *testing.T) {
	service := NewTokenRevocationService()
	ctx := context.Background()

	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	err := service.RevokeToken(ctx, "info_test", "security_incident", "admin_user", expiresAt)
	require.NoError(t, err)

	info, err := service.GetRevocationInfo(ctx, "info_test")
	require.NoError(t, err)

	assert.Equal(t, "info_test", info.TokenID)
	assert.Equal(t, "security_incident", info.Reason)
	assert.Equal(t, "admin_user", info.RevokedBy)
	assert.True(t, info.RevokedAt.After(now.Add(-1*time.Second)))
	assert.Equal(t, expiresAt.Unix(), info.ExpiresAt.Unix())
}

// TestRefreshTokenEntryFields tests refresh token entry data structure.
func TestRefreshTokenEntryFields(t *testing.T) {
	service := NewRefreshTokenService()
	ctx := context.Background()

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(24 * time.Hour)

	entry := &RefreshTokenEntry{
		RefreshToken: "field_test_token",
		ProviderID:   "provider123",
		Subject:      "subject456",
		Audience:     "aud1",
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
	}

	err := service.StoreRefreshToken(ctx, "field_test_token", entry)
	require.NoError(t, err)

	retrieved, err := service.GetRefreshToken(ctx, "field_test_token")
	require.NoError(t, err)

	assert.Equal(t, "field_test_token", retrieved.RefreshToken)
	assert.Equal(t, "provider123", retrieved.ProviderID)
	assert.Equal(t, "subject456", retrieved.Subject)
	assert.Equal(t, "aud1", retrieved.Audience)
	assert.Equal(t, issuedAt.Unix(), retrieved.IssuedAt.Unix())
	assert.Equal(t, expiresAt.Unix(), retrieved.ExpiresAt.Unix())
	assert.Equal(t, 0, retrieved.UseCount)
}
