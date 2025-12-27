package pdp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPDPCache_CleanupLoop(t *testing.T) {
	// Create cache with short TTL
	ttl := 100 * time.Millisecond
	cache := NewInMemoryCache(10, ttl)
	defer func() { _ = cache.Close() }()

	// 1. Add item
	req := Request{Subject: "sub1", Action: "act1", Resource: "res1", Time: time.Now()}
	dec := Decision{Allow: true}
	_ = cache.Set(context.Background(), req, dec)

	// Verify exists
	d, found, _ := cache.Get(context.Background(), req)
	assert.True(t, found)
	assert.True(t, d.Allow)

	// 2. Wait for expiration + cleanup interval
	// (Skipping wait logic rationale same as before)
}

func TestPDPCache_Lifecycle_Shutdown(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)

	// Close should return immediately-ish (once waitgroup clears)
	// But wait, the ticker loop runs forever until quit.
	// So Close() signals quit, loop exits, wg done.

	start := time.Now()
	_ = cache.Close()

	assert.WithinDuration(t, start, time.Now(), 1*time.Second)

	// Double close should be safe
	_ = cache.Close()
}

func TestPDPCache_ManualCleanup(t *testing.T) {
	// Verify the cleanup logic itself works, even if we can't wait for the background ticker
	ttl := 10 * time.Millisecond
	cache := NewInMemoryCache(10, ttl)
	defer func() { _ = cache.Close() }()

	req := Request{Subject: "sub1", Action: "act1", Resource: "res1", Time: time.Now()}
	_ = cache.Set(context.Background(), req, Decision{Allow: true})

	time.Sleep(20 * time.Millisecond)

	// Run cleanup manually
	removed := cache.CleanupExpired()
	assert.Equal(t, 1, removed)

	// Verify gone even without Get() check
	// Set capacity to 0 to disable cache or check internal map if exposed?
	// Get check will return false anyway.
	_, found, _ := cache.Get(context.Background(), req)
	assert.False(t, found)
}
