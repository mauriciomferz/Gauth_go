// Copyright (c) 2025 GAuth. All rights reserved.
package pdp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
)

// TestPDPCache_GetSet tests basic cache operations
func TestPDPCache_GetSet(t *testing.T) {
	cache := pdp.NewPDPCache(10, 5*time.Minute)
	ctx := context.Background()

	req := pdp.Request{
		Subject:  "alice",
		Action:   "read",
		Resource: "doc123",
		Time:     time.Now(),
		Attributes: map[string]string{
			"role": "admin",
		},
	}

	decision := pdp.Decision{
		Allow:    true,
		Reason:   "policy:admin-read",
		Metadata: map[string]string{},
	}

	// Cache miss on first Get
	_, found := cache.Get(req)
	if found {
		t.Errorf("Expected cache miss on first Get, got hit")
	}

	// Set decision
	cache.Set(req, decision)

	// Cache hit on second Get
	cachedDec, found := cache.Get(req)
	if !found {
		t.Fatalf("Expected cache hit after Set, got miss")
	}
	if cachedDec.Allow != decision.Allow {
		t.Errorf("Expected Allow=%v, got %v", decision.Allow, cachedDec.Allow)
	}
	if cachedDec.Reason != decision.Reason {
		t.Errorf("Expected Reason=%v, got %v", decision.Reason, cachedDec.Reason)
	}
	if cachedDec.Metadata["cache_hit"] != "true" {
		t.Errorf("Expected cache_hit=true in metadata, got %v", cachedDec.Metadata["cache_hit"])
	}

	// Verify metrics
	metrics := cache.GetMetrics()
	if metrics.Lookups != 2 {
		t.Errorf("Expected 2 lookups, got %d", metrics.Lookups)
	}
	if metrics.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}
	if metrics.Size != 1 {
		t.Errorf("Expected size=1, got %d", metrics.Size)
	}

	_ = ctx // avoid unused warning
}

// TestPDPCache_LRUEviction tests capacity-based eviction
func TestPDPCache_LRUEviction(t *testing.T) {
	cache := pdp.NewPDPCache(3, 1*time.Hour) // Small capacity

	decisions := []struct {
		req pdp.Request
		dec pdp.Decision
	}{
		{
			req: pdp.Request{Subject: "alice", Action: "read", Resource: "doc1", Time: time.Now()},
			dec: pdp.Decision{Allow: true, Reason: "policy1"},
		},
		{
			req: pdp.Request{Subject: "bob", Action: "write", Resource: "doc2", Time: time.Now()},
			dec: pdp.Decision{Allow: false, Reason: "policy2"},
		},
		{
			req: pdp.Request{Subject: "charlie", Action: "delete", Resource: "doc3", Time: time.Now()},
			dec: pdp.Decision{Allow: true, Reason: "policy3"},
		},
	}

	// Fill cache to capacity
	for _, d := range decisions {
		cache.Set(d.req, d.dec)
	}

	// Verify all cached
	for _, d := range decisions {
		_, found := cache.Get(d.req)
		if !found {
			t.Errorf("Expected to find %s in cache", d.req.Subject)
		}
	}

	metrics := cache.GetMetrics()
	if metrics.Size != 3 {
		t.Errorf("Expected size=3, got %d", metrics.Size)
	}
	if metrics.Evictions != 0 {
		t.Errorf("Expected 0 evictions before capacity exceeded, got %d", metrics.Evictions)
	}

	// Add fourth entry (should evict oldest - alice)
	req4 := pdp.Request{Subject: "dave", Action: "read", Resource: "doc4", Time: time.Now()}
	dec4 := pdp.Decision{Allow: true, Reason: "policy4"}
	cache.Set(req4, dec4)

	// Verify alice evicted
	_, found := cache.Get(decisions[0].req)
	if found {
		t.Errorf("Expected alice to be evicted (LRU)")
	}

	// Verify others still cached
	for i := 1; i < len(decisions); i++ {
		_, found := cache.Get(decisions[i].req)
		if !found {
			t.Errorf("Expected to find %s in cache after eviction", decisions[i].req.Subject)
		}
	}

	// Verify dave cached
	_, found = cache.Get(req4)
	if !found {
		t.Errorf("Expected to find dave in cache")
	}

	metrics = cache.GetMetrics()
	if metrics.Size != 3 {
		t.Errorf("Expected size=3 after eviction, got %d", metrics.Size)
	}
	if metrics.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", metrics.Evictions)
	}
}

// TestPDPCache_TTLExpiration tests time-based expiration
func TestPDPCache_TTLExpiration(t *testing.T) {
	cache := pdp.NewPDPCache(10, 100*time.Millisecond) // Short TTL

	req := pdp.Request{
		Subject:  "alice",
		Action:   "read",
		Resource: "doc123",
		Time:     time.Now(),
	}
	decision := pdp.Decision{Allow: true, Reason: "policy1"}

	// Cache decision
	cache.Set(req, decision)

	// Verify cached
	_, found := cache.Get(req)
	if !found {
		t.Fatalf("Expected cache hit immediately after Set")
	}

	// Wait for TTL expiration
	time.Sleep(150 * time.Millisecond)

	// Verify expired (cache miss)
	_, found = cache.Get(req)
	if found {
		t.Errorf("Expected cache miss after TTL expiration")
	}

	metrics := cache.GetMetrics()
	if metrics.Expirations != 1 {
		t.Errorf("Expected 1 expiration, got %d", metrics.Expirations)
	}
	if metrics.Size != 0 {
		t.Errorf("Expected size=0 after expiration, got %d", metrics.Size)
	}
}

// TestPDPCache_InvalidateAll tests full cache invalidation
func TestPDPCache_InvalidateAll(t *testing.T) {
	cache := pdp.NewPDPCache(10, 1*time.Hour)

	requests := []pdp.Request{
		{Subject: "alice", Action: "read", Resource: "doc1", Time: time.Now()},
		{Subject: "bob", Action: "write", Resource: "doc2", Time: time.Now()},
		{Subject: "charlie", Action: "delete", Resource: "doc3", Time: time.Now()},
	}

	// Cache decisions
	for _, req := range requests {
		dec := pdp.Decision{Allow: true, Reason: "policy"}
		cache.Set(req, dec)
	}

	// Verify all cached
	for _, req := range requests {
		_, found := cache.Get(req)
		if !found {
			t.Errorf("Expected to find %s in cache", req.Subject)
		}
	}

	metrics := cache.GetMetrics()
	if metrics.Size != 3 {
		t.Errorf("Expected size=3, got %d", metrics.Size)
	}

	// Invalidate all
	cache.InvalidateAll()

	// Verify all evicted
	for _, req := range requests {
		_, found := cache.Get(req)
		if found {
			t.Errorf("Expected %s to be invalidated", req.Subject)
		}
	}

	metrics = cache.GetMetrics()
	if metrics.Size != 0 {
		t.Errorf("Expected size=0 after InvalidateAll, got %d", metrics.Size)
	}
	if metrics.Invalidations != 3 {
		t.Errorf("Expected 3 invalidations, got %d", metrics.Invalidations)
	}
}

// TestPDPCache_InvalidateSubject tests subject-specific invalidation
func TestPDPCache_InvalidateSubject(t *testing.T) {
	cache := pdp.NewPDPCache(10, 1*time.Hour)

	requests := []pdp.Request{
		{Subject: "alice", Action: "read", Resource: "doc1", Time: time.Now()},
		{Subject: "alice", Action: "write", Resource: "doc2", Time: time.Now()},
		{Subject: "bob", Action: "read", Resource: "doc3", Time: time.Now()},
	}

	// Cache decisions
	for _, req := range requests {
		dec := pdp.Decision{Allow: true, Reason: "policy"}
		cache.Set(req, dec)
	}

	// Invalidate alice
	cache.InvalidateSubject("alice")

	// Verify alice entries invalidated
	for i := 0; i < 2; i++ {
		_, found := cache.Get(requests[i])
		if found {
			t.Errorf("Expected alice request %d to be invalidated", i)
		}
	}

	// Verify bob entry still cached
	_, found := cache.Get(requests[2])
	if !found {
		t.Errorf("Expected bob request to remain cached")
	}

	metrics := cache.GetMetrics()
	if metrics.Size != 1 {
		t.Errorf("Expected size=1 after invalidating alice, got %d", metrics.Size)
	}
	if metrics.Invalidations != 2 {
		t.Errorf("Expected 2 invalidations, got %d", metrics.Invalidations)
	}
}

// TestPDPCache_InvalidateResource tests resource-specific invalidation
func TestPDPCache_InvalidateResource(t *testing.T) {
	cache := pdp.NewPDPCache(10, 1*time.Hour)

	requests := []pdp.Request{
		{Subject: "alice", Action: "read", Resource: "doc123", Time: time.Now()},
		{Subject: "bob", Action: "write", Resource: "doc123", Time: time.Now()},
		{Subject: "charlie", Action: "delete", Resource: "doc456", Time: time.Now()},
	}

	// Cache decisions
	for _, req := range requests {
		dec := pdp.Decision{Allow: true, Reason: "policy"}
		cache.Set(req, dec)
	}

	// Invalidate doc123
	cache.InvalidateResource("doc123")

	// Verify doc123 entries invalidated
	for i := 0; i < 2; i++ {
		_, found := cache.Get(requests[i])
		if found {
			t.Errorf("Expected doc123 request %d to be invalidated", i)
		}
	}

	// Verify doc456 entry still cached
	_, found := cache.Get(requests[2])
	if !found {
		t.Errorf("Expected doc456 request to remain cached")
	}

	metrics := cache.GetMetrics()
	if metrics.Size != 1 {
		t.Errorf("Expected size=1 after invalidating doc123, got %d", metrics.Size)
	}
	if metrics.Invalidations != 2 {
		t.Errorf("Expected 2 invalidations, got %d", metrics.Invalidations)
	}
}

// TestPDPCache_InvalidateAction tests action-specific invalidation
func TestPDPCache_InvalidateAction(t *testing.T) {
	cache := pdp.NewPDPCache(10, 1*time.Hour)

	requests := []pdp.Request{
		{Subject: "alice", Action: "read", Resource: "doc1", Time: time.Now()},
		{Subject: "bob", Action: "read", Resource: "doc2", Time: time.Now()},
		{Subject: "charlie", Action: "write", Resource: "doc3", Time: time.Now()},
	}

	// Cache decisions
	for _, req := range requests {
		dec := pdp.Decision{Allow: true, Reason: "policy"}
		cache.Set(req, dec)
	}

	// Invalidate read action
	cache.InvalidateAction("read")

	// Verify read entries invalidated
	for i := 0; i < 2; i++ {
		_, found := cache.Get(requests[i])
		if found {
			t.Errorf("Expected read request %d to be invalidated", i)
		}
	}

	// Verify write entry still cached
	_, found := cache.Get(requests[2])
	if !found {
		t.Errorf("Expected write request to remain cached")
	}

	metrics := cache.GetMetrics()
	if metrics.Size != 1 {
		t.Errorf("Expected size=1 after invalidating read, got %d", metrics.Size)
	}
	if metrics.Invalidations != 2 {
		t.Errorf("Expected 2 invalidations, got %d", metrics.Invalidations)
	}
}

// TestPDPCache_ThreadSafety tests concurrent cache access
func TestPDPCache_ThreadSafety(t *testing.T) {
	cache := pdp.NewPDPCache(100, 1*time.Hour)

	const numGoroutines = 10
	const numOpsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				req := pdp.Request{
					Subject:  "user",
					Action:   "read",
					Resource: "doc",
					Time:     time.Now(),
					Attributes: map[string]string{
						"goroutine": fmt.Sprintf("%d", id),
						"iteration": fmt.Sprintf("%d", j),
					},
				}
				dec := pdp.Decision{Allow: true, Reason: "policy"}
				cache.Set(req, dec)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is consistent (no panics, no race conditions)
	metrics := cache.GetMetrics()
	if metrics.Size > 100 {
		t.Errorf("Expected size <= 100 (capacity), got %d", metrics.Size)
	}
	if metrics.Size == 0 {
		t.Errorf("Expected size > 0 after concurrent operations")
	}

	t.Logf("Concurrent operations completed: size=%d, evictions=%d", metrics.Size, metrics.Evictions)
}

// TestPDPCache_Metrics tests hit rate calculations
func TestPDPCache_Metrics(t *testing.T) {
	cache := pdp.NewPDPCache(10, 1*time.Hour)

	req := pdp.Request{Subject: "alice", Action: "read", Resource: "doc123", Time: time.Now()}
	dec := pdp.Decision{Allow: true, Reason: "policy"}

	// 1 miss
	cache.Get(req)

	// 1 set
	cache.Set(req, dec)

	// 3 hits
	cache.Get(req)
	cache.Get(req)
	cache.Get(req)

	metrics := cache.GetMetrics()
	if metrics.Lookups != 4 {
		t.Errorf("Expected 4 lookups (1 miss + 3 hits), got %d", metrics.Lookups)
	}
	if metrics.Hits != 3 {
		t.Errorf("Expected 3 hits, got %d", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}

	expectedHitRate := 3.0 / 4.0 // 0.75
	if metrics.HitRate < expectedHitRate-0.01 || metrics.HitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f, got %.4f", expectedHitRate, metrics.HitRate)
	}
}

// TestInMemoryEngine_WithCache tests engine integration
func TestInMemoryEngine_WithCache(t *testing.T) {
	cache := pdp.NewPDPCache(10, 1*time.Hour)

	// Create engine with simple policy
	policy := pdp.Policy{
		ID:       "test-policy",
		Subjects: []string{"alice"},
		Rules: []pdp.Rule{
			{
				ID:        "rule1",
				Actions:   []string{"read"},
				Resources: []string{"*"},
				Effect:    "allow",
			},
		},
	}

	engine := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{})
	engine.AddPolicy(policy)
	engine.WithCache(cache)

	ctx := context.Background()
	req := pdp.Request{
		Subject:  "alice",
		Action:   "read",
		Resource: "doc123",
		Time:     time.Now(),
	}

	// First evaluation (cache miss, policy evaluation)
	dec1, err := engine.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if !dec1.Allow {
		t.Errorf("Expected allow decision")
	}
	if dec1.Metadata["cache_hit"] == "true" {
		t.Errorf("Expected cache_hit=false on first evaluation")
	}

	// Second evaluation (cache hit, no policy evaluation)
	dec2, err := engine.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if !dec2.Allow {
		t.Errorf("Expected allow decision")
	}
	if dec2.Metadata["cache_hit"] != "true" {
		t.Errorf("Expected cache_hit=true on second evaluation, got %v", dec2.Metadata["cache_hit"])
	}

	// Verify cache metrics
	metrics := cache.GetMetrics()
	if metrics.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Expected 1 cache miss (first eval before Set), got %d", metrics.Misses)
	}

	// Invalidate cache and verify miss
	engine.InvalidateCache()
	dec3, err := engine.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if dec3.Metadata["cache_hit"] == "true" {
		t.Errorf("Expected cache_hit=false after InvalidateCache")
	}

	metrics = cache.GetMetrics()
	if metrics.Invalidations != 1 {
		t.Errorf("Expected 1 invalidation, got %d", metrics.Invalidations)
	}
}

// BenchmarkPDPCache_GetHit benchmarks cache hit performance
func BenchmarkPDPCache_GetHit(b *testing.B) {
	cache := pdp.NewPDPCache(1000, 1*time.Hour)

	req := pdp.Request{
		Subject:  "alice",
		Action:   "read",
		Resource: "doc123",
		Time:     time.Now(),
	}
	dec := pdp.Decision{Allow: true, Reason: "policy"}

	// Prime cache
	cache.Set(req, dec)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, found := cache.Get(req)
		if !found {
			b.Fatalf("Expected cache hit")
		}
	}
}

// BenchmarkPDPCache_SetEviction benchmarks cache set with eviction
func BenchmarkPDPCache_SetEviction(b *testing.B) {
	cache := pdp.NewPDPCache(100, 1*time.Hour) // Small capacity to force evictions

	baseTime := time.Now()
	dec := pdp.Decision{Allow: true, Reason: "policy"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := pdp.Request{
			Subject:  "user",
			Action:   "read",
			Resource: "doc",
			Time:     baseTime,
			Attributes: map[string]string{
				"iteration": fmt.Sprintf("%d", i), // Force unique cache key
			},
		}
		cache.Set(req, dec)
	}
}
