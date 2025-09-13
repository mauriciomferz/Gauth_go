package benchmarks

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func BenchmarkAuthFlow(b *testing.B) {
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "bench-client",
		ClientSecret:      "bench-secret",
		Scopes:            []string{"read", "write"},
		AccessTokenExpiry: time.Hour,
	}

	auth, _ := gauth.New(config)
	server := gauth.NewResourceServer("bench-resource", auth)

	b.ResetTimer()

	b.Run("CompleteFlow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Step 1: Authorization
			grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
				ClientID:        "bench-client",
				ClientOwnerID:   "owner-1",
				ResourceOwnerID: "resource-1",
				Scopes:          []string{"read"},
			})

			// Step 2: Token Request
			tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
				GrantID: grant.GrantID,
				Scope:   grant.Scope,
			})

			// Step 3: Transaction
			tx := gauth.TransactionDetails{
				Type:       "payment",
				Amount:     100.0,
				ResourceID: "bench-resource",
				Timestamp:  time.Now(),
			}

			server.ProcessTransaction(tx, tokenResp.Token)
		}
	})
}

func BenchmarkTokenValidation(b *testing.B) {
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "bench-client",
		ClientSecret:      "bench-secret",
		AccessTokenExpiry: time.Hour,
	}

	auth, _ := gauth.New(config)

	// Create a token first
	grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
		ClientID:        "bench-client",
		ClientOwnerID:   "owner-1",
		ResourceOwnerID: "resource-1",
		Scopes:          []string{"read"},
	})

	tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})

	b.ResetTimer()

	b.Run("TokenValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			auth.ValidateToken(tokenResp.Token)
		}
	})
}

func BenchmarkConcurrentTransactions(b *testing.B) {
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "bench-client",
		ClientSecret:      "bench-secret",
		AccessTokenExpiry: time.Hour,
	}

	auth, _ := gauth.New(config)
	server := gauth.NewResourceServer("bench-resource", auth)

	// Create a token
	grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
		ClientID:        "bench-client",
		ClientOwnerID:   "owner-1",
		ResourceOwnerID: "resource-1",
		Scopes:          []string{"read", "write"},
	})

	tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})

	b.ResetTimer()

	b.Run("ConcurrentTransactions", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := gauth.TransactionDetails{
					Type:       "payment",
					Amount:     100.0,
					ResourceID: "bench-resource",
					Timestamp:  time.Now(),
				}
				server.ProcessTransaction(tx, tokenResp.Token)
			}
		})
	})
}

func BenchmarkRateLimiting(b *testing.B) {
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "bench-client",
		ClientSecret:      "bench-secret",
		AccessTokenExpiry: time.Hour,
	}

	auth, _ := gauth.New(config)
	server := gauth.NewResourceServer("bench-resource", auth)

	// Configure rate limits
	server.SetRateLimit(100, time.Second) // 100 requests per second

	// Create a token
	grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{
		ClientID:        "bench-client",
		ClientOwnerID:   "owner-1",
		ResourceOwnerID: "resource-1",
		Scopes:          []string{"read"},
	})

	tokenResp, _ := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})

	tx := gauth.TransactionDetails{
		Type:       "payment",
		Amount:     100.0,
		ResourceID: "bench-resource",
		Timestamp:  time.Now(),
	}

	b.ResetTimer()

	b.Run("RateLimitedTransactions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			server.ProcessTransaction(tx, tokenResp.Token)
		}
	})
}
