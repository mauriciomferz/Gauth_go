package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {

	// Create a new memory-based token store with a 1 hour TTL
	store := token.NewMemoryStore(1 * time.Hour)
	ctx := context.Background()

	// Create a strongly-typed token with metadata
	deviceInfo := &token.DeviceInfo{
		ID:        "device-123",
		UserAgent: "Example/1.0",
		IPAddress: "192.168.1.1",
		Platform:  "web",
		Version:   "1.0.0",
	}

	metadata := &token.Metadata{
		Device:     deviceInfo,
		AppID:      "example-app",
		AppVersion: "1.0.0",
		Labels: map[string]string{
			"environment": "production",
			"region":      "us-west",
		},
		Tags: []string{"web", "mobile-friendly"},
		Attributes: map[string][]string{
			"roles": {"user", "admin"},
		},
	}

	// Create a new token with claims
	newToken := &token.Token{
		ID:        "token-123",
		Type:      token.Access,
		Value:     "dummy-token-value", // In practice, this would be a signed JWT
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		NotBefore: time.Now(),
		Issuer:    "auth.example.com",
		Subject:   "user-123",
		Audience:  []string{"api.example.com"},
		Scopes:    []string{"read", "write"},
		Algorithm: token.RS256,
		Metadata:  metadata,
	}

	// Save token using token.ID as the key
	if err := store.Save(ctx, newToken.ID, newToken); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}

	// Retrieve token using token.ID as the key
	retrievedToken, err := store.Get(ctx, newToken.ID)
	if err != nil {
		log.Fatalf("Failed to retrieve token: %v", err)
	}

	// Validate token
	if err := store.Validate(ctx, retrievedToken); err != nil {
		if validationErr, ok := err.(*token.ValidationError); ok {
			log.Printf("Validation error: %s - %s", validationErr.Code, validationErr.Message)
		}
		log.Fatalf("Token validation failed: %v", err)
	}

	// List tokens with type-safe filter
	filter := token.Filter{
		Types:        []token.Type{token.Access},
		Subject:      "user-123",
		Issuer:       "auth.example.com",
		ExpiresAfter: time.Now(),
		Active:       true,
		Metadata: map[string]string{
			"environment": "production",
		},
	}

	tokens, err := store.List(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to list tokens: %v", err)
	}

	// Demonstrate type safety in token operations
	for _, t := range tokens {
		fmt.Printf("Token ID: %s\n", t.ID)
		fmt.Printf("Type: %s\n", t.Type)
		fmt.Printf("Scopes: %v\n", t.Scopes)
		fmt.Printf("Device: %s (%s)\n", t.Metadata.Device.ID, t.Metadata.Device.Platform)
		fmt.Printf("App: %s v%s\n", t.Metadata.AppID, t.Metadata.AppVersion)
		fmt.Printf("Labels: %v\n", t.Metadata.Labels)
		fmt.Printf("Tags: %v\n", t.Metadata.Tags)
		fmt.Printf("Roles: %v\n", t.Metadata.Attributes["roles"])

		// Type-safe token revocation
		if t.RevocationStatus != nil {
			fmt.Printf("Revoked at: %v by %s\n",
				t.RevocationStatus.RevokedAt,
				t.RevocationStatus.RevokedBy)
			fmt.Printf("Reason: %s\n", t.RevocationStatus.Reason)
		}
	}

	// Demonstrate token rotation with type safety
	newRotatedToken := &token.Token{
		ID:        "token-124",
		Type:      token.Access,
		Value:     "new-dummy-token-value",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		NotBefore: time.Now(),
		Issuer:    retrievedToken.Issuer,
		Subject:   retrievedToken.Subject,
		Audience:  retrievedToken.Audience,
		Scopes:    retrievedToken.Scopes,
		Algorithm: retrievedToken.Algorithm,
		Metadata:  retrievedToken.Metadata,
	}

	if err := store.Rotate(ctx, retrievedToken, newRotatedToken); err != nil {
		log.Fatalf("Failed to rotate token: %v", err)
	}

	fmt.Println("\nToken successfully rotated!")
}
