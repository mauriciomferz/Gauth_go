package benchmarks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/rate"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

// BenchmarkTokenGeneration benchmarks token generation performance
func BenchmarkTokenGeneration(b *testing.B) {
	ctx := context.Background()
	// Use a real RSA private key for signing
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("failed to generate RSA key: %v", err)
	}
	store := token.NewMemoryStore()
	mgr := token.NewService(&token.Config{
		SigningKey:     privKey,
		Store:          store,
		SigningMethod:  token.RS256,
		ValidityPeriod: time.Hour,
	}, store)

       b.ResetTimer()
       for i := 0; i < b.N; i++ {
	       t := &token.Token{
		       Subject:  "user-123",
		       Scopes:   []string{"read", "write"},
		       Issuer:   "test-issuer",
		       Type:     token.Access,
	       }
	       _, err := mgr.Issue(ctx, t)
	       if err != nil {
		       b.Fatal(err)
	       }
       }
}

// BenchmarkTokenValidation benchmarks token validation performance
func BenchmarkTokenValidation(b *testing.B) {
	ctx := context.Background()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("failed to generate RSA key: %v", err)
	}
	store := token.NewMemoryStore()
	mgr := token.NewService(&token.Config{
		SigningKey:     privKey,
		Store:          store,
		SigningMethod:  token.RS256,
		ValidityPeriod: time.Hour,
	}, store)

       // Generate a token for validation with valid times
       now := time.Now()
       t := &token.Token{
	       Subject:   "user-123",
	       Scopes:    []string{"read", "write"},
	       Issuer:    "test-issuer",
	       Type:      token.Access,
	       IssuedAt:  now,
	       NotBefore: now,
	       ExpiresAt: now.Add(time.Hour),
       }
       issued, err := mgr.Issue(ctx, t)
       if err != nil {
	       b.Fatal(err)
       }

       b.ResetTimer()
       for i := 0; i < b.N; i++ {
	       err := mgr.Validate(ctx, issued)
	       if err != nil {
		       b.Fatal(err)
	       }
       }
}

// BenchmarkRateLimiting benchmarks rate limiting performance
func BenchmarkRateLimiting(b *testing.B) {
       ctx := context.Background()
       limiter := rate.NewSlidingWindow(rate.Config{
	       Rate:      100,
	       Window:    time.Second,
	       BurstSize: 10,
       })

       b.ResetTimer()
       for i := 0; i < b.N; i++ {
	       _ = limiter.Allow(ctx, "test-key")
       }
}

// BenchmarkDistributedRateLimiting benchmarks distributed rate limiting
func BenchmarkDistributedRateLimiting(b *testing.B) {
	ctx := context.Background()
	limiter, err := rate.NewDistributedLimiter(rate.Config{
		Rate:      100,
		Window:    time.Second,
		BurstSize: 10,
		DistributedConfig: &rate.RedisConfig{
			Addresses:  []string{"localhost:6379"},
			Password:   "",
			DB:         0,
			KeyPrefix:  "benchmark:",
		},
	})
	if err != nil {
		b.Skipf("Distributed limiter not available: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow(ctx, "test-key")
	}
}

// BenchmarkAuthenticationFlow benchmarks the complete authentication flow
func BenchmarkAuthenticationFlow(b *testing.B) {
	// This is a placeholder; update with actual auth service usage if available
	b.Skip("Authentication flow benchmark not implemented: update with actual service usage.")
}

// BenchmarkParallelTokenGeneration benchmarks parallel token generation
func BenchmarkParallelTokenGeneration(b *testing.B) {
       ctx := context.Background()
       privKey, err := rsa.GenerateKey(rand.Reader, 2048)
       if err != nil {
	       b.Fatalf("failed to generate RSA key: %v", err)
       }
       store := token.NewMemoryStore()
       mgr := token.NewService(&token.Config{
	       SigningKey:     privKey,
	       Store:          store,
	       SigningMethod:  token.RS256,
	       ValidityPeriod: time.Hour,
       }, store)

       b.ResetTimer()
       b.RunParallel(func(pb *testing.PB) {
	       for pb.Next() {
		       t := &token.Token{
			       Subject:  "user-123",
			       Scopes:   []string{"read", "write"},
			       Issuer:   "test-issuer",
			       Type:     token.Access,
		       }
		       _, err := mgr.Issue(ctx, t)
		       if err != nil {
			       b.Fatal(err)
		       }
	       }
       })
}

// BenchmarkParallelRateLimiting benchmarks parallel rate limiting
func BenchmarkParallelRateLimiting(b *testing.B) {
	ctx := context.Background()
       limiter := rate.NewSlidingWindow(rate.Config{
	       Rate:      1000,
	       Window:    time.Second,
	       BurstSize: 100,
       })

       b.ResetTimer()
       b.RunParallel(func(pb *testing.PB) {
	       for pb.Next() {
		       _ = limiter.Allow(ctx, "test-key")
	       }
       })
}

// BenchmarkTokenStore benchmarks token store operations
func BenchmarkTokenStore(b *testing.B) {
	ctx := context.Background()
	store := token.NewMemoryStore()

	tok := &token.Token{
		Value:     "test-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	b.Run("Save", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := store.Save(ctx, "key", tok)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := store.Get(ctx, "key")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

       b.Run("Delete", func(b *testing.B) {
	       for i := 0; i < b.N; i++ {
		       // Save the token before each delete to ensure it exists
		       err := store.Save(ctx, "key", tok)
		       if err != nil {
			       b.Fatal(err)
		       }
		       err = store.Delete(ctx, "key")
		       if err != nil {
			       b.Fatal(err)
		       }
	       }
       })
}

// BenchmarkConcurrentTokenStore benchmarks concurrent token store operations
func BenchmarkConcurrentTokenStore(b *testing.B) {
	ctx := context.Background()
	store := token.NewMemoryStore()

	tok := &token.Token{
		Value:     "test-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}

       b.Run("Concurrent", func(b *testing.B) {
	       var workerID int64
	       b.RunParallel(func(pb *testing.PB) {
		       id := time.Now().UnixNano() + workerID
		       workerID++
		       key := fmt.Sprintf("key-%d", id)
		       for pb.Next() {
			       err := store.Save(ctx, key, tok)
			       if err != nil {
				       b.Fatal(err)
			       }

			       _, err = store.Get(ctx, key)
			       if err != nil {
				       b.Fatal(err)
			       }

			       err = store.Delete(ctx, key)
			       if err != nil {
				       b.Fatal(err)
			       }
		       }
	       })
       })
}
