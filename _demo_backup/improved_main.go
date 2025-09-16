package main

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	fmt.Println("GAuth RFC111 Demo Application")
	fmt.Println("==============================")

	// Create a GAuth instance with typed options
	options := &gauth.Options{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "demo-client",
		ClientSecret:      "demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: 3600,
		EnableTracing:     true,
	}

	// Initialize the GAuth service
	fmt.Println("1. Initializing GAuth service...")
	gauthService := gauth.New(options)
	fmt.Println("   ✓ GAuth service initialized")

	// Create an authorization request with proper typing
	authReq := gauth.AuthorizationRequest{
		ClientID:        "demo-client",
		ClientOwnerID:   "demo-owner",
		ResourceOwnerID: "resource-owner",
		RequestDetails:  "Request to execute payment transaction",
		Scopes:          []string{"transaction:execute"},
		Timestamp:       time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Request authorization
	fmt.Println("\n2. Requesting authorization...")
	authGrant, err := gauthService.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("   ✗ Authorization request failed:", err)
		return
	}
	fmt.Println("   ✓ Authorization grant received")
	fmt.Printf("     - Grant ID: %s\n", authGrant.ID)
	fmt.Printf("     - Scopes: %v\n", authGrant.Scope)
	fmt.Printf("     - Expires: %v\n", authGrant.ExpiresAt.Format(time.RFC3339))

	// Request a token with the grant
	fmt.Println("\n3. Requesting token...")
	tokenReq := gauth.TokenRequest{
		GrantID:       authGrant.ID,
		ClientID:      "demo-client",
		Scope:         authGrant.Scope,
		Restrictions:  authGrant.Restrictions,
		RequestedType: string(token.AccessToken),
	}

	tokenResp, err := gauthService.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("   ✗ Token request failed:", err)
		return
	}
	fmt.Println("   ✓ Token issued")
	fmt.Printf("     - Token ID: %s\n", tokenResp.ID)
	fmt.Printf("     - Type: %s\n", tokenResp.Type)
	fmt.Printf("     - Expires: %v\n", tokenResp.ExpiresAt.Format(time.RFC3339))

	// Create a typed transaction
	fmt.Println("\n4. Creating transaction...")
	transaction := gauth.Transaction{
		ID:     "tx-12345",
		Type:   "payment",
		Amount: 50.0,
		Date:   time.Now(),
		Metadata: map[string]string{
			"currency": "USD",
			"purpose":  "product purchase",
		},
	}
	fmt.Println("   ✓ Transaction created")
	fmt.Printf("     - ID: %s\n", transaction.ID)
	fmt.Printf("     - Type: %s\n", transaction.Type)
	fmt.Printf("     - Amount: %.2f\n", transaction.Amount)

	// Create a resource server
	fmt.Println("\n5. Creating resource server...")
	serverOptions := gauth.ResourceServerOptions{
		ID:          "demo-resource",
		AuthService: gauthService,
		Permissions: []string{"transaction:execute"},
		AuditConfig: &gauth.AuditConfig{
			Enabled:     true,
			DetailLevel: "high",
		},
	}
	resourceServer := gauth.NewResourceServer(serverOptions)
	fmt.Println("   ✓ Resource server initialized")

	// Process the transaction
	fmt.Println("\n6. Processing transaction...")
	result, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("   ✗ Transaction failed:", err)
	} else {
		fmt.Println("   ✓ Transaction succeeded")
		fmt.Printf("     - Status: %s\n", result.Status)
		fmt.Printf("     - Message: %s\n", result.Message)
	}

	// Show recent audit events
	fmt.Println("\n7. Retrieving audit log...")
	auditEvents := resourceServer.GetRecentAuditEvents(5)
	fmt.Println("   ✓ Audit log retrieved")
	for i, event := range auditEvents {
		fmt.Printf("     %d. [%s] %s - %s\n",
			i+1,
			event.Time.Format("15:04:05"),
			event.Action,
			event.Result,
		)
	}

	// Test token expiration
	fmt.Println("\n8. Testing token expiration...")
	fmt.Println("   Manually expiring token...")
	err = gauthService.ExpireToken(tokenResp.Token)
	if err != nil {
		fmt.Println("   ✗ Failed to expire token:", err)
		return
	}

	// Give a moment for expiration to take effect
	time.Sleep(500 * time.Millisecond)

	// Try processing with expired token
	fmt.Println("   Attempting transaction with expired token...")
	_, err = resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("   ✓ Transaction correctly rejected with expired token")
		fmt.Printf("     - Error: %v\n", err)
	} else {
		fmt.Println("   ✗ Transaction unexpectedly succeeded with expired token")
	}

	fmt.Println("\nDemo completed successfully!")
}
