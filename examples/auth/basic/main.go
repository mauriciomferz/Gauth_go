package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	// Generate RSA key pair for signing tokens
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	// Create a memory store for tokens
	store := token.NewMemoryStore(24 * time.Hour)

	// Configure token service
	tokenConfig := token.Config{
		SigningMethod:    token.RS256,
		SigningKey:       privateKey,
		ValidityPeriod:   time.Hour,
		DefaultScopes:    []string{"read", "write"},
		ValidateAudience: false,
		ValidateIssuer:   false,
	}
	tokenService := token.NewService(tokenConfig, store)

	// Create auth service
	authService := auth.NewService(tokenService)

	ctx := context.Background()

	// Authenticate using password grant
	tokenReq := &auth.ServiceTokenRequest{
		GrantType: "password",
		Username:  "user123",
		Password:  "testpass",
		Scope:     "read write",
	}
	tokenResp, err := authService.Token(ctx, tokenReq)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Printf("Authentication successful. Token: %s\nToken ID: %s\n\n", tokenResp.AccessToken, tokenResp.IDToken)

	// Validate the token using the token ID
	validatedToken, err := authService.Validate(ctx, tokenResp.IDToken, []string{"read", "write"})
	if err != nil {
		log.Fatalf("Token validation failed: %v", err)
	}
	fmt.Printf("Token validated. Subject: %s, Scopes: %v\n", validatedToken.Subject, validatedToken.Scopes)
}
