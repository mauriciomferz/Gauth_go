package basic
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	// Generate RSA key pair for signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create token service configuration
	config := token.Config{
		SigningMethod:    token.RS256,
		SigningKey:       privateKey,
		ValidityPeriod:   time.Hour,
		RefreshPeriod:    24 * time.Hour,
		DefaultScopes:    []string{"read"},
		ValidateAudience: true,
		ValidateIssuer:   true,
		AllowedAudiences: []string{"example-app"},
		AllowedIssuers:   []string{"auth-service"},
		MaxTokens:        1000,
		CleanupInterval:  15 * time.Minute,
	}

	// Create memory store with cleanup
	store, err := token.NewMemoryStore(
		token.WithTTL(24 * time.Hour),
		token.WithCleanup(15 * time.Minute),
		token.WithCapacity(1000),
	)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create token service
	service := token.NewService(config, store)

	ctx := context.Background()

	// Create a new access token with typed metadata
	token := &token.Token{
		Type:    token.Access,
		Subject: "user123",
		Issuer:  "auth-service",
		Audience: []string{"example-app"},
		Scopes:   []string{"read", "write"},
		Metadata: &token.Metadata{
			Device: &token.DeviceInfo{
				ID:        "device123",
				UserAgent: "ExampleApp/1.0",
				Platform:  "iOS",
				Version:   "15.0",
			},
			AppID:      "example-app",
			AppVersion: "1.0.0",
			Labels: map[string]string{
				"environment": "production",
				"region":     "us-west",
			},
			Tags: []string{"mobile", "authenticated"},
		},
	}

	// Issue the token
	issued, err := service.Issue(ctx, token)
	if err != nil {
		log.Fatalf("Failed to issue token: %v", err)
	}
	fmt.Printf("Issued token: %s\n\n", issued.Value)

	// Validate the token
	if err := service.Validate(ctx, issued); err != nil {
		log.Fatalf("Token validation failed: %v", err)
	}
	fmt.Println("Token validated successfully")

	// List tokens with filters
	filter := &token.Filter{
		Types:            []token.Type{token.Access},
		Subject:         "user123",
		Active:          true,
		RequireAllScopes: true,
		Scopes:          []string{"read"},
		Tags:            []string{"mobile"},
		Labels: map[string]string{
			"environment": "production",
		},
	}

	tokens, err := store.List(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to list tokens: %v", err)
	}
	fmt.Printf("Found %d matching tokens\n\n", len(tokens))

	// Revoke the token
	issued.RevocationStatus = &token.RevocationStatus{
		RevokedAt: time.Now(),
		Reason:    "user logout",
		RevokedBy: "example-app",
	}

	if err := store.Save(ctx, issued, time.Hour); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}

	// Verify token is now invalid
	err = service.Validate(ctx, issued)
	if err == nil {
		log.Fatal("Expected revoked token to be invalid")
	}
	fmt.Printf("Token validation failed as expected: %v\n", err)
}