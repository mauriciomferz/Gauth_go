package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
)

func main() {
	// Create a memory store with default configuration
	// This example uses the in-memory backend for demonstration purposes.
	memoryStore := token.NewMemoryStore()
	defer closeStore(memoryStore)

	// Create a context for our operations
	ctx := context.Background()

	// Example 1: Basic token storage and retrieval
	// Demonstrates storing and retrieving a token with metadata.
	fmt.Println("=== Example 1: Basic Token Operations ===")
	basicTokenOperations(ctx, memoryStore)

	// Example 2: Working with multiple tokens for the same subject
	// Shows how to store and list multiple tokens for a single user.
	fmt.Println("\n=== Example 2: Working with Multiple Tokens ===")
	multipleTokensExample(ctx, memoryStore)

	// Example 3: Error handling
	// Demonstrates handling errors when retrieving non-existent tokens.
	fmt.Println("\n=== Example 3: Error Handling ===")
	errorHandlingExample(ctx, memoryStore)

	// Example 4: Token revocation
	// Shows how to revoke a token and check its status.
	fmt.Println("\n=== Example 4: Token Revocation ===")
	revocationExample(ctx, memoryStore)
}

// closeStore safely closes any store that implements a Close method
func closeStore(s *token.MemoryStore) {
	if err := s.Close(); err != nil {
		log.Printf("Warning: Error closing store: %v", err)
	}
}

// basicTokenOperations demonstrates storing and retrieving a token
func basicTokenOperations(ctx context.Context, tokenStore *token.MemoryStore) {
	// Create a sample token with metadata
	tok := &token.Token{
		ID:        "token123",
		Subject:   "user123",
		Issuer:    "gauth",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Type:      token.Access,
	}
	fmt.Println("Storing token...")
	if err := tokenStore.Save(ctx, tok.ID, tok); err != nil {
		log.Fatalf("Failed to store token: %v", err)
	}
	retrievedToken, err := tokenStore.Get(ctx, tok.ID)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	fmt.Printf("Retrieved token by ID: %+v\n", retrievedToken)
	fmt.Printf("Token stored with ID: %s\n", tok.ID)
}

// multipleTokensExample demonstrates working with multiple tokens for a subject
func multipleTokensExample(ctx context.Context, tokenStore *token.MemoryStore) {
	// Create multiple tokens for the same user
	subject := "multi_user"
	tokenTypes := []token.TokenType{token.Access, token.Refresh, token.ID}
	fmt.Println("Storing multiple tokens for the same subject...")
	for i := 0; i < 3; i++ {
		tok := &token.Token{
			ID:        fmt.Sprintf("multi_%d", i),
			Subject:   subject,
			Issuer:    "gauth",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
			Type:      tokenTypes[i],
		}
		if err := tokenStore.Save(ctx, tok.ID, tok); err != nil {
			log.Fatalf("Failed to store token %d: %v", i, err)
		}
	}
	fmt.Printf("Stored 3 tokens for subject %s\n", subject)
}

// errorHandlingExample demonstrates proper error handling
func errorHandlingExample(ctx context.Context, tokenStore *token.MemoryStore) {
	// Try to get a non-existent token
	nonExistentToken := "this_token_does_not_exist"
	fmt.Printf("Attempting to get non-existent token: %s\n", nonExistentToken)
	_, err := tokenStore.Get(ctx, nonExistentToken)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// revocationExample demonstrates token revocation
func revocationExample(ctx context.Context, tokenStore *token.MemoryStore) {
	// Create a token to revoke
	tok := &token.Token{
		ID:        "revocable_token",
		Subject:   "revoke_test_user",
		Issuer:    "gauth",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Type:      token.Access,
	}
	fmt.Println("Storing a token for revocation...")
	if err := tokenStore.Save(ctx, tok.ID, tok); err != nil {
		log.Fatalf("Failed to store token: %v", err)
	}
	// Check if token is revoked (should be false)
	revoked := tok.RevocationStatus != nil && !tok.RevocationStatus.RevokedAt.IsZero()
	fmt.Printf("Is token revoked? %v\n", revoked)
	// Revoke the token
	fmt.Println("Revoking the token...")
	if err := tokenStore.MarkRevoked(ctx, tok.ID, "demo revocation"); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}
	// Check again if token is revoked (should be true)
	revokedToken, err := tokenStore.Get(ctx, tok.ID)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	revoked = revokedToken.RevocationStatus != nil && !revokedToken.RevocationStatus.RevokedAt.IsZero()
	fmt.Printf("Is token revoked now? %v\n", revoked)
	// Try to use the revoked token
	fmt.Println("Attempting to use revoked token...")
	err = tokenStore.Validate(ctx, revokedToken)
	if err != nil {
		fmt.Println("Correctly rejected the revoked token")
	} else {
		fmt.Println("ERROR: Revoked token was still usable! (This is a demo error; in production, revoked tokens should not be usable)")
	}
}
