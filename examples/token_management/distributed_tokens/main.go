package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	// Create a memory-based token store (replace with Redis if available)
	store := token.NewMemoryStore(24 * time.Hour)
	defer store.Close()

	// Create a validation chain (adjust config as needed)
	validator := token.NewValidationChain(token.ValidationConfig{
		AllowedIssuers:   []string{"example-service"},
		AllowedAudiences: []string{"example-app"},
		RequiredScopes:   []string{"read"},
		ClockSkew:        time.Minute,
	}, nil)

	fmt.Println("Token Management Example")
	fmt.Println("=======================")

	ctx := context.Background()

	// Create a token
	fmt.Println("\n1. Creating a new token")
	fmt.Println("----------------------")
	deviceInfo := &token.DeviceInfo{
		ID:        "device123",
		UserAgent: "Example/1.0",
		IPAddress: "192.168.1.1",
	}
	metadata := &token.Metadata{
		Device: deviceInfo,
		AppID: "example-app",
		Labels: map[string]string{"environment": "test"},
		Tags: []string{"demo"},
		Attributes: map[string][]string{"roles": {"user"}},
	}
	newToken := &token.Token{
		ID:        "example-token",
		Value:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		Type:      token.Access,
		Subject:   "user123",
		Issuer:    "example-service",
		Audience:  []string{"example-app"},
		Scopes:    []string{"read", "write"},
		Metadata:  metadata,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := store.Save(ctx, newToken.ID, newToken); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}
	fmt.Printf("Token created: %s\n", newToken.ID)

	// Retrieve and validate token
	fmt.Println("\n2. Retrieving and validating token")
	fmt.Println("--------------------------------")
	retrieved, err := store.Get(ctx, newToken.ID)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	fmt.Printf("Retrieved token for subject: %s\n", retrieved.Subject)

	if err := validator.Validate(ctx, retrieved); err != nil {
		fmt.Printf("Token validation failed: %v\n", err)
	} else {
		fmt.Println("Token validation successful")
	}

	// List tokens for user
	fmt.Println("\n3. Listing user tokens")
	fmt.Println("--------------------")
	userTokens, err := store.List(ctx, token.Filter{
		Types:   []token.Type{token.Access},
		Subject: "user123",
	})
	if err != nil {
		log.Fatalf("Failed to list tokens: %v", err)
	}
	fmt.Printf("Found %d tokens for user\n", len(userTokens))

	// Revoke token
	fmt.Println("\n4. Revoking token")
	fmt.Println("---------------")
	// Set revocation status and revoke the token
	newToken.RevocationStatus = &token.RevocationStatus{
		RevokedAt: time.Now(),
		Reason:    "user logout",
		RevokedBy: "example-service",
	}
	if err := store.Revoke(ctx, newToken); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}
	fmt.Println("Token revoked successfully")

	// Verify revocation
	fmt.Println("\n5. Verifying revocation")
	fmt.Println("---------------------")
	revokedToken, err := store.Get(ctx, newToken.ID)
	if err != nil {
		fmt.Printf("Token not found after revocation (expected if store deletes revoked tokens): %v\n", err)
	} else if revokedToken.RevocationStatus != nil {
		fmt.Printf("Token revoked at: %v\n", revokedToken.RevocationStatus.RevokedAt)
		fmt.Printf("Revocation reason: %s\n", revokedToken.RevocationStatus.Reason)
	}

	// Demonstrate validation failure
	fmt.Println("\n6. Demonstrating validation failure")
	fmt.Println("--------------------------------")
	invalidToken := &token.Token{
		ID:        "invalid-token",
		Type:      token.Access,
		Subject:   "user123",
		Issuer:    "unknown-service", // Not in allowed issuers
		Audience:  []string{"example-app"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read", "write"},
	}

	if err := validator.Validate(ctx, invalidToken); err != nil {
		fmt.Printf("Expected validation failure: %v\n", err)
	}
}
