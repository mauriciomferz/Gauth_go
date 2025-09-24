package integration

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func setupTestAuth(t *testing.T) *gauth.GAuth {
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "test-client",
		ClientSecret:      "test-secret",
		Scopes:            []string{"read", "write", "admin", "transaction:execute"},
		AccessTokenExpiry: time.Hour,
	}

	auth, err := gauth.New(config)
	if err != nil {
		t.Fatalf("Failed to create GAuth instance: %v", err)
	}
	return auth
}

func TestCompleteAuthFlow(t *testing.T) {
	auth := setupTestAuth(t)

	// Step 1: Request Authorization
	authReq := gauth.AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"read", "write", "transaction:execute"},
	}

	grant, err := auth.InitiateAuthorization(authReq)
	if err != nil {
		t.Fatalf("Authorization request failed: %v", err)
	}

	// Step 2: Request Token
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	}

	tokenResp, err := auth.RequestToken(tokenReq)
	if err != nil {
		t.Fatalf("Token request failed: %v", err)
	}

	// Step 3: Create Resource Server
	server := gauth.NewResourceServer("test-resource", auth)

	// Step 4: Process Transaction
	tx := gauth.TransactionDetails{
		Type:       "payment",
		Amount:     100.0,
		ResourceID: "test-resource",
		Timestamp:  time.Now(),
	}

	result, err := server.ProcessTransaction(tx, tokenResp.Token)
	if err != nil {
		t.Fatalf("Transaction processing failed: %v", err)
	}

	// Verify transaction result
	if result == "" {
		t.Fatal("Expected non-empty transaction result")
	}
}

func TestConcurrentTransactions(t *testing.T) {
	auth := setupTestAuth(t)
	server := gauth.NewResourceServer("test-resource", auth)

	// Get a valid token
	grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"read", "write", "transaction:execute"},
	})

	tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})

	// Run concurrent transactions
	concurrency := 10
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			tx := gauth.TransactionDetails{
				Type:       "payment",
				Amount:     float64(index * 10),
				ResourceID: "test-resource",
				Timestamp:  time.Now(),
			}

			_, err := server.ProcessTransaction(tx, tokenResp.Token)
			if err != nil {
				t.Errorf("Concurrent transaction %d failed: %v", index, err)
			}
			done <- true
		}(i)
	}

	// Wait for all transactions
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestResourceServerFailover(t *testing.T) {
	auth := setupTestAuth(t)

	// Create multiple resource servers
	servers := []*gauth.ResourceServer{
		gauth.NewResourceServer("server-1", auth),
		gauth.NewResourceServer("server-2", auth),
		gauth.NewResourceServer("server-3", auth),
	}

	// Get a valid token
	grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"read", "write", "transaction:execute"},
	})

	tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})

	// Simulate server failures and failover
	tx := gauth.TransactionDetails{
		Type:       "payment",
		Amount:     100.0,
		ResourceID: "test-resource",
		Timestamp:  time.Now(),
	}

	var lastResult interface{}
	var lastErr error

	for _, server := range servers {
		lastResult, lastErr = server.ProcessTransaction(tx, tokenResp.Token)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		t.Fatalf("All servers failed to process transaction: %v", lastErr)
	}
	if lastResult == nil {
		t.Fatal("Expected non-nil transaction result after failover")
	}
}
