package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/internal/tokenstore"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func TestTokenManagementIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("TokenLifecycle", func(t *testing.T) {
		// Create token manager
		manager := token.NewManager(token.ManagerConfig{
			Issuer:     "test-issuer",
			KeyID:      "test-key-1",
			SigningKey: []byte("test-signing-key"),
			Store:      tokenstore.NewMemoryStore(),
		})

		// Test token creation
		t.Run("Creation", func(t *testing.T) {
			claims := map[string]interface{}{
				"sub":  "user123",
				"role": "admin",
			}
			token, err := manager.CreateToken(ctx, claims, 1*time.Hour)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Verify token
			validated, err := manager.ValidateToken(ctx, token)
			require.NoError(t, err)
			assert.Equal(t, claims["sub"], validated["sub"])
			assert.Equal(t, claims["role"], validated["role"])
		})

		// Test token revocation
		t.Run("Revocation", func(t *testing.T) {
			claims := map[string]interface{}{"sub": "user456"}
			token, err := manager.CreateToken(ctx, claims, 1*time.Hour)
			require.NoError(t, err)

			// Revoke token
			err = manager.RevokeToken(ctx, token)
			require.NoError(t, err)

			// Verify revoked token fails validation
			_, err = manager.ValidateToken(ctx, token)
			assert.Error(t, err)
		})

		// Test token refresh
		t.Run("Refresh", func(t *testing.T) {
			claims := map[string]interface{}{"sub": "user789"}
			token, refresh, err := manager.CreateTokenWithRefresh(ctx, claims, 1*time.Hour, 24*time.Hour)
			require.NoError(t, err)

			// Use refresh token
			newToken, err := manager.RefreshToken(ctx, refresh)
			require.NoError(t, err)
			assert.NotEmpty(t, newToken)
			assert.NotEqual(t, token, newToken)

			// Verify new token
			validated, err := manager.ValidateToken(ctx, newToken)
			require.NoError(t, err)
			assert.Equal(t, claims["sub"], validated["sub"])
		})
	})

	t.Run("DistributedTokenStore", func(t *testing.T) {
		store := tokenstore.NewDistributedStore(tokenstore.DistributedConfig{
			Addresses: []string{"localhost:6379"},
			Password:  "",
			DB:        0,
		})
		defer store.Close()

		manager := token.NewManager(token.ManagerConfig{
			Issuer:     "test-issuer",
			KeyID:      "test-key-1",
			SigningKey: []byte("test-signing-key"),
			Store:      store,
		})

		// Test distributed token storage
		t.Run("StorageAndRetrieval", func(t *testing.T) {
			claims := map[string]interface{}{"sub": "user123"}
			token, err := manager.CreateToken(ctx, claims, 1*time.Hour)
			require.NoError(t, err)

			// Verify token can be validated
			validated, err := manager.ValidateToken(ctx, token)
			require.NoError(t, err)
			assert.Equal(t, claims["sub"], validated["sub"])
		})

		// Test distributed revocation
		t.Run("DistributedRevocation", func(t *testing.T) {
			claims := map[string]interface{}{"sub": "user456"}
			token, err := manager.CreateToken(ctx, claims, 1*time.Hour)
			require.NoError(t, err)

			// Revoke token
			err = manager.RevokeToken(ctx, token)
			require.NoError(t, err)

			// Verify token is revoked across all nodes
			_, err = manager.ValidateToken(ctx, token)
			assert.Error(t, err)
		})
	})

	t.Run("KeyRotation", func(t *testing.T) {
		manager := token.NewManager(token.ManagerConfig{
			Issuer:     "test-issuer",
			KeyID:      "test-key-1",
			SigningKey: []byte("test-signing-key"),
			Store:      tokenstore.NewMemoryStore(),
		})

		// Create token with old key
		claims := map[string]interface{}{"sub": "user123"}
		oldToken, err := manager.CreateToken(ctx, claims, 1*time.Hour)
		require.NoError(t, err)

		// Rotate key
		err = manager.RotateKey("test-key-2", []byte("new-signing-key"))
		require.NoError(t, err)

		// Create token with new key
		newToken, err := manager.CreateToken(ctx, claims, 1*time.Hour)
		require.NoError(t, err)

		// Both tokens should be valid during rotation period
		_, err = manager.ValidateToken(ctx, oldToken)
		assert.NoError(t, err)
		_, err = manager.ValidateToken(ctx, newToken)
		assert.NoError(t, err)

		// Complete rotation
		err = manager.CompleteRotation()
		require.NoError(t, err)

		// Only new tokens should be valid
		_, err = manager.ValidateToken(ctx, oldToken)
		assert.Error(t, err)
		_, err = manager.ValidateToken(ctx, newToken)
		assert.NoError(t, err)
	})

	t.Run("TokenMonitoring", func(t *testing.T) {
		monitor := token.NewMonitor()
		manager := token.NewManager(token.ManagerConfig{
			Issuer:     "test-issuer",
			KeyID:      "test-key-1",
			SigningKey: []byte("test-signing-key"),
			Store:      tokenstore.NewMemoryStore(),
			Monitor:    monitor,
		})

		// Generate some token activity
		claims := map[string]interface{}{"sub": "user123"}
		token, err := manager.CreateToken(ctx, claims, 1*time.Hour)
		require.NoError(t, err)

		_, err = manager.ValidateToken(ctx, token)
		require.NoError(t, err)

		err = manager.RevokeToken(ctx, token)
		require.NoError(t, err)

		// Check metrics
		stats := monitor.GetStats()
		assert.Greater(t, stats.TokensCreated, uint64(0))
		assert.Greater(t, stats.TokensValidated, uint64(0))
		assert.Greater(t, stats.TokensRevoked, uint64(0))
	})
}
