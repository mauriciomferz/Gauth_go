package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/rate"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// BenchmarkTokenGeneration benchmarks token generation performance
func BenchmarkTokenGeneration(b *testing.B) {
	ctx := context.Background()
	signer := token.NewMockSigner()
	store := token.NewMemoryStore()
	mgr := token.NewService(token.Config{
		SigningKey:     signer,
		Store:          store,
		SigningMethod:  token.RS256,
		ValidityPeriod: time.Hour,
	}, store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := &token.Token{
			Subject: "user-123",
			Scopes:  []string{"read", "write"},
			Issuer:  "test-issuer",
			Type:    token.Access,
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
	signer := token.NewMockSigner()
	store := token.NewMemoryStore()
	mgr := token.NewService(token.Config{
		SigningKey:     signer,
		Store:          store,
		SigningMethod:  token.RS256,
		ValidityPeriod: time.Hour,
	}, store)

	// Generate a token for validation
	t := &token.Token{
		Subject: "user-123",
		Scopes:  []string{"read", "write"},
		Issuer:  "test-issuer",
		Type:    token.Access,
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
	limiter, _ := rate.NewDistributedLimiter(rate.Config{
		Rate:      100,
		Window:    time.Second,
		BurstSize: 10,
	})

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
	signer := token.NewMockSigner()
	store := token.NewMemoryStore()
	mgr := token.NewService(token.Config{
		SigningKey:     signer,
		Store:          store,
		SigningMethod:  token.RS256,
		ValidityPeriod: time.Hour,
	}, store)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t := &token.Token{
				Subject: "user-123",
				Scopes:  []string{"read", "write"},
				Issuer:  "test-issuer",
				Type:    token.Access,
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
			err := store.Delete(ctx, "key")
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
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := store.Save(ctx, "key", tok)
				if err != nil {
					b.Fatal(err)
				}

				_, err = store.Get(ctx, "key")
				if err != nil {
					b.Fatal(err)
				}

				err = store.Delete(ctx, "key")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}
