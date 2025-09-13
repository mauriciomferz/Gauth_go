package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/rate"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// BenchmarkTokenGeneration benchmarks token generation performance
func BenchmarkTokenGeneration(b *testing.B) {
	ctx := context.Background()
	mgr := token.NewManager(token.Config{
		SigningKey:    []byte("test-key"),
		EncryptionKey: []byte("encryption-key-32-bytes-required!"),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mgr.Generate(ctx, token.Request{
			Subject: "user-123",
			Scope:   []string{"read", "write"},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenValidation benchmarks token validation performance
func BenchmarkTokenValidation(b *testing.B) {
	ctx := context.Background()
	mgr := token.NewManager(token.Config{
		SigningKey:    []byte("test-key"),
		EncryptionKey: []byte("encryption-key-32-bytes-required!"),
	})

	// Generate a token for validation
	tok, err := mgr.Generate(ctx, token.Request{
		Subject: "user-123",
		Scope:   []string{"read", "write"},
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mgr.Validate(ctx, tok.Token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRateLimiting benchmarks rate limiting performance
func BenchmarkRateLimiting(b *testing.B) {
	ctx := context.Background()
	limiter := rate.NewSlidingWindow(100, time.Second, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow(ctx)
	}
}

// BenchmarkDistributedRateLimiting benchmarks distributed rate limiting
func BenchmarkDistributedRateLimiting(b *testing.B) {
	ctx := context.Background()
	limiter := rate.NewDistributedLimiter(rate.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow(ctx)
	}
}

// BenchmarkAuthenticationFlow benchmarks the complete authentication flow
func BenchmarkAuthenticationFlow(b *testing.B) {
	ctx := context.Background()
	authSvc := auth.New([]auth.ProviderConfig{
		{
			Type: "jwt",
			Config: map[string]interface{}{
				"signing_key": "test-key",
			},
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authSvc.Authenticate(ctx, auth.Request{
			Type:       "jwt",
			Principal:  "user-123",
			Credential: "password",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParallelTokenGeneration benchmarks parallel token generation
func BenchmarkParallelTokenGeneration(b *testing.B) {
	ctx := context.Background()
	mgr := token.NewManager(token.Config{
		SigningKey:    []byte("test-key"),
		EncryptionKey: []byte("encryption-key-32-bytes-required!"),
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mgr.Generate(ctx, token.Request{
				Subject: "user-123",
				Scope:   []string{"read", "write"},
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkParallelRateLimiting benchmarks parallel rate limiting
func BenchmarkParallelRateLimiting(b *testing.B) {
	ctx := context.Background()
	limiter := rate.NewSlidingWindow(1000, time.Second, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = limiter.Allow(ctx)
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
