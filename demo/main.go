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
	
	// Create a GAuth instance with type-safe options
	options := gauth.Options{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "demo-client",
		ClientSecret:      "demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: 3600,
		EnableTracing:     true, // Enable detailed tracing for demo
	}
	authService := gauth.New(options)

	// Simulate an authorization request and grant with proper typing
	authReq := gauth.AuthorizationRequest{
		ClientID:        "demo-client",
		ClientOwnerID:   "demo-owner",
		ResourceOwnerID: "resource-owner",
		RequestDetails:  "Request to execute payment transaction",
		Scopes:          []string{"transaction:execute"},
		Timestamp:       time.Now().UnixNano() / int64(time.Millisecond),
	}
	
	fmt.Println("\n1. Requesting Authorization")
	authGrant, err := authService.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("Error requesting authorization:", err)
		return
	}
	fmt.Println("✓ Authorization granted")
	fmt.Printf("  - Grant ID: %s\n", authGrant.ID)
	fmt.Printf("  - Scopes: %v\n", authGrant.Scope)
	fmt.Printf("  - Expires: %v\n", authGrant.ExpiresAt.Format(time.RFC3339))
	
	// Issue a token using the grant with typed request
	fmt.Println("\n2. Requesting Token")
	tokenReq := gauth.TokenRequest{
		GrantID:       authGrant.ID,
		ClientID:      "demo-client",
		Scope:         authGrant.Scope,
		Restrictions:  authGrant.Restrictions,
		RequestedType: "transaction",
	}
	
	tokenResp, err := authService.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("Error issuing token:", err)
		return
	}
	fmt.Println("✓ Token issued")
	fmt.Printf("  - Token ID: %s\n", tokenResp.ID)
	fmt.Printf("  - Type: %s\n", tokenResp.Type)
	fmt.Printf("  - Expires: %v\n", tokenResp.ExpiresAt.Format(time.RFC3339))
	
	// Create a typed transaction
	fmt.Println("\n3. Creating Transaction")
	transaction := gauth.Transaction{
		ID:     "tx-12345",
		Type:   "payment",
		Amount: 50.0,
		Date:   time.Now(),
		Metadata: map[string]string{
			"currency": "USD",
			"purpose":  "demo",
		},
	}
	fmt.Println("✓ Transaction created")
	fmt.Printf("  - ID: %s\n", transaction.ID)
	fmt.Printf("  - Type: %s\n", transaction.Type)
	fmt.Printf("  - Amount: %.2f\n", transaction.Amount)
	
	// Create a resource server with typed configuration
	fmt.Println("\n4. Initializing Resource Server")
	serverOptions := gauth.ResourceServerOptions{
		ID:          "demo-resource",
		AuthService: authService,
		Permissions: []string{"transaction:execute"},
		AuditConfig: &gauth.AuditConfig{
			Enabled:     true,
			DetailLevel: "high",
		},
	}
	resourceServer := gauth.NewResourceServer(serverOptions)
	fmt.Println("✓ Resource server initialized")
	
	// Process the transaction
	fmt.Println("\n5. Processing Transaction")
	result, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("✗ Transaction failed:", err)
	} else {
		fmt.Println("✓ Transaction succeeded")
		fmt.Printf("  - Status: %s\n", result.Status)
		fmt.Printf("  - Message: %s\n", result.Message)
	}

	// Show recent audit events
	fmt.Println("\n6. Audit Log:")
	events := resourceServer.GetRecentAuditEvents(5)
	for i, event := range events {
		fmt.Printf("  %d. [%s] %s - %s\n", 
			i+1, 
			event.Time.Format("15:04:05"),
			event.Action,
			event.Result,
		)
	}

	// Wait to show token expiry
	fmt.Println("\n7. Testing Token Expiration")
	fmt.Println("  Waiting for token to expire...")
	
	// For demo purposes, we manually expire the token
	err = authService.ExpireToken(tokenResp.Token)
	if err != nil {
		fmt.Println("Error expiring token:", err)
		return
	}
	time.Sleep(500 * time.Millisecond)
	
	// Try processing with expired token
	fmt.Println("  Attempting transaction with expired token...")
	_, err = resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("✓ Transaction correctly rejected with expired token")
		fmt.Printf("  - Error: %v\n", err)
	} else {
		fmt.Println("✗ Transaction unexpectedly succeeded with expired token")
	}
	
	fmt.Println("\nDemo completed successfully!")
}
	authGrant, err := gauth.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("Error requesting authorization:", err)
		return
	}
	fmt.Println("Authorization grant:", authGrant)

	// Issue a token using the grant
	tokenResp, err := gauth.RequestExtendedToken(map[string]interface{}{
		"scope":        authGrant.Scope,
		"restrictions": authGrant.Restrictions,
	})
	if err != nil {
		fmt.Println("Error issuing token:", err)
		return
	}
	token := tokenResp["token"].(string)
	fmt.Println("Issued token:", token)

	// Prepare transaction details using the new struct
	transaction := Gauth_go.TransactionDetails{
		Type:   "payment",
		Amount: 50.0,
	}

	// Create a ResourceServer
	resourceServer := Gauth_go.NewResourceServer("demo-resource", nil, gauth)

	// Process a transaction
	result, err := resourceServer.ProcessTransaction(transaction, token)
	if err != nil {
		fmt.Println("Transaction failed:", err)
	} else {
		fmt.Println("Transaction result:", result)
	}

	// Show recent audit events
	fmt.Println("\nAudit log:")
	resourceServer.AuditLogger.PrintRecentEvents(5)

	// Wait to show token expiry
	fmt.Println("\nWaiting for token to expire...")
	time.Sleep(2 * time.Second)
	gauth.TokenStoreMutex.Lock()
	gauth.TokenStore[token].ValidUntil = time.Now().Add(-1 * time.Second)
	gauth.TokenStoreMutex.Unlock()

	// Try processing with expired token
	_, err = resourceServer.ProcessTransaction(transaction, token)
	if err != nil {
		fmt.Println("Transaction with expired token failed as expected:", err)
	}
}