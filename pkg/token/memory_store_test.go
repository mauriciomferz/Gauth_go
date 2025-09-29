package token

import (
	"context"
	"strconv"
	"testing"
	"time"
)

//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

func TestMemoryStore(t *testing.T) {
	t.Run("Save and Get", testMemoryStoreSaveAndGet)
	t.Run("Token Not Found", testMemoryStoreNotFound)
	t.Run("Delete Token", testMemoryStoreDelete)
	t.Run("List Tokens", testMemoryStoreList)
	t.Run("TTL Expiration", testMemoryStoreTTLExpiration)
}

func testMemoryStoreSaveAndGet(t *testing.T) {
	ttl := time.Hour
	store := NewMemoryStore(ttl)
	ctx := context.Background()

	token := &Token{
		Value:     "test-token",
		Type:      Access,
		ExpiresAt: time.Now().Add(ttl),
		Scopes:    []string{"read", "write"},
		Metadata:  &Metadata{AppData: map[string]string{"user": "123"}},
	}

	// Save token
	err := store.Save(ctx, "key1", token)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// Get token
	retrieved, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	// Verify token
	if retrieved.Value != token.Value {
		t.Errorf("Got wrong token value: got %v want %v", retrieved.Value, token.Value)
	}
}

func testMemoryStoreNotFound(t *testing.T) {
	store := NewMemoryStore(time.Hour)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func testMemoryStoreDelete(t *testing.T) {
	ttl := time.Hour
	store := NewMemoryStore(ttl)
	ctx := context.Background()

	token := &Token{
		Value:     "test-token",
		Type:      Access,
		ExpiresAt: time.Now().Add(ttl),
		Scopes:    []string{"read", "write"},
	}

	// Save and delete
	err := store.Save(ctx, "key2", token)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	err = store.Delete(ctx, "key2")
	if err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, "key2")
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound after deletion, got %v", err)
	}
}

func testMemoryStoreList(t *testing.T) {
	ttl := time.Hour
	store := NewMemoryStore(ttl)
	ctx := context.Background()

	// Save multiple tokens
	tokens := []*Token{
		{
			Value:     "access1",
			Type:      Access,
			ExpiresAt: time.Now().Add(ttl),
			Scopes:    []string{"read"},
		},
		{
			Value:     "refresh1",
			Type:      Refresh,
			ExpiresAt: time.Now().Add(ttl),
			Scopes:    []string{"refresh"},
		},
	}

	for i, tok := range tokens {
		key := "list-key" + strconv.Itoa(i)
		err := store.Save(ctx, key, tok)
		if err != nil {
			t.Fatalf("Failed to save token %d: %v", i, err)
		}
	}

	// List with type filter
	filter := Filter{
		Types:        []Type{Access},
		ExpiresAfter: time.Now(),
	}

	list, err := store.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list tokens: %v", err)
	}

	// Verify filter
	for _, tok := range list {
		if tok.Type != Access {
			t.Errorf("Listed token has wrong type: got %v want %v", tok.Type, Access)
		}
	}
}

func testMemoryStoreTTLExpiration(t *testing.T) {
	shortTTL := 10 * time.Millisecond
	expStore := NewMemoryStore(shortTTL)
	ctx := context.Background()

	// Create a token that expires very soon
	expToken := &Token{
		Value:     "expiring-token",
		Type:      Access,
		ExpiresAt: time.Now().Add(shortTTL),
		Scopes:    []string{"read"},
	}
	err := expStore.Save(ctx, "exp-key", expToken)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// Wait for expiration (increase to 10x TTL for robustness)
	time.Sleep(shortTTL * 10)

	// Try to get expired token
	_, err = expStore.Get(ctx, "exp-key")
	if err != ErrTokenExpired && err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenExpired or ErrTokenNotFound, got %v", err)
	}
}
