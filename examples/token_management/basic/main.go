package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
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

	// Create memory store
	store := token.NewMemoryStore()

	// Create token service with config and store
	tokenService := token.NewService(config, store)

	ctx := context.Background()

	// Create a new access token
	accessToken := &token.Token{
		ID:        token.GenerateID(),
		Type:      token.Access,
		Value:     "", // Will be set by service
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		NotBefore: time.Now(),
		Issuer:    "auth-service",
		Subject:   "user123",
		Audience:  []string{"example-app"},
		Scopes:    []string{"read", "write"},
		Algorithm: token.RS256,
		Metadata: &token.Metadata{
			AppData: map[string]string{
				"device": "mobile",
				"os":     "ios",
			},
		},
	}

	// Issue token (signs and stores)
	issuedToken, err := tokenService.Issue(ctx, accessToken)
	if err != nil {
		log.Fatalf("Failed to issue token: %v", err)
	}
	fmt.Printf("Issued token: %s\n", issuedToken.Value)

	// Validate issued token
	if err := tokenService.Validate(ctx, issuedToken); err != nil {
		log.Fatalf("Token validation failed: %v", err)
	}
	fmt.Println("Token is valid")

	// Query stored tokens
	filter := token.Filter{
		Types:            []token.Type{token.Access},
		Subject:          "user123",
		Active:           true,
		RequireAllScopes: true,
		Scopes:           []string{"read"},
		Metadata: map[string]string{
			"device": "mobile",
		},
	}

	tokens, err := tokenService.List(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to query tokens: %v", err)
	}
	fmt.Printf("Found %d matching tokens\n", len(tokens))

	// Revoke token
	if err := tokenService.Revoke(ctx, issuedToken); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}

	// Verify token is now invalid
	err = tokenService.Validate(ctx, issuedToken)
	if err == nil {
		log.Fatalf("Token should be invalid after revocation")
	}
	fmt.Println("Token revoked and invalid as expected")
}
