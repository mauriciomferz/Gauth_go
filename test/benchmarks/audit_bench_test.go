package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
)

func BenchmarkEntry(b *testing.B) {
	b.Run("CreateEntry", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i), audit.ActorUser).
				WithAction(audit.ActionLogin).
				WithTarget("webapp", "application").
				WithResult(audit.ResultSuccess).
				WithMetadata("ip", "192.168.1.1")
		}
	})

	b.Run("HashChain", func(b *testing.B) {
		entry := audit.NewEntry(audit.TypeAuth)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			entry.CalculateHash()
		}
	})
}

func BenchmarkFileStorage(b *testing.B) {
	dir := b.TempDir()
	storage, err := audit.NewFileStorage(audit.FileConfig{
		Directory: dir,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer storage.Close()

	ctx := context.Background()

	b.Run("Store", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			entry := audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i), audit.ActorUser).
				WithAction(audit.ActionLogin)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Retrieve", func(b *testing.B) {
		// Create test entries first
		ids := make([]string, 1000)
		for i := range ids {
			entry := audit.NewEntry(audit.TypeAuth)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
			ids[i] = entry.ID
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := ids[i%len(ids)]
			if _, err := storage.GetByID(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Search", func(b *testing.B) {
		// Create test entries first
		for i := 0; i < 1000; i++ {
			entry := audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i%10), audit.ActorUser)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter := &audit.Filter{
				ActorIDs: []string{fmt.Sprintf("user%d", i%10)},
			}
			if _, err := storage.Search(ctx, filter); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRedisStorage(b *testing.B) {
	storage, err := audit.NewRedisStorage(audit.RedisConfig{
		Addresses: []string{"localhost:6379"},
		KeyPrefix: "benchmark:",
	})
	if err != nil {
		b.Skip("Redis not available:", err)
	}
	defer storage.Close()

	ctx := context.Background()

	b.Run("Store", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			entry := audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i), audit.ActorUser).
				WithAction(audit.ActionLogin)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Retrieve", func(b *testing.B) {
		// Create test entries first
		ids := make([]string, 1000)
		for i := range ids {
			entry := audit.NewEntry(audit.TypeAuth)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
			ids[i] = entry.ID
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := ids[i%len(ids)]
			if _, err := storage.GetByID(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Search", func(b *testing.B) {
		// Create test entries first
		for i := 0; i < 1000; i++ {
			entry := audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i%10), audit.ActorUser)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter := &audit.Filter{
				ActorIDs: []string{fmt.Sprintf("user%d", i%10)},
			}
			if _, err := storage.Search(ctx, filter); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSQLStorage(b *testing.B) {
	storage, err := audit.NewSQLStorage(audit.SQLConfig{
		Driver: "postgres",
		DSN:    "postgres://localhost/test?sslmode=disable",
	})
	if err != nil {
		b.Skip("PostgreSQL not available:", err)
	}
	defer storage.Close()

	ctx := context.Background()

	b.Run("Store", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			entry := audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i), audit.ActorUser).
				WithAction(audit.ActionLogin)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Retrieve", func(b *testing.B) {
		// Create test entries first
		ids := make([]string, 1000)
		for i := range ids {
			entry := audit.NewEntry(audit.TypeAuth)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
			ids[i] = entry.ID
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := ids[i%len(ids)]
			if _, err := storage.GetByID(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Search", func(b *testing.B) {
		// Create test entries first
		for i := 0; i < 1000; i++ {
			entry := audit.NewEntry(audit.TypeAuth).
				WithActor(fmt.Sprintf("user%d", i%10), audit.ActorUser)
			if err := storage.Store(ctx, entry); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter := &audit.Filter{
				ActorIDs: []string{fmt.Sprintf("user%d", i%10)},
			}
			if _, err := storage.Search(ctx, filter); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkMetrics(b *testing.B) {
	metrics := audit.NewMetrics("benchmark")

	b.Run("ObserveEntry", func(b *testing.B) {
		entry := audit.NewEntry(audit.TypeAuth).
			WithAction(audit.ActionLogin).
			WithResult(audit.ResultSuccess)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			metrics.ObserveEntry(entry, time.Millisecond)
		}
	})

	b.Run("StorageOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			metrics.ObserveStorageOperation("store", "redis", time.Millisecond)
		}
	})
}
