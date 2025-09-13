package tokenstore_test

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/tokenstore"
)

func TestMemoryStore(t *testing.T) {
	store := tokenstore.NewMemoryStore()

	// Test storing and retrieving token
	t.Run("Store and Get Token", func(t *testing.T) {
		token := "test-token"
		data := tokenstore.TokenData{
			Valid:      true,
			ValidUntil: time.Now().Add(time.Hour),
			ClientID:   "test-client",
			OwnerID:    "test-owner",
			Scope:      []string{"read", "write"},
		}

		err := store.Store(token, data)
		if err != nil {
			t.Errorf("Failed to store token: %v", err)
		}

		retrieved, exists := store.Get(token)
		if !exists {
			t.Error("Token not found in store")
		}

		if retrieved.ClientID != data.ClientID {
			t.Errorf("Expected ClientID %s, got %s", data.ClientID, retrieved.ClientID)
		}
	})

	// Test token deletion
	t.Run("Delete Token", func(t *testing.T) {
		token := "delete-test-token"
		data := tokenstore.TokenData{
			Valid:      true,
			ValidUntil: time.Now().Add(time.Hour),
			ClientID:   "test-client",
		}

		_ = store.Store(token, data)
		err := store.Delete(token)
		if err != nil {
			t.Errorf("Failed to delete token: %v", err)
		}

		_, exists := store.Get(token)
		if exists {
			t.Error("Token still exists after deletion")
		}
	})

	// Test cleanup of expired tokens
	t.Run("Cleanup Expired Tokens", func(t *testing.T) {
		expiredToken := "expired-token"
		validToken := "valid-token"

		expiredData := tokenstore.TokenData{
			Valid:      true,
			ValidUntil: time.Now().Add(-time.Hour),
			ClientID:   "test-client",
		}

		validData := tokenstore.TokenData{
			Valid:      true,
			ValidUntil: time.Now().Add(time.Hour),
			ClientID:   "test-client",
		}

		_ = store.Store(expiredToken, expiredData)
		_ = store.Store(validToken, validData)

		err := store.Cleanup()
		if err != nil {
			t.Errorf("Failed to cleanup tokens: %v", err)
		}

		_, expiredExists := store.Get(expiredToken)
		if expiredExists {
			t.Error("Expired token still exists after cleanup")
		}

		_, validExists := store.Get(validToken)
		if !validExists {
			t.Error("Valid token was removed during cleanup")
		}
	})
}
