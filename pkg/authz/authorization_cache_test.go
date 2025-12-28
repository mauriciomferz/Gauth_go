package authz

import (
	"testing"
	"time"
)

// TestNewLRUDecisionCache verifies constructor with various capacity values
func TestNewLRUDecisionCache(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		wantCap  int
	}{
		{"positive capacity", 100, 100},
		{"zero capacity", 0, 0},
		{"negative capacity", -10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLRUDecisionCache(tt.capacity)
			if cache == nil {
				t.Fatal("NewLRUDecisionCache returned nil")
			}
			if cache.capacity != tt.wantCap {
				t.Errorf("capacity = %d, want %d", cache.capacity, tt.wantCap)
			}
			if cache.items == nil {
				t.Error("items map not initialized")
			}
			if cache.order == nil {
				t.Error("order list not initialized")
			}
		})
	}
}

// TestMakeKey verifies cache key generation
func TestMakeKey(t *testing.T) {
	tests := []struct {
		name          string
		subject       string
		action        string
		resource      string
		policyVersion int64
		jurisdiction  string
		want          string
	}{
		{
			name:          "basic key",
			subject:       "user:123",
			action:        "read",
			resource:      "document:456",
			policyVersion: 1,
			jurisdiction:  "us-west",
			want:          "user:123|read|document:456|1|us-west",
		},
		{
			name:          "empty jurisdiction",
			subject:       "admin",
			action:        "write",
			resource:      "file",
			policyVersion: 5,
			jurisdiction:  "",
			want:          "admin|write|file|5|",
		},
		{
			name:          "large version number",
			subject:       "svc",
			action:        "delete",
			resource:      "obj",
			policyVersion: 999999999,
			jurisdiction:  "global",
			want:          "svc|delete|obj|999999999|global",
		},
		{
			name:          "version 0",
			subject:       "guest",
			action:        "view",
			resource:      "page",
			policyVersion: 0,
			jurisdiction:  "local",
			want:          "guest|view|page|0|local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeKey(tt.subject, tt.action, tt.resource, tt.policyVersion, tt.jurisdiction)
			if got != tt.want {
				t.Errorf("makeKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFmtInt verifies int64 formatting helper
func TestFmtInt(t *testing.T) {
	tests := []struct {
		name string
		v    int64
		want string
	}{
		{"zero", 0, "0"},
		{"single digit", 5, "5"},
		{"nine", 9, "9"},
		{"ten", 10, "10"},
		{"negative", -5, "-5"},
		{"large positive", 123456789, "123456789"},
		{"large negative", -987654321, "-987654321"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmtInt(tt.v)
			if got != tt.want {
				t.Errorf("fmtInt(%d) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

// TestLRUDecisionCache_GetSet verifies basic get/set operations
func TestLRUDecisionCache_GetSet(t *testing.T) {
	cache := NewLRUDecisionCache(10)
	key := makeKey("user1", "read", "doc1", 1, "us")

	// Test miss on empty cache
	entry, found := cache.Get(key)
	if found {
		t.Error("Get() on empty cache should return false")
	}
	if entry.Decision.Allow {
		t.Error("Get() miss should return zero-value entry")
	}

	// Test set and get
	expected := AuthorizationCacheEntry{
		Decision: Decision{
			Allow:  true,
			Reason: "policy allows",
		},
		PolicyVersion: 1,
		Jurisdiction:  "us",
		Inserted:      time.Now(),
	}
	cache.Set(key, expected)

	entry, found = cache.Get(key)
	if !found {
		t.Fatal("Get() should find entry after Set()")
	}
	if entry.Decision.Allow != expected.Decision.Allow {
		t.Errorf("Decision.Allow = %v, want %v", entry.Decision.Allow, expected.Decision.Allow)
	}
	if entry.Decision.Reason != expected.Decision.Reason {
		t.Errorf("Decision.Reason = %q, want %q", entry.Decision.Reason, expected.Decision.Reason)
	}
	if entry.PolicyVersion != expected.PolicyVersion {
		t.Errorf("PolicyVersion = %d, want %d", entry.PolicyVersion, expected.PolicyVersion)
	}
	if entry.Jurisdiction != expected.Jurisdiction {
		t.Errorf("Jurisdiction = %q, want %q", entry.Jurisdiction, expected.Jurisdiction)
	}
}

// TestLRUDecisionCache_Update verifies updating existing entries
func TestLRUDecisionCache_Update(t *testing.T) {
	cache := NewLRUDecisionCache(5)
	key := makeKey("user1", "read", "doc1", 1, "us")

	// Initial entry
	cache.Set(key, AuthorizationCacheEntry{
		Decision:      Decision{Allow: true, Reason: "initial"},
		PolicyVersion: 1,
		Jurisdiction:  "us",
	})

	// Update entry
	cache.Set(key, AuthorizationCacheEntry{
		Decision:      Decision{Allow: false, Reason: "updated"},
		PolicyVersion: 2,
		Jurisdiction:  "us",
	})

	entry, found := cache.Get(key)
	if !found {
		t.Fatal("Get() should find updated entry")
	}
	if entry.Decision.Allow {
		t.Error("Decision should be updated to false")
	}
	if entry.Decision.Reason != "updated" {
		t.Errorf("Reason = %q, want %q", entry.Decision.Reason, "updated")
	}
	if entry.PolicyVersion != 2 {
		t.Errorf("PolicyVersion = %d, want 2", entry.PolicyVersion)
	}
}

// TestLRUDecisionCache_Eviction verifies LRU eviction when capacity exceeded
func TestLRUDecisionCache_Eviction(t *testing.T) {
	cache := NewLRUDecisionCache(3) // Small capacity for testing

	// Fill cache to capacity
	for i := 0; i < 3; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Set(key, AuthorizationCacheEntry{
			Decision:      Decision{Allow: true},
			PolicyVersion: int64(i),
		})
	}

	// Verify all 3 entries are present
	for i := 0; i < 3; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		if _, found := cache.Get(key); !found {
			t.Errorf("Entry %d should be in cache", i)
		}
	}

	// Add 4th entry (should evict oldest, which is entry 0)
	key4 := makeKey("user", "read", "doc", 3, "us")
	cache.Set(key4, AuthorizationCacheEntry{
		Decision:      Decision{Allow: true},
		PolicyVersion: 3,
	})

	// Entry 0 should be evicted (LRU)
	key0 := makeKey("user", "read", "doc", 0, "us")
	if _, found := cache.Get(key0); found {
		t.Error("Oldest entry should have been evicted")
	}

	// Entries 1, 2, 3 should still be present
	for i := 1; i <= 3; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		if _, found := cache.Get(key); !found {
			t.Errorf("Entry %d should still be in cache", i)
		}
	}
}

// TestLRUDecisionCache_LRU verifies least-recently-used ordering
func TestLRUDecisionCache_LRU(t *testing.T) {
	cache := NewLRUDecisionCache(3)

	// Add 3 entries
	key1 := makeKey("user", "read", "doc1", 1, "us")
	key2 := makeKey("user", "read", "doc2", 1, "us")
	key3 := makeKey("user", "read", "doc3", 1, "us")

	cache.Set(key1, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})
	cache.Set(key2, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})
	cache.Set(key3, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})

	// Access key1 (moves it to front)
	cache.Get(key1)

	// Add 4th entry (should evict key2, the LRU)
	key4 := makeKey("user", "read", "doc4", 1, "us")
	cache.Set(key4, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})

	// key2 should be evicted (oldest access)
	if _, found := cache.Get(key2); found {
		t.Error("key2 should have been evicted (LRU)")
	}

	// key1, key3, key4 should remain
	if _, found := cache.Get(key1); !found {
		t.Error("key1 should still be in cache")
	}
	if _, found := cache.Get(key3); !found {
		t.Error("key3 should still be in cache")
	}
	if _, found := cache.Get(key4); !found {
		t.Error("key4 should still be in cache")
	}
}

// TestLRUDecisionCache_Invalidate verifies single key invalidation
func TestLRUDecisionCache_Invalidate(t *testing.T) {
	cache := NewLRUDecisionCache(10)

	key1 := makeKey("user1", "read", "doc1", 1, "us")
	key2 := makeKey("user2", "read", "doc2", 1, "us")

	cache.Set(key1, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})
	cache.Set(key2, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})

	// Verify both present
	if _, found := cache.Get(key1); !found {
		t.Error("key1 should be in cache before invalidation")
	}
	if _, found := cache.Get(key2); !found {
		t.Error("key2 should be in cache before invalidation")
	}

	// Invalidate key1
	cache.Invalidate(key1)

	// key1 should be gone
	if _, found := cache.Get(key1); found {
		t.Error("key1 should be invalidated")
	}

	// key2 should still be present
	if _, found := cache.Get(key2); !found {
		t.Error("key2 should still be in cache")
	}

	// Verify invalidation counter
	metrics := cache.Snapshot()
	if metrics.Invalidations != 1 {
		t.Errorf("Invalidations = %d, want 1", metrics.Invalidations)
	}
}

// TestLRUDecisionCache_InvalidateNonExistent verifies invalidating non-existent key is safe
func TestLRUDecisionCache_InvalidateNonExistent(t *testing.T) {
	cache := NewLRUDecisionCache(10)
	key := makeKey("user", "read", "doc", 1, "us")

	// Invalidate non-existent key (should not panic)
	cache.Invalidate(key)

	metrics := cache.Snapshot()
	if metrics.Invalidations != 0 {
		t.Errorf("Invalidations = %d, want 0 for non-existent key", metrics.Invalidations)
	}
}

// TestLRUDecisionCache_InvalidateAll verifies clearing entire cache
func TestLRUDecisionCache_InvalidateAll(t *testing.T) {
	cache := NewLRUDecisionCache(10)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: int64(i)})
	}

	// Verify entries present
	sizeBefore := cache.Size()
	if sizeBefore != 5 {
		t.Errorf("Size before = %d, want 5", sizeBefore)
	}

	// Clear all
	cache.InvalidateAll()

	// Verify all gone
	sizeAfter := cache.Size()
	if sizeAfter != 0 {
		t.Errorf("Size after InvalidateAll = %d, want 0", sizeAfter)
	}

	// Verify entries not found
	for i := 0; i < 5; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		if _, found := cache.Get(key); found {
			t.Errorf("Entry %d should be cleared", i)
		}
	}

	// Verify invalidation counter incremented
	metrics := cache.Snapshot()
	if metrics.Invalidations != 1 {
		t.Errorf("Invalidations = %d, want 1", metrics.Invalidations)
	}
}

// TestLRUDecisionCache_MarkStale verifies stale eviction tracking
func TestLRUDecisionCache_MarkStale(t *testing.T) {
	cache := NewLRUDecisionCache(10)
	key := makeKey("user", "read", "doc", 1, "us")

	cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})

	// Verify entry present
	if _, found := cache.Get(key); !found {
		t.Error("Entry should be in cache before MarkStale")
	}

	// Mark as stale
	cache.MarkStale(key)

	// Entry should be removed
	if _, found := cache.Get(key); found {
		t.Error("Entry should be removed after MarkStale")
	}

	// Verify stale eviction counter
	metrics := cache.Snapshot()
	if metrics.StaleEvictions != 1 {
		t.Errorf("StaleEvictions = %d, want 1", metrics.StaleEvictions)
	}
}

// TestLRUDecisionCache_Size verifies size tracking
func TestLRUDecisionCache_Size(t *testing.T) {
	cache := NewLRUDecisionCache(10)

	if size := cache.Size(); size != 0 {
		t.Errorf("Initial size = %d, want 0", size)
	}

	// Add entries
	for i := 0; i < 5; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: int64(i)})

		expectedSize := i + 1
		if size := cache.Size(); size != expectedSize {
			t.Errorf("Size after %d inserts = %d, want %d", i+1, size, expectedSize)
		}
	}

	// Remove entry
	key := makeKey("user", "read", "doc", 2, "us")
	cache.Invalidate(key)

	if size := cache.Size(); size != 4 {
		t.Errorf("Size after invalidation = %d, want 4", size)
	}
}

// TestLRUDecisionCache_Snapshot verifies metrics snapshot
func TestLRUDecisionCache_Snapshot(t *testing.T) {
	cache := NewLRUDecisionCache(10)

	// Initial snapshot
	metrics := cache.Snapshot()
	if metrics.Capacity != 10 {
		t.Errorf("Capacity = %d, want 10", metrics.Capacity)
	}
	if metrics.Size != 0 {
		t.Errorf("Size = %d, want 0", metrics.Size)
	}
	if metrics.Lookups != 0 {
		t.Errorf("Lookups = %d, want 0", metrics.Lookups)
	}
	if metrics.HitRatio != 0 {
		t.Errorf("HitRatio = %f, want 0", metrics.HitRatio)
	}

	// Add entry
	key := makeKey("user", "read", "doc", 1, "us")
	cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})

	// Hit
	cache.Get(key)
	// Miss
	cache.Get("nonexistent")

	metrics = cache.Snapshot()
	if metrics.Size != 1 {
		t.Errorf("Size = %d, want 1", metrics.Size)
	}
	if metrics.Lookups != 2 {
		t.Errorf("Lookups = %d, want 2", metrics.Lookups)
	}
	if metrics.Hits != 1 {
		t.Errorf("Hits = %d, want 1", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Misses = %d, want 1", metrics.Misses)
	}

	expectedRatio := 0.5 // 1 hit / 2 lookups
	if metrics.HitRatio != expectedRatio {
		t.Errorf("HitRatio = %f, want %f", metrics.HitRatio, expectedRatio)
	}
}

// TestLRUDecisionCache_HitRatioCalculation verifies hit ratio calculation
func TestLRUDecisionCache_HitRatioCalculation(t *testing.T) {
	cache := NewLRUDecisionCache(10)

	// Add entries
	for i := 0; i < 5; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: int64(i)})
	}

	// Generate hits and misses
	// 5 hits
	for i := 0; i < 5; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Get(key)
	}
	// 5 misses
	for i := 10; i < 15; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Get(key)
	}

	metrics := cache.Snapshot()
	if metrics.Hits != 5 {
		t.Errorf("Hits = %d, want 5", metrics.Hits)
	}
	if metrics.Misses != 5 {
		t.Errorf("Misses = %d, want 5", metrics.Misses)
	}
	if metrics.Lookups != 10 {
		t.Errorf("Lookups = %d, want 10", metrics.Lookups)
	}

	expectedRatio := 0.5 // 5 hits / 10 lookups
	if metrics.HitRatio != expectedRatio {
		t.Errorf("HitRatio = %f, want %f", metrics.HitRatio, expectedRatio)
	}
}

// TestLRUDecisionCache_ZeroCapacity verifies zero-capacity cache behavior
func TestLRUDecisionCache_ZeroCapacity(t *testing.T) {
	cache := NewLRUDecisionCache(0)

	key := makeKey("user", "read", "doc", 1, "us")

	// Set should be no-op
	cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})

	// Get should always miss
	if _, found := cache.Get(key); found {
		t.Error("Zero-capacity cache should always return miss")
	}

	// Size should be 0
	if size := cache.Size(); size != 0 {
		t.Errorf("Size = %d, want 0", size)
	}

	metrics := cache.Snapshot()
	if metrics.Lookups != 1 {
		t.Errorf("Lookups = %d, want 1", metrics.Lookups)
	}
	if metrics.Misses != 1 {
		t.Errorf("Misses = %d, want 1", metrics.Misses)
	}
	if metrics.Hits != 0 {
		t.Errorf("Hits = %d, want 0", metrics.Hits)
	}
}

// TestLRUDecisionCache_Concurrent verifies thread-safety (smoke test)
func TestLRUDecisionCache_Concurrent(t *testing.T) {
	cache := NewLRUDecisionCache(100)
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := makeKey("user", "read", "doc", int64(i%10), "us")
			cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: int64(i)})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := makeKey("user", "read", "doc", int64(i%10), "us")
			cache.Get(key)
		}
		done <- true
	}()

	// Wait for completion (no panics = success)
	<-done
	<-done
}
