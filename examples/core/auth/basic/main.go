package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/events"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	// Generate RSA key pair for signing tokens
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	// Create an event publisher for monitoring
	publisher := events.NewPublisher()
	publisher.Subscribe(&events.LogHandler{}) // Log all auth events

	// Create a memory store for tokens
	store, err := token.NewMemoryStore(
		token.WithTTL(24 * time.Hour),
		token.WithCleanup(15 * time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Configure token service
	tokenService := token.NewService(token.Config{
		SigningMethod:    token.RS256,
		SigningKey:       privateKey,
		ValidityPeriod:   time.Hour,
		DefaultScopes:    []string{"read"},
		ValidateAudience: true,
		ValidateIssuer:   true,
		AllowedAudiences: []string{"example-app"},
		AllowedIssuers:   []string{"auth-service"},
	}, store)

	// Create auth service with basic provider
	authService := auth.NewService(auth.Config{
		TokenService: tokenService,
		Provider:    auth.NewBasicProvider(validateCredentials),
		EventHandler: publisher,
	})

	ctx := context.Background()

	// Attempt authentication
	creds := auth.Credentials{
		Username: "testuser",
		Password: "testpass",
		Metadata: &auth.AuthMetadata{
			Device: &token.DeviceInfo{
				ID:        "device123",
				UserAgent: "ExampleApp/1.0",
				Platform:  "iOS",
			},
			ClientID: "example-app",
			Scopes:   []string{"read", "write"},
		},
	}

	// Authenticate and get token
	token, err := authService.Authenticate(ctx, creds)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Printf("Authentication successful. Token: %s\n\n", token.Value)

	// Validate the token
	claims, err := authService.ValidateToken(ctx, token.Value)
	if err != nil {
		log.Fatalf("Token validation failed: %v", err)
	}
	fmt.Printf("Token validated. Subject: %s, Scopes: %v\n", claims.Subject, claims.Scopes)
}

// validateCredentials simulates credential validation
func validateCredentials(ctx context.Context, username, password string) error {
	// In a real app, validate against a database
	if username == "testuser" && password == "testpass" {
		return nil
	}
	return auth.ErrInvalidCredentials
}