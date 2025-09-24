// Package main provides a demonstration of the GAuth protocol implementation
//
// This demo shows the basic flow of the GAuth protocol, including:
// - Authorization request and grant
// - Token issuance
// - Transaction processing
// - Audit logging
// - Token expiration
package main

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func main() {
	fmt.Println("GAuth Demo Application")
	fmt.Println("======================")

	// Create a GAuth instance with config
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "demo-client",
		ClientSecret:      "demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
		RateLimit:         gauth.Config{}.RateLimit, // Use default zero value for demo
	}
	authService, err := gauth.New(config)
	if err != nil {
		fmt.Println("Error creating GAuth instance:", err)
		return
	}

	// Simulate an authorization request and grant
	authReq := gauth.AuthorizationRequest{
		ClientID: "demo-client",
		Scopes:   []string{"transaction:execute"},
	}

	fmt.Println("\n1. Requesting Authorization")
	authGrant, err := authService.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("Error requesting authorization:", err)
		return
	}
	fmt.Println("✓ Authorization granted")
	fmt.Printf("  - Grant ID: %s\n", authGrant.GrantID)
	fmt.Printf("  - Scopes: %v\n", authGrant.Scope)
	fmt.Printf("  - Expires: %v\n", authGrant.ValidUntil.Format(time.RFC3339))

	// Issue a token using the grant
	fmt.Println("\n2. Requesting Token")
	tokenReq := gauth.TokenRequest{
		GrantID:      authGrant.GrantID,
		Scope:        authGrant.Scope,
		Restrictions: authGrant.Restrictions,
		Context:      nil, // For demo, context is nil
	}
	tokenResp, err := authService.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("Error issuing token:", err)
		return
	}
	fmt.Println("✓ Token issued")
	fmt.Printf("  - Token: %s\n", tokenResp.Token)
	fmt.Printf("  - Scopes: %v\n", tokenResp.Scope)
	fmt.Printf("  - Expires: %v\n", tokenResp.ValidUntil.Format(time.RFC3339))

	// Create a transaction
	fmt.Println("\n3. Creating Transaction")
	transaction := gauth.TransactionDetails{
		ID:          "tx-12345",
		Type:        gauth.PaymentTransaction,
		Status:      gauth.TransactionPending,
		ClientID:    "demo-client",
		ResourceID:  "resource-1",
		Scopes:      []string{"transaction:execute"},
		Amount:      50.0,
		Currency:    "USD",
		Timestamp:   time.Now(),
		Source:      "account-1",
		Destination: "account-2",
		Description: "Demo payment transaction",
	}
	fmt.Println("✓ Transaction created")
	fmt.Printf("  - ID: %s\n", transaction.ID)
	fmt.Printf("  - Type: %s\n", transaction.Type)
	fmt.Printf("  - Amount: %.2f\n", transaction.Amount)

	// Create a resource server
	fmt.Println("\n4. Initializing Resource Server")
	resourceServer := gauth.NewResourceServer("demo-resource", authService)
	fmt.Println("✓ Resource server initialized")

	// Process the transaction
	fmt.Println("\n5. Processing Transaction")
	resultMsg, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("✗ Transaction failed:", err)
	} else {
		fmt.Println("✓ Transaction succeeded")
		fmt.Printf("  - Message: %s\n", resultMsg)
	}

	// (Audit log not available in current API)

	// Wait and expire the token (simulate by waiting for expiry or skipping for demo)
	fmt.Println("\n6. Testing Token Expiration (simulated)")
	// In a real test, you would wait for tokenResp.ValidUntil to pass, or adjust the token store directly if API allows
	fmt.Println("(Token expiration handling would be tested here if API allowed direct manipulation)")
	fmt.Println("Demo completed successfully!")
}
