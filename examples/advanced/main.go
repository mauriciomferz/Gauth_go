package main

import (
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func main() {
	// Create a GAuth instance with custom rate limits and token expiry
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "advanced-client",
		ClientSecret:      "advanced-secret",
		Scopes:            []string{"read", "write", "admin"},
		AccessTokenExpiry: 30 * time.Minute,
	}

	auth, err := gauth.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize GAuth: %v", err)
	}

	// Create a resource server with custom settings
	server := gauth.NewResourceServer("advanced-resource", auth)
	server.SetRateLimit(100, time.Second) // 100 requests per second

	// Example 1: Multi-scope authorization
	authReq := gauth.AuthorizationRequest{
		ClientID:        "advanced-client",
		ClientOwnerID:   "org-123",
		ResourceOwnerID: "user-456",
		Scopes:          []string{"read", "write"},
		RequestDetails:  "High-privilege access request",
		Timestamp:       time.Now().UnixNano() / 1e6,
	}

	grant, err := auth.InitiateAuthorization(authReq)
	if err != nil {
		log.Fatalf("Authorization failed: %v", err)
	}

	// Example 2: Token request with restrictions
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
		Restrictions: []gauth.Restriction{
			{
				Type:  "ip_range",
				Value: "192.168.1.0/24",
			},
			{
				Type:  "time_window",
				Value: "business_hours",
			},
		},
	}

	tokenResp, err := auth.RequestToken(tokenReq)
	if err != nil {
		log.Fatalf("Token request failed: %v", err)
	}

	// Example 3: Batch transaction processing
	transactions := []gauth.TransactionDetails{
		{
			Type:       "payment",
			Amount:     100.0,
			ResourceID: "resource-1",
			Metadata: map[string]string{
				"currency": "USD",
				"method":   "credit_card",
			},
		},
		{
			Type:       "transfer",
			Amount:     50.0,
			ResourceID: "resource-2",
			Metadata: map[string]string{
				"destination": "account-789",
			},
		},
	}

	// Process transactions in parallel
	resultChan := make(chan error, len(transactions))
	for _, tx := range transactions {
		go func(t gauth.TransactionDetails) {
			_, err := server.ProcessTransaction(t, tokenResp.Token)
			resultChan <- err
		}(tx)
	}

	// Collect results
	for i := 0; i < len(transactions); i++ {
		if err := <-resultChan; err != nil {
			log.Printf("Transaction %d failed: %v", i, err)
		}
	}

	// Example 4: Error handling and token refresh
	for {
		_, err := server.ProcessTransaction(transactions[0], tokenResp.Token)
		if err != nil {
			// Check if token expired
			if gauth.IsTokenExpiredError(err) {
				// Request new token
				newTokenResp, err := auth.RequestToken(tokenReq)
				if err != nil {
					log.Fatalf("Token refresh failed: %v", err)
				}
				tokenResp = newTokenResp
				continue
			}
			log.Fatalf("Transaction failed: %v", err)
		}
		break
	}

	// Example 5: Audit log analysis
	events := server.GetAuditEvents(gauth.AuditQuery{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Types:     []string{"transaction_start", "transaction_complete"},
		ActorID:   "user-456",
	})

	for _, event := range events {
		log.Printf("Audit event: %+v\n", event)
	}
}
