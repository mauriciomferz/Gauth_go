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
				ClientID: "bench-client",
				Scopes:   []string{"read"},
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

			_, _ = server.ProcessTransaction(tx, tokenResp.Token)
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
		ClientID: "bench-client",
		Scopes:   []string{"read", "write"},
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
				_, _ = server.ProcessTransaction(tx, tokenResp.Token)
			}
		})
	})
}
