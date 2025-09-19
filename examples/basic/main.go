// Package main provides a basic example of using the GAuth library.
// This example demonstrates the core functionality of GAuth with a focus on clarity and usability.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/common"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

func main() {
	fmt.Println("Starting GAuth Basic Example")

	// Create a GAuth instance with typed configuration
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "example-client",
		ClientSecret:      "example-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
		RateLimit: common.RateLimitConfig{
			RequestsPerSecond: 10,
			BurstSize:         5,
			WindowSize:        1,
		},
		TokenConfig:       &token.Config{SigningMethod: token.RS256},
	}
       gauthService, err := gauth.New(&config, audit.NewLogger(100))
       if err != nil {
	       log.Fatalf("Failed to create GAuth service: %v", err)
       }

	// Simulate an authorization request and grant
	authReq := gauth.AuthorizationRequest{
		ClientID: "example-client",
		Scopes:   []string{"payment:execute"},
	}

	fmt.Println("\n1. Initiating Authorization Request")
	authGrant, err := gauthService.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("Error requesting authorization:", err)
		return
	}
	fmt.Println("✓ Authorization grant received")
	fmt.Printf("  - Grant ID: %s\n", authGrant.GrantID)
	fmt.Printf("  - Scopes: %v\n", authGrant.Scope)

	// To extend this example:
	// - Add token issuance and validation using gauthService.RequestToken(...)
	// - Integrate audit logging and error handling
	// - See the README for more extension ideas
}
