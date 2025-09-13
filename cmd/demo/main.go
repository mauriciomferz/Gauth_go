package main

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func main() {
	fmt.Println("Starting GAuth Demo Application")

	// Create a GAuth instance with configuration
	auth := gauth.New(gauth.Options{
		AuthServerURL:    "https://auth.example.com",
		ClientID:         "demo-client",
		ClientSecret:     "demo-secret",
		Scopes:           []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: 3600,
	})

	// Simulate an authorization request and grant
	authReq := gauth.AuthorizationRequest{
		ClientID:        "demo-client",
		ClientOwnerID:   "demo-owner",
		ResourceOwnerID: "resource-owner",
		RequestDetails:  "Request to execute payment transaction",
		Timestamp:       time.Now().UnixNano() / 1e6,
	}
	
	fmt.Println("\n1. Initiating Authorization Request")
	authGrant, err := auth.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("❌ Error requesting authorization:", err)
		return
	}
	fmt.Println("✓ Authorization grant received:", authGrant.ID)
	fmt.Printf("  - Granted scopes: %v\n", authGrant.Scope)
	fmt.Printf("  - Expires at: %v\n", authGrant.ExpiresAt)

	// Issue a token using the grant
	fmt.Println("\n2. Requesting Extended Token")
	tokenReq := gauth.TokenRequest{
		GrantID:      authGrant.ID,
		Scope:        authGrant.Scope,
		Restrictions: authGrant.Restrictions,
	}
	
	tokenResp, err := auth.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("❌ Error issuing token:", err)
		return
	}
	fmt.Println("✓ Token issued successfully:", tokenResp.ID)
	fmt.Printf("  - Token type: %v\n", tokenResp.Type)
	fmt.Printf("  - Expires at: %v\n", tokenResp.ExpiresAt)

	// Create a resource server
	fmt.Println("\n3. Creating Resource Server")
	resourceServer := gauth.NewResourceServer(gauth.ResourceServerOptions{
		ID:        "demo-resource",
		AuthService: auth,
	})
	fmt.Println("✓ Resource server created")

	// Prepare and process a transaction
	fmt.Println("\n4. Processing Transaction")
	transaction := gauth.Transaction{
		ID:     "tx-12345",
		Type:   "payment",
		Amount: 50.0,
		Date:   time.Now(),
	}
	
	result, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("❌ Transaction failed:", err)
	} else {
		fmt.Println("✓ Transaction processed successfully")
		fmt.Printf("  - Status: %s\n", result.Status)
		fmt.Printf("  - Message: %s\n", result.Message)
	}

	// Show recent audit events
	fmt.Println("\n5. Audit Log")
	auditEvents := resourceServer.GetRecentAuditEvents(5)
	for i, event := range auditEvents {
		fmt.Printf("  %d. [%s] %s - %s\n", 
			i+1, 
			event.Time.Format("15:04:05"), 
			event.Action,
			event.Result,
		)
	}

	// Wait to show token expiry
	fmt.Println("\n6. Testing Token Expiration")
	fmt.Println("  Simulating token expiry...")
	
	// For demo, we manually expire the token
	auth.ExpireToken(tokenResp.Token)
	time.Sleep(1 * time.Second)
	
	// Try processing with expired token
	_, err = resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("✓ Transaction with expired token failed as expected")
		fmt.Printf("  - Error: %v\n", err)
	}
	
	fmt.Println("\nDemo completed successfully!")
}
}