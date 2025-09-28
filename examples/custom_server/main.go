package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// CustomResourceServer extends the basic ResourceServer with additional functionality
type CustomResourceServer struct {
	*gauth.ResourceServer
	customMetrics map[string]int64
}

func NewCustomResourceServer(id string, auth *gauth.GAuth) *CustomResourceServer {
	return &CustomResourceServer{
		ResourceServer: gauth.NewResourceServer(id, auth),
		customMetrics:  make(map[string]int64),
	}
}

func (c *CustomResourceServer) ProcessCustomTransaction(tx gauth.TransactionDetails, token string) error {
	// Custom validation
	if tx.Amount <= 0 {
		return fmt.Errorf("invalid amount: %f", tx.Amount)
	}

	// Process using base implementation
	result, err := c.ResourceServer.ProcessTransaction(tx, token)
	if err != nil {
		return err
	}

	// Custom metrics
	c.customMetrics["total_transactions"]++
	c.customMetrics["total_amount"] += int64(tx.Amount)

	log.Printf("Transaction processed: %v, Result: %v", tx, result)
	return nil
}

func main() {
	// Initialize GAuth
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "custom-client",
		ClientSecret:      "custom-secret",
		Scopes:            []string{"custom:write"},
		AccessTokenExpiry: time.Hour,
	}

	auth, err := gauth.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize GAuth: %v", err)
	}

	// Create custom resource server
	server := NewCustomResourceServer("custom-resource", auth)

	// Get authorization
	authReq := gauth.AuthorizationRequest{
		ClientID: "custom-client",
		Scopes:   []string{"custom:write"},
	}

	grant, err := auth.InitiateAuthorization(authReq)
	if err != nil {
		log.Fatalf("Authorization failed: %v", err)
	}

	// Get token
	tokenResp, err := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})
	if err != nil {
		log.Fatalf("Token request failed: %v", err)
	}

	// Process custom transactions
	transactions := []gauth.TransactionDetails{
		{
			Type:   "special_payment",
			Amount: 150.0,
			CustomMetadata: map[string]string{
				"purpose": "premium_service",
			},
		},
		{
			Type:   "bulk_transfer",
			Amount: 300.0,
			CustomMetadata: map[string]string{
				"recipients": "5",
			},
		},
	}

	for _, tx := range transactions {
		if err := server.ProcessCustomTransaction(tx, tokenResp.Token); err != nil {
			log.Printf("Transaction failed: %v", err)
			continue
		}
	}

	// Print metrics
	log.Printf("Total transactions: %d", server.customMetrics["total_transactions"])
	log.Printf("Total amount processed: %d", server.customMetrics["total_amount"])
}
