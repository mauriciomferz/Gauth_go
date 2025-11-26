package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache implements the Cache interface using in-memory storage
// Used as a fallback when Redis is not available
type MemoryCache struct {
	items   map[string]*cacheItem
	mu      sync.RWMutex
	config  *Config
	stats   *Stats
	statsMu sync.RWMutex
}

type cacheItem struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache(config *Config) *MemoryCache {
	cache := &MemoryCache{
		items:  make(map[string]*cacheItem),
		config: config,
		stats: &Stats{
			Hits:   0,
			Misses: 0,
		},
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a value from memory cache
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	item, exists := m.items[key]
	m.mu.RUnlock()

	if !exists {
		m.incrementMisses()
		return nil, nil
	}

	// Check expiration
	if time.Now().After(item.expiration) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		m.incrementMisses()
		return nil, nil
	}

	m.incrementHits()
	return item.value, nil
}

// Set stores a value in memory cache with TTL
func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	expiration := time.Now().Add(ttl)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max size
	if len(m.items) >= m.config.MaxSize {
		// Simple eviction: remove oldest expired item or first item
		m.evictOne()
	}

	m.items[key] = &cacheItem{
		value:      value,
		expiration: expiration,
	}

	return nil
}

// Delete removes a value from memory cache
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}

// DeletePattern removes all keys matching the pattern (simple prefix match)
func (m *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple pattern matching (supports * at the end)
	prefix := pattern
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix = pattern[:len(pattern)-1]
	}

	for key := range m.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(m.items, key)
		}
	}

	return nil
}

// Exists checks if a key exists in memory cache
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	item, exists := m.items[key]
	m.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Check expiration
	if time.Now().After(item.expiration) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// GetStats returns memory cache statistics
func (m *MemoryCache) GetStats(ctx context.Context) (*Stats, error) {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	m.mu.RLock()
	keys := int64(len(m.items))
	m.mu.RUnlock()

	stats := &Stats{
		Hits:   m.stats.Hits,
		Misses: m.stats.Misses,
		Keys:   keys,
	}

	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
	}

	return stats, nil
}

// Close closes the memory cache (no-op)
func (m *MemoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]*cacheItem)
	return nil
}

// Ping checks if the cache is operational (always true for memory cache)
func (m *MemoryCache) Ping(ctx context.Context) error {
	return nil
}

// Helper methods

func (m *MemoryCache) incrementHits() {
	m.statsMu.Lock()
	m.stats.Hits++
	m.statsMu.Unlock()
}

func (m *MemoryCache) incrementMisses() {
	m.statsMu.Lock()
	m.stats.Misses++
	m.statsMu.Unlock()
}

func (m *MemoryCache) evictOne() {
	// Remove first expired item found
	now := time.Now()
	for key, item := range m.items {
		if now.After(item.expiration) {
			delete(m.items, key)
			return
		}
	}

	// If no expired items, remove first item (simple FIFO)
	for key := range m.items {
		delete(m.items, key)
		return
	}
}

func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, item := range m.items {
			if now.After(item.expiration) {
				delete(m.items, key)
			}
		}
		m.mu.Unlock()
	}
}
