package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_Lifecycle(t *testing.T) {
	config := &Config{
		MaxSize: 100,
	}

	cache := NewMemoryCache(config)

	// Create context
	ctx := context.Background()

	// Add item
	err := cache.Set(ctx, "key", []byte("value"), time.Minute)
	require.NoError(t, err)

	exists, err := cache.Exists(ctx, "key")
	require.NoError(t, err)
	assert.True(t, exists)

	// Close
	start := time.Now()
	err = cache.Close()
	require.NoError(t, err)

	// Should close quickly (waiting for goroutine)
	assert.WithinDuration(t, start, time.Now(), 1*time.Second)

	// Verify cleared
	exists, err = cache.Exists(ctx, "key")
	require.NoError(t, err)
	assert.False(t, exists, "Items should be cleared after Close")
}

func TestMemoryCache_Expiration(t *testing.T) {
	// Note: We can't easily test the background loop timing without exposing internals
	// or waiting a full minute (ticker duration).
	// However, we can verified Get/Exists checks expiration lazily too.
	// But the background loop is what we fixed for leaks.
	// The Lifecycle test above confirms the background loop listens to quit.

	config := &Config{MaxSize: 100}
	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()
	// Set item with very short TTL
	err := cache.Set(ctx, "short", []byte("val"), 10*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)

	exists, err := cache.Exists(ctx, "short")
	require.NoError(t, err)
	assert.False(t, exists, "Item should assume expired on access check")
}
