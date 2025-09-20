// Package main provides a basic example of using the GAuth library.
// This example demonstrates the core functionality of GAuth with a focus on clarity and usability.
package main

import (
	"os"
	"time"

	"golang.org/x/exp/slog"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/common"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

func main() {

	// Set up structured logger (slog)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting GAuth Basic Example")

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
		 logger.Error("Failed to create GAuth service", "error", err)
		 os.Exit(1)
	}

	// Simulate an authorization request and grant
	authReq := gauth.AuthorizationRequest{
		ClientID: "example-client",
		Scopes:   []string{"payment:execute"},
	}

       logger.Info("Initiating Authorization Request")
       authGrant, err := gauthService.InitiateAuthorization(authReq)
       if err != nil {
	       logger.Error("Error requesting authorization", "error", err)
	       return
       }
       logger.Info("Authorization grant received", "grant_id", authGrant.GrantID, "scopes", authGrant.Scope)

	// To extend this example:
	// - Add token issuance and validation using gauthService.RequestToken(...)
	// - Integrate audit logging and error handling
	// - See the README for more extension ideas
}
