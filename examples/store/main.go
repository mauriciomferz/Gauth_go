package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/store"
)

func main() {
	// Create a memory store with default configuration
	memoryStore, err := store.NewTokenStore(store.MemoryStoreType, nil)
	if err != nil {
		log.Fatalf("Failed to create memory store: %v", err)
	}
	defer closeStore(memoryStore)

	// Create a context for our operations
	ctx := context.Background()

	// Example 1: Basic token storage and retrieval
	fmt.Println("=== Example 1: Basic Token Operations ===")
	basicTokenOperations(ctx, memoryStore)

	// Example 2: Working with multiple tokens for the same subject
	fmt.Println("\n=== Example 2: Working with Multiple Tokens ===")
	multipleTokensExample(ctx, memoryStore)

	// Example 3: Error handling
	fmt.Println("\n=== Example 3: Error Handling ===")
	errorHandlingExample(ctx, memoryStore)

	// Example 4: Token revocation
	fmt.Println("\n=== Example 4: Token Revocation ===")
	revocationExample(ctx, memoryStore)
}

// closeStore safely closes any store that implements a Close method
func closeStore(s store.TokenStore) {
	if closer, ok := s.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			log.Printf("Warning: Error closing store: %v", err)
		}
	}
}

// basicTokenOperations demonstrates storing and retrieving a token
func basicTokenOperations(ctx context.Context, tokenStore store.TokenStore) {
	// Create a sample token with metadata
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0" // #nosec G101 - This is a test JWT token for demo purposes
	metadata := store.TokenMetadata{
		ID:        "token123",
		Subject:   "user123",
		Issuer:    "gauth",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		KeyID:     "key123",
		Type:      "access",
		Status:    "active",
	}

	// Store the token
	fmt.Println("Storing token...")
	if err := tokenStore.Store(ctx, token, metadata); err != nil {
		log.Fatalf("Failed to store token: %v", err)
	}

	// Retrieve the token by its string value
	retrievedMetadata, err := tokenStore.Get(ctx, token)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	fmt.Printf("Retrieved token by value: %+v\n", retrievedMetadata)

	// Retrieve the token by its ID
	byIDMetadata, err := tokenStore.GetByID(ctx, metadata.ID)
	if err != nil {
		log.Fatalf("Failed to get token by ID: %v", err)
	}
	fmt.Printf("Retrieved token by ID: %+v\n", byIDMetadata)
}

// multipleTokensExample demonstrates working with multiple tokens for a subject
func multipleTokensExample(ctx context.Context, tokenStore store.TokenStore) {
	// Create multiple tokens for the same user
	subject := "multi_user"
	tokens := []string{
		"token_1_for_multi_user",
		"token_2_for_multi_user",
		"token_3_for_multi_user",
	}

	// Store the tokens with different types
	tokenTypes := []string{"access", "refresh", "api_key"}

	fmt.Println("Storing multiple tokens for the same subject...")
	for i, tokenValue := range tokens {
		metadata := store.TokenMetadata{
			ID:        fmt.Sprintf("multi_%d", i),
			Subject:   subject,
			Issuer:    "gauth",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
			Type:      tokenTypes[i],
			Status:    "active",
		}

		if err := tokenStore.Store(ctx, tokenValue, metadata); err != nil {
			log.Fatalf("Failed to store token %d: %v", i, err)
		}
	}

	// List all tokens for the subject
	subjectTokens, err := tokenStore.List(ctx, subject)
	if err != nil {
		log.Fatalf("Failed to list tokens: %v", err)
	}

	fmt.Printf("Found %d tokens for subject %s:\n", len(subjectTokens), subject)
	for i, t := range subjectTokens {
		fmt.Printf("  %d: ID=%s, Type=%s, Expires=%s\n",
			i+1, t.ID, t.Type, t.ExpiresAt.Format(time.RFC3339))
	}
}

// errorHandlingExample demonstrates proper error handling
func errorHandlingExample(ctx context.Context, tokenStore store.TokenStore) {
	// Try to get a non-existent token
	nonExistentToken := "this_token_does_not_exist"

	fmt.Printf("Attempting to get non-existent token: %s\n", nonExistentToken)
	_, err := tokenStore.Get(ctx, nonExistentToken)

	if err != nil {
		// Check if it's a StorageError and handle appropriately
		if storageErr, ok := err.(*store.StorageError); ok {
			fmt.Printf("StorageError details:\n")
			fmt.Printf("  Operation: %s\n", storageErr.Op)
			fmt.Printf("  Key: %s\n", storageErr.Key)
			fmt.Printf("  Underlying error: %v\n", storageErr.Err)
			fmt.Printf("  Detail: %s\n", storageErr.Detail)
		} else {
			fmt.Printf("Unexpected error type: %T: %v\n", err, err)
		}
	}
}

// revocationExample demonstrates token revocation
func revocationExample(ctx context.Context, tokenStore store.TokenStore) {
	// Create a token to revoke
	token := "token_to_be_revoked" // #nosec G101 - This is a test token for demo purposes
	metadata := store.TokenMetadata{
		ID:        "revocable_token",
		Subject:   "revoke_test_user",
		Issuer:    "gauth",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Long expiry
		Type:      "access",
		Status:    "active",
	}

	// Store the token
	fmt.Println("Storing a token for revocation...")
	if err := tokenStore.Store(ctx, token, metadata); err != nil {
		log.Fatalf("Failed to store token: %v", err)
	}

	// Check if token is revoked (should be false)
	revoked, err := tokenStore.IsRevoked(ctx, token)
	if err != nil {
		log.Fatalf("Failed to check revocation status: %v", err)
	}
	fmt.Printf("Is token revoked? %v\n", revoked)

	// Revoke the token
	fmt.Println("Revoking the token...")
	if err := tokenStore.Revoke(ctx, token); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}

	// Check again if token is revoked (should be true)
	revoked, err = tokenStore.IsRevoked(ctx, token)
	if err != nil {
		log.Fatalf("Failed to check revocation status: %v", err)
	}
	fmt.Printf("Is token revoked now? %v\n", revoked)

	// Try to use the revoked token
	fmt.Println("Attempting to use revoked token...")
	_, err = tokenStore.Get(ctx, token)
	if err != nil {
		if storageErr, ok := err.(*store.StorageError); ok && storageErr.Err == store.ErrTokenRevoked {
			fmt.Println("Correctly rejected the revoked token")
		} else {
			log.Fatalf("Unexpected error when using revoked token: %v", err)
		}
	} else {
		log.Fatal("ERROR: Revoked token was still usable!")
	}
}
