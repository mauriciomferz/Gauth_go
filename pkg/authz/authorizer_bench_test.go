package authz

import (
	"context"
	"testing"
	"time"
)

// BenchmarkAuthorize_CacheMiss measures initial authorization calls before cache population.
func BenchmarkAuthorize_CacheMiss(b *testing.B) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p1", Subject: "alice", Resource: "vault", Actions: []string{"read"}, Effect: Allow})
	ma.EnableCaching(500 * time.Millisecond)
	req := Request{Subject: "alice", Resource: "vault", Action: "read"}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Invalidate each loop to force miss path
		ma.InvalidateAll()
		dec, _ := ma.Authorize(ctx, req)
		if !dec.Allow {
			b.Fatalf("expected allow")
		}
	}
}

// BenchmarkAuthorize_CacheHit measures repeated authorizations hitting the in-memory cache.
func BenchmarkAuthorize_CacheHit(b *testing.B) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p1", Subject: "alice", Resource: "vault", Actions: []string{"read"}, Effect: Allow})
	ma.EnableCaching(5 * time.Second) // long TTL to avoid expiry during benchmark
	req := Request{Subject: "alice", Resource: "vault", Action: "read"}
	ctx := context.Background()
	// Prime cache once
	if dec, _ := ma.Authorize(ctx, req); !dec.Allow {
		b.Fatalf("expected initial allow")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, _ := ma.Authorize(ctx, req)
		if !dec.Allow {
			b.Fatalf("expected allow")
		}
	}
}

// BenchmarkAuthorize_Mixed simulates mixed cache hit/miss workload (every 10th request forces an invalidation).
func BenchmarkAuthorize_Mixed(b *testing.B) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p1", Subject: "alice", Resource: "vault", Actions: []string{"read"}, Effect: Allow})
	ma.EnableCaching(2 * time.Second)
	req := Request{Subject: "alice", Resource: "vault", Action: "read"}
	ctx := context.Background()
	// Warm once
	_, _ = ma.Authorize(ctx, req)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%10 == 0 { // force periodic miss
			ma.InvalidateAll()
		}
		dec, _ := ma.Authorize(ctx, req)
		if !dec.Allow {
			b.Fatalf("expected allow")
		}
	}
}
