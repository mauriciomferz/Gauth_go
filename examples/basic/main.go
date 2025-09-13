// Package main provides a basic example of using the GAuth library.
// This example demonstrates the core functionality of GAuth with a focus on clarity and usability.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
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
		RateLimit: gauth.RateLimitConfig{
			RequestsPerSecond: 10,
			BurstSize:         5,
			WindowSize:        1,
		},
	}
	gauthService, err := gauth.New(config)
	if err != nil {
		log.Fatalf("Failed to create GAuth service: %v", err)
	}

	// Simulate an authorization request and grant
	authReq := gauth.AuthorizationRequest{
		ClientID:        "example-client",
		ClientOwnerID:   "example-owner",
		ResourceOwnerID: "resource-owner",
		RequestDetails:  "Request to execute payment transaction",
		Timestamp:       time.Now().UnixNano() / 1e6,
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

	// ctx := context.Background() // Not used in this example

	// The rest of the example should be updated to use the GAuth API for token issuance, validation, etc.
	// ...existing code...

	// The rest of the example (transaction, resource server, etc.) is commented out because the types and methods do not exist in the current GAuth API.
	// transaction := gauth.TransactionDetails{
	// 	Type:       "payment",
	// 	Amount:     50.0,
	// 	ResourceID: "demo-resource",
	// 	Timestamp:  time.Now(),
	// }
	// resourceServer := gauth.NewResourceServer("demo-resource", auth)
	// result, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	// if err != nil {
	// 	log.Printf("Transaction failed: %v", err)
	// } else {
	// 	fmt.Printf("Transaction result: %+v\n", result)
	// }
	// fmt.Println("\nRecent audit events:")
	// events := resourceServer.GetRecentAuditEvents(5)
	// for _, event := range events {
	// 	fmt.Printf("%+v\n", event)
	// }
	// fmt.Println("\nWaiting for token to expire...")
	// time.Sleep(2 * time.Second)
	// _, err = resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	// if err != nil {
	// 	fmt.Printf("Transaction with expired token failed as expected: %v\n", err)
	// }
}
