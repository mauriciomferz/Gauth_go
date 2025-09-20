package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)


func main() {
	fmt.Println("GAuth RFC111 Demo Application")
	fmt.Println("==============================")

	// 1. Initialize GAuth service with current Config
	config := &gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "demo-client",
		ClientSecret:      "demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
	}
	fmt.Println("1. Initializing GAuth service...")
	gauthService, err := gauth.New(config, nil)
	if err != nil {
		fmt.Println("   ✗ Failed to initialize GAuth:", err)
		return
	}
	fmt.Println("   ✓ GAuth service initialized")

	// 2. Create an authorization request
	authReq := gauth.AuthorizationRequest{
		ClientID: "demo-client",
		Scopes:   []string{"transaction:execute"},
	}
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

	// 3. Request a token with the grant
	fmt.Println("\n3. Requesting token...")
	tokenReq := gauth.TokenRequest{
		GrantID:      authGrant.GrantID,
		Scope:        authGrant.Scope,
		Restrictions: authGrant.Restrictions,
		Context:      context.Background(),
	}
	tokenResp, err := gauthService.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("   ✗ Token request failed:", err)
		return
	}
	fmt.Println("   ✓ Token issued")
	fmt.Printf("     - Token: %s\n", tokenResp.Token)
	fmt.Printf("     - Expires: %v\n", tokenResp.ValidUntil.Format(time.RFC3339))

	// 4. Create a transaction (TransactionDetails)
	fmt.Println("\n4. Creating transaction...")
	transaction := gauth.TransactionDetails{
		ID:         "tx-12345",
		Type:       gauth.PaymentTransaction,
		Status:     gauth.TransactionPending,
		ClientID:   "demo-client",
		ResourceID: "resource-1",
		Scopes:     []string{"transaction:execute"},
		Amount:     50.0,
		Currency:   "USD",
		Timestamp:  time.Now(),
		Description: "product purchase",
		CustomMetadata: map[string]string{
			"purpose": "product purchase",
		},
	}
	fmt.Println("   ✓ Transaction created")
	fmt.Printf("     - ID: %s\n", transaction.ID)
	fmt.Printf("     - Type: %s\n", transaction.Type)
	fmt.Printf("     - Amount: %.2f\n", transaction.Amount)

	// 5. Create a resource server
	fmt.Println("\n5. Creating resource server...")
	resourceServer := gauth.NewResourceServer("demo-resource", gauthService)
	fmt.Println("   ✓ Resource server initialized")

	// 6. Process the transaction
	fmt.Println("\n6. Processing transaction...")
	result, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("   ✗ Transaction failed:", err)
	} else {
		fmt.Println("   ✓ Transaction succeeded")
		fmt.Printf("     - Result: %s\n", result)
	}

	// 7. Test token expiration (simulate by waiting for expiry or skipping)
	fmt.Println("\n7. Token expiration test (skipped in demo)")
	// In a real test, you would wait for expiry or revoke the token, then retry

	fmt.Println("\nDemo completed successfully!")
}
