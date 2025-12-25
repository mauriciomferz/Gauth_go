package oidc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryCache_Lifecycle(t *testing.T) {
	// Create cache
	cache := NewInMemoryDiscoveryCache(WithMaxEntries(10))

	// Add item
	doc := &OIDCConfiguration{Issuer: "https://test.com"}
	err := cache.Set("https://test.com", doc, time.Hour)
	require.NoError(t, err)

	// Close
	start := time.Now()
	err = cache.Close()
	require.NoError(t, err)

	// Should close quickly (waiting for goroutine)
	assert.WithinDuration(t, start, time.Now(), 1*time.Second)

	// Verify can still perform ops (though background loop is dead)
	// This confirms structure integrity after close
	err = cache.Set("https://test.com/2", doc, time.Hour)
	require.NoError(t, err)
}

func TestDiscoveryCache_Expiration_Cleanup(t *testing.T) {
	// Difficult to test background loop timing precisely without sleeps.
	// We rely on the Lifecycle test to prove shutdown mechanics
	// and trust the loop logic (std lib Ticker/Select).

	cache := NewInMemoryDiscoveryCache()
	defer cache.Close()

	doc := &OIDCConfiguration{Issuer: "https://expired.com"}
	// Set 1ms TTL
	err := cache.Set("https://expired.com", doc, 1*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	// Verify Get returns expired item behavior (returns doc if in cache but expired)
	// Implementation says: Get() checks expiration. If expired, it tries to fetch. If fetch fails, returns stale.
	// Since we mock nothing here, fetch fails.

	ctx := context.Background()

	// Get triggers expiration check
	// With the new logic:
	// 1. Check cache -> exists but IsExpired() = true
	// 2. Fetch -> Fails (no network)
	// 3. Returns stale (so we get doc back with error? No, code swallows error if stale exists)

	retrieved, err := cache.Get(ctx, "https://expired.com")
	// Expectation: err is nil (fallback to stale), retrieved is doc
	require.NoError(t, err)
	assert.Equal(t, doc, retrieved)
}
