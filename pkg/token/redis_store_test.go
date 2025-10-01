package token

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRevocationReason = "test revocation"
)

//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

func TestRedisStore(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	store, err := NewRedisStore(RedisConfig{
		Addresses:  []string{s.Addr()},
		KeyPrefix:  "test:",
		DefaultTTL: time.Hour,
	})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("Save and Get Token", func(t *testing.T) {
		token := &Token{
			ID:        "test-id",
			Value:     "test-value",
			Type:      Access,
			Subject:   "test-user",
			Issuer:    "test-issuer",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		err := store.Save(ctx, token)
		require.NoError(t, err)
		retrieved, err := store.Get(ctx, token.ID)
		require.NoError(t, err)
		assert.Equal(t, token.ID, retrieved.ID)
		assert.Equal(t, token.Value, retrieved.Value)
		assert.Equal(t, token.Subject, retrieved.Subject)
	})

	t.Run("Token Not Found", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrTokenNotFound)
		_, err = store.GetByValue(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrTokenNotFound)
	})

	t.Run("Delete Token", func(t *testing.T) {
		token := &Token{
			ID:    "delete-test",
			Value: "delete-value",
		}
		require.NoError(t, store.Save(ctx, token))
		_, err := store.Get(ctx, token.ID)
		require.NoError(t, err)
		require.NoError(t, store.Delete(ctx, token.ID))
		_, err = store.Get(ctx, token.ID)
		assert.ErrorIs(t, err, ErrTokenNotFound)
		_, err = store.GetByValue(ctx, token.Value)
		assert.ErrorIs(t, err, ErrTokenNotFound)
	})

	t.Run("Revoke Token", func(t *testing.T) {
		token := &Token{
			ID:    "revoke-test",
			Value: "revoke-value",
		}
		require.NoError(t, store.Save(ctx, token))
		reason := testRevocationReason
		require.NoError(t, store.Revoke(ctx, token.ID, reason))
		retrieved, err := store.Get(ctx, token.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.RevocationStatus)
		assert.Equal(t, reason, retrieved.RevocationStatus.Reason)
	})
}
