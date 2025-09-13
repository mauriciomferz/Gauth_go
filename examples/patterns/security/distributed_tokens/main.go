package distributedtokens
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
	"github.com/Gimel-Foundation/gauth/pkg/events"
)

func main() {
	// Create a Redis-backed token store
	store, err := token.NewRedisStore(token.RedisConfig{
		Addresses:  []string{"localhost:6379"},
		KeyPrefix:  "example:",
		DefaultTTL: time.Hour * 24,
	})
	if err != nil {
		log.Fatalf("Failed to create token store: %v", err)
	}
	defer store.Close()

	// Create event publisher for monitoring
	publisher := events.NewPublisher()
	publisher.Subscribe(&events.LogHandler{})

	// Create validation chain with custom rules
	validator := token.NewValidationChain(token.ValidationConfig{
		AllowedIssuers:   []string{"example-service"},
		AllowedAudiences: []string{"example-app"},
		RequiredScopes:   []string{"read"},
		ClockSkew:        time.Minute,
	})

	fmt.Println("Distributed Token Management Example")
	fmt.Println("==================================")

	ctx := context.Background()

	// Create a new token with rich metadata
	newToken := &token.Token{
		Type:    token.Access,
		Subject: "user123",
		Issuer:  "example-service",
		Metadata: &token.Metadata{
			Device: &token.DeviceInfo{
				ID:        "device123",
				UserAgent: "ExampleApp/1.0",
				IPAddress: "192.168.1.1",
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
		Claims: token.Claims{
			Audience: []string{"example-app"},
			Scopes:   []string{"read", "write"},
		},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	fmt.Println("\n1. Creating and storing token")
	fmt.Println("--------------------------")
	if err := store.Save(ctx, newToken, time.Hour); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}
	fmt.Printf("Token created: %s\n", newToken.ID)

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

	fmt.Println("\n3. Listing tokens with filters")
	fmt.Println("--------------------------")
	filter := &token.Filter{
		Types:   []token.Type{token.Access},
		Subject: "user123",
		Tags:    []string{"mobile"},
		Labels: map[string]string{
			"environment": "production",
		},
	}

	tokens, err := store.List(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to list tokens: %v", err)
	}
	fmt.Printf("Found %d matching tokens\n", len(tokens))

	fmt.Println("\n4. Revoking token")
	fmt.Println("---------------")
	retrieved.RevocationStatus = &token.RevocationStatus{
		RevokedAt: time.Now(),
		Reason:    "user logout",
		RevokedBy: "example-app",
	}
	if err := store.Save(ctx, retrieved, time.Hour); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}
	fmt.Println("Token revoked successfully")

	fmt.Println("\n5. Verifying revocation")
	fmt.Println("---------------------")
	revokedToken, err := store.Get(ctx, newToken.ID)
	if err != nil {
		log.Fatalf("Failed to get revoked token: %v", err)
	}
	if revokedToken.RevocationStatus != nil {
		fmt.Printf("Token revoked at: %v\n", revokedToken.RevocationStatus.RevokedAt)
		fmt.Printf("Revocation reason: %s\n", revokedToken.RevocationStatus.Reason)
	}

	fmt.Println("\n6. Demonstrating validation failure")
	fmt.Println("--------------------------------")
	if err := validator.Validate(ctx, revokedToken); err != nil {
		fmt.Printf("Expected validation failure: %v\n", err)
	}
}