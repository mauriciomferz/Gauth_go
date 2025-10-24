package authz

import (
	"container/list"
	"strconv"
	"sync"
	"time"
)

// AuthorizationCacheEntry stores a decision plus versioning metadata to detect staleness.
type AuthorizationCacheEntry struct {
	Decision      Decision
	PolicyVersion int64
	Jurisdiction  string
	// insertion timestamp (optional future TTL logic or aging metrics)
	Inserted time.Time
}

// AuthorizationCache provides an LRU cache keyed by composite authorization attributes.
// Key shape (pipe-delimited): subject|action|resource|policyVersion|jurisdiction
// It supports staleness detection when policyVersion / jurisdiction differ from current evaluation context.
// Metrics exposed via Snapshot for hit ratio & stale evictions.
type AuthorizationCache struct {
	capacity int
	mu       sync.Mutex
	items    map[string]*list.Element
	order    *list.List // front = most recent
	// metrics counters
	lookups        uint64
	hits           uint64
	misses         uint64
	staleEvictions uint64
	// explicit invalidation counters
	invalidations uint64
}

// cacheListPayload wraps key and value for list element.
type cacheListPayload struct {
	key   string
	value AuthorizationCacheEntry
}

// NewAuthorizationCache constructs an LRU cache with the provided positive capacity.
// capacity <= 0 results in a no-op cache (all operations degrade to misses).
func NewAuthorizationCache(capacity int) *AuthorizationCache {
	if capacity <= 0 {
		capacity = 0
	}
	return &AuthorizationCache{capacity: capacity, items: make(map[string]*list.Element, capacity), order: list.New()}
}

// makeKey builds the composite cache key.
func makeKey(subject, action, resource string, policyVersion int64, jurisdiction string) string {
	return subject + "|" + action + "|" + resource + "|" + fmtInt(policyVersion) + "|" + jurisdiction
}

// fmtInt local lightweight int64 formatter without strconv allocations in hot path.
func fmtInt(v int64) string {
	// fast path small numbers
	if v >= 0 && v < 10 {
		return string(rune('0' + v))
	}
	// fallback to standard formatting (still efficient)
	return strconv.FormatInt(v, 10)
}

// Get returns a decision and whether it was found (not yet judged stale). Version & jurisdiction are validated externally.
func (c *AuthorizationCache) Get(key string) (AuthorizationCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lookups++
	if c.capacity == 0 {
		c.misses++
		return AuthorizationCacheEntry{}, false
	}
	if el, ok := c.items[key]; ok {
		// Move to front (MRU)
		c.order.MoveToFront(el)
		c.hits++
		payload := el.Value.(*cacheListPayload)
		return payload.value, true
	}
	c.misses++
	return AuthorizationCacheEntry{}, false
}

// Set inserts or updates a decision entry.
func (c *AuthorizationCache) Set(key string, entry AuthorizationCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.capacity == 0 {
		return
	}
	if el, ok := c.items[key]; ok {
		payload := el.Value.(*cacheListPayload)
		payload.value = entry
		c.order.MoveToFront(el)
		return
	}
	// Evict if over capacity
	if c.order.Len() >= c.capacity {
		last := c.order.Back()
		if last != nil {
			payload := last.Value.(*cacheListPayload)
			delete(c.items, payload.key)
			c.order.Remove(last)
		}
	}
	payload := &cacheListPayload{key: key, value: entry}
	el := c.order.PushFront(payload)
	c.items[key] = el
}

// Invalidate removes a specific key.
func (c *AuthorizationCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		delete(c.items, key)
		c.order.Remove(el)
		c.invalidations++
	}
}

// InvalidateAll clears the cache.
func (c *AuthorizationCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element, c.capacity)
	c.order.Init()
	c.invalidations++
}

// MarkStale evicts a key counting it as stale eviction (used when version/jurisdiction mismatch detected on access).
func (c *AuthorizationCache) MarkStale(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		delete(c.items, key)
		c.order.Remove(el)
		c.staleEvictions++
	}
}

// Size returns current number of entries.
func (c *AuthorizationCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// Snapshot returns current metrics (hit ratio derived).
func (c *AuthorizationCache) Snapshot() AuthorizationCacheMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ratio float64
	if c.lookups > 0 {
		ratio = float64(c.hits) / float64(c.lookups)
	}
	return AuthorizationCacheMetrics{
		Capacity:       c.capacity,
		Size:           c.order.Len(),
		Lookups:        c.lookups,
		Hits:           c.hits,
		Misses:         c.misses,
		HitRatio:       ratio,
		StaleEvictions: c.staleEvictions,
		Invalidations:  c.invalidations,
	}
}

// AuthorizationCacheMetrics describes snapshot counters.
type AuthorizationCacheMetrics struct {
	Capacity       int     `json:"capacity"`
	Size           int     `json:"size"`
	Lookups        uint64  `json:"lookups"`
	Hits           uint64  `json:"hits"`
	Misses         uint64  `json:"misses"`
	HitRatio       float64 `json:"hit_ratio"`
	StaleEvictions uint64  `json:"stale_evictions"`
	Invalidations  uint64  `json:"invalidations"`
}
