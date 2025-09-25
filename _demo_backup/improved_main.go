package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func main() {
	fmt.Println("GAuth RFC111 Demo Application")
	fmt.Println("==============================")

	// Create a GAuth instance with correct config
	// Use environment variables for secrets in production
	clientSecret := os.Getenv("GAUTH_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = "demo-secret" // Default for development only
	}
	
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "demo-client",
		ClientSecret:      clientSecret,
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
	}

	// Initialize the GAuth service
	fmt.Println("1. Initializing GAuth service...")
	gauthService, err := gauth.New(config)
	if err != nil {
		fmt.Printf("   ✗ Failed to initialize GAuth: %v\n", err)
		return
	}
	fmt.Println("   ✓ GAuth service initialized")

	// Create an authorization request with correct structure
	authReq := gauth.AuthorizationRequest{
		ClientID: "demo-client",
		Scopes:   []string{"transaction:execute"},
	}

	// Request authorization
	fmt.Println("\n2. Requesting authorization...")
	authGrant, err := gauthService.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("   ✗ Authorization request failed:", err)
		return
	}
	fmt.Println("   ✓ Authorization grant received")
	fmt.Printf("     - Grant ID: %s\n", authGrant.GrantID)
	fmt.Printf("     - Scopes: %v\n", authGrant.Scope)
	fmt.Printf("     - Expires: %v\n", authGrant.ValidUntil.Format(time.RFC3339))

	// Request a token with the grant
	fmt.Println("\n3. Requesting token...")
	tokenReq := gauth.TokenRequest{
		GrantID:      authGrant.GrantID,
		Scope:        authGrant.Scope,
		Restrictions: authGrant.Restrictions,
	}

	tokenResp, err := gauthService.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("   ✗ Token request failed:", err)
		return
	}
	fmt.Println("   ✓ Token issued")
	fmt.Printf("     - Token: %s\n", tokenResp.Token[:20]+"...")
	fmt.Printf("     - Scopes: %v\n", tokenResp.Scope)
	fmt.Printf("     - Expires: %v\n", tokenResp.ValidUntil.Format(time.RFC3339))

	// Create a typed transaction
	fmt.Println("\n4. Creating transaction...")
	transaction := gauth.TransactionDetails{
		ID:        "tx-12345",
		Type:      gauth.PaymentTransaction,
		Amount:    50.0,
		Currency:  "USD",
		Timestamp: time.Now(),
		CustomMetadata: map[string]string{
			"purpose": "product purchase",
		},
	}
	fmt.Println("   ✓ Transaction created")
	fmt.Printf("     - ID: %s\n", transaction.ID)
	fmt.Printf("     - Type: %s\n", transaction.Type)
	fmt.Printf("     - Amount: %.2f\n", transaction.Amount)

	// Create a resource server
	fmt.Println("\n5. Creating resource server...")
	resourceServer := gauth.NewResourceServer("demo-resource", gauthService)
	fmt.Println("   ✓ Resource server initialized")

	// Process the transaction
	fmt.Println("\n6. Processing transaction...")
	result, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("   ✗ Transaction failed:", err)
	} else {
		fmt.Println("   ✓ Transaction succeeded")
		fmt.Printf("     - Result: %s\n", result)
	}

	// Demo completed successfully
	fmt.Println("\n7. Demo completed successfully!")
	fmt.Println("   ✓ All operations completed")
	fmt.Println("   ✓ Token-based authentication working")

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
