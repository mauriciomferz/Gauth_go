package benchmarks

import (
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

func BenchmarkAuthFlow(b *testing.B) {
       config := gauth.Config{
	       AuthServerURL:     "https://auth.example.com",
	       ClientID:          "bench-client",
	       ClientSecret:      "bench-secret",
	       AccessTokenExpiry: time.Hour,
       }

       auth, err := gauth.New(&config, audit.NewLogger(100))
       if err != nil {
	       b.Skipf("Failed to initialize GAuth: %v", err)
       }
       server := gauth.NewResourceServer("bench-resource", auth)

       b.ResetTimer()

       b.Run("CompleteFlow", func(b *testing.B) {
	       for i := 0; i < b.N; i++ {
		       // Step 1: Authorization
		       grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			       ClientID: "bench-client",
			       Scopes:   []string{"read"},
		       })
		       if err != nil {
			       b.Fatalf("InitiateAuthorization failed: %v", err)
		       }

		       // Step 2: Token Request
		       tokenResp, err := auth.RequestToken(gauth.TokenRequest{
			       GrantID: grant.GrantID,
			       Scope:   grant.Scope,
		       })
		       if err != nil {
			       b.Fatalf("RequestToken failed: %v", err)
		       }

		       // Step 3: Transaction
		       tx := gauth.TransactionDetails{
			       Type:       "payment",
			       Amount:     100.0,
			       ResourceID: "bench-resource",
			       Timestamp:  time.Now(),
		       }

		       if _, err := server.ProcessTransaction(tx, tokenResp.Token); err != nil {
			       b.Errorf("ProcessTransaction failed: %v", err)
		       }
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

       auth, err := gauth.New(&config, audit.NewLogger(100))
       if err != nil {
	       b.Skipf("Failed to initialize GAuth: %v", err)
       }
       server := gauth.NewResourceServer("bench-resource", auth)

       // Create a token
       grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
	       ClientID: "bench-client",
	       Scopes:   []string{"read", "write"},
       })
       if err != nil {
	       b.Skipf("InitiateAuthorization failed: %v", err)
       }

       tokenResp, err := auth.RequestToken(gauth.TokenRequest{
	       GrantID: grant.GrantID,
	       Scope:   grant.Scope,
       })
       if err != nil {
	       b.Skipf("RequestToken failed: %v", err)
       }

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
			       if _, err := server.ProcessTransaction(tx, tokenResp.Token); err != nil {
				       b.Errorf("ProcessTransaction failed: %v", err)
			       }
		       }
	       })
       })
}
