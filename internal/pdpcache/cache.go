// Package pdpcache provides distributed policy decision point caching (AAP001 sec2.item5).
package pdpcache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached authorization decision.
type CacheEntry struct {
	Key       string                 `json:"key"`
	Decision  string                 `json:"decision"` // "permit", "deny"
	Context   map[string]interface{} `json:"context"`
	CachedAt  time.Time              `json:"cached_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Cache provides distributed PDP decision caching.
type Cache interface {
	// Get retrieves a cached decision.
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set stores a decision in the cache.
	Set(ctx context.Context, key string, decision string, ttl time.Duration, context map[string]interface{}) error

	// Invalidate removes a cached decision.
	Invalidate(ctx context.Context, key string) error

	// InvalidatePattern removes all cached decisions matching a pattern.
	InvalidatePattern(ctx context.Context, pattern string) error

	// Clear removes all cached decisions.
	Clear(ctx context.Context) error

	// Stats returns cache statistics.
	Stats() CacheStats
}

// CacheStats provides cache performance metrics.
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	Size        int       `json:"size"`
	HitRate     float64   `json:"hit_rate"`
	LastClearAt time.Time `json:"last_clear_at,omitempty"`
}

// InMemoryCache provides a simple in-memory cache implementation.
type InMemoryCache struct {
	entries   map[string]*CacheEntry
	mu        sync.RWMutex
	hits      int64
	misses    int64
	evictions int64
	maxSize   int
	ttl       time.Duration
}

// NewInMemoryCache creates a new in-memory cache with default settings.
func NewInMemoryCache(maxSize int, defaultTTL time.Duration) *InMemoryCache {
	cache := &InMemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     defaultTTL,
	}

	// Start background cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached decision.
func (c *InMemoryCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.misses++
		return nil, fmt.Errorf("cache miss: key not found")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.misses++
		return nil, fmt.Errorf("cache miss: entry expired")
	}

	c.hits++
	return entry, nil
}

// Set stores a decision in the cache.
func (c *InMemoryCache) Set(ctx context.Context, key string, decision string, ttl time.Duration, context map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check cache size
	if len(c.entries) >= c.maxSize {
		// Evict oldest entry (simple FIFO)
		c.evictOldest()
	}

	if ttl == 0 {
		ttl = c.ttl
	}

	entry := &CacheEntry{
		Key:       key,
		Decision:  decision,
		Context:   context,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	c.entries[key] = entry
	return nil
}

// Invalidate removes a cached decision.
func (c *InMemoryCache) Invalidate(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		delete(c.entries, key)
		c.evictions++
	}

	return nil
}

// InvalidatePattern removes all entries matching a pattern (simple prefix matching).
func (c *InMemoryCache) InvalidatePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	keysToDelete := []string{}
	for key := range c.entries {
		if matchesPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.entries, key)
		c.evictions++
	}

	return nil
}

// Clear removes all cached decisions.
func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.evictions += int64(len(c.entries))

	return nil
}

// Stats returns cache statistics.
func (c *InMemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		Size:      len(c.entries),
		HitRate:   hitRate,
	}
}

// evictOldest removes the oldest cache entry (FIFO).
func (c *InMemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CachedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.evictions++
	}
}

// cleanup periodically removes expired entries.
func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		keysToDelete := []string{}

		for key, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				keysToDelete = append(keysToDelete, key)
			}
		}

		for _, key := range keysToDelete {
			delete(c.entries, key)
			c.evictions++
		}

		c.mu.Unlock()
	}
}

// matchesPattern checks if a key matches a pattern (simple prefix/suffix/contains matching).
func matchesPattern(key, pattern string) bool {
	// Simple pattern matching: *suffix, prefix*, *contains*
	if len(pattern) == 0 {
		return false
	}

	switch {
	case pattern[0] == '*' && pattern[len(pattern)-1] == '*':
		// Contains
		return contains(key, pattern[1:len(pattern)-1])
	case pattern[0] == '*':
		// Suffix
		return hasSuffix(key, pattern[1:])
	case pattern[len(pattern)-1] == '*':
		// Prefix
		return hasPrefix(key, pattern[:len(pattern)-1])
	default:
		// Exact match
		return key == pattern
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
