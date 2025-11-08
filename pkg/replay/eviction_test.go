package replay

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// TestTTLEviction verifies TTL-based eviction removes expired entries.
func TestTTLEviction(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "replay_ttl.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              100 * time.Millisecond,
		SnapshotInterval: 1 * time.Hour, // Disable automatic snapshots
		EvictionPolicy:   &TTLEvictionPolicy{TTL: 100 * time.Millisecond},
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	// Record 3 tokens
	if err := store.Record("token1", time.Now()); err != nil {
		t.Fatalf("Record token1 failed: %v", err)
	}
	if err := store.Record("token2", time.Now()); err != nil {
		t.Fatalf("Record token2 failed: %v", err)
	}
	if err := store.Record("token3", time.Now()); err != nil {
		t.Fatalf("Record token3 failed: %v", err)
	}

	// All should be seen immediately
	if seen, _ := store.Seen("token1"); !seen {
		t.Error("token1 should be seen")
	}
	if seen, _ := store.Seen("token2"); !seen {
		t.Error("token2 should be seen")
	}
	if seen, _ := store.Seen("token3"); !seen {
		t.Error("token3 should be seen")
	}

	// Wait for TTL expiration
	time.Sleep(150 * time.Millisecond)

	// All should be evicted now
	if seen, _ := store.Seen("token1"); seen {
		t.Error("token1 should be evicted after TTL")
	}
	if seen, _ := store.Seen("token2"); seen {
		t.Error("token2 should be evicted after TTL")
	}
	if seen, _ := store.Seen("token3"); seen {
		t.Error("token3 should be evicted after TTL")
	}

	// Verify eviction stats
	if store.evictionStats.TTLEvictions == 0 {
		t.Error("TTLEvictions should be > 0")
	}
}

// TestSizeBasedEviction verifies size-based eviction removes oldest entries.
func TestSizeBasedEviction(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "replay_size.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              10 * time.Minute,
		SnapshotInterval: 1 * time.Hour,
		EvictionPolicy:   &SizeBasedEvictionPolicy{MaxSize: 3},
		MaxSize:          3,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	// Record 5 tokens (should evict oldest 2)
	now := time.Now()
	store.Record("token1", now.Add(-5*time.Second)) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	store.Record("token2", now.Add(-4*time.Second)) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	store.Record("token3", now.Add(-3*time.Second)) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	store.Record("token4", now.Add(-2*time.Second)) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	store.Record("token5", now.Add(-1*time.Second)) //nolint:errcheck

	// Trigger eviction by checking one
	_, _ = store.Seen("token5") //nolint:errcheck

	// Oldest 2 should be evicted (token1, token2)
	if seen, _ := store.Seen("token1"); seen {
		t.Error("token1 (oldest) should be evicted")
	}
	if seen, _ := store.Seen("token2"); seen {
		t.Error("token2 (2nd oldest) should be evicted")
	}

	// Newest 3 should remain
	if seen, _ := store.Seen("token3"); !seen {
		t.Error("token3 should remain")
	}
	if seen, _ := store.Seen("token4"); !seen {
		t.Error("token4 should remain")
	}
	if seen, _ := store.Seen("token5"); !seen {
		t.Error("token5 should remain")
	}

	// Verify eviction stats
	if store.evictionStats.SizeEvictions == 0 {
		t.Error("SizeEvictions should be > 0")
	}
}

// TestLRUEviction verifies LRU eviction basics (size limit enforcement).
// Note: Full LRU semantics (access time ordering) tested via size-based fallback.
func TestLRUEviction(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "replay_lru.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              10 * time.Minute,
		SnapshotInterval: 1 * time.Hour,
		EvictionPolicy:   &LRUEvictionPolicy{MaxSize: 3},
		MaxSize:          3,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	now := time.Now()
	// Record 5 tokens (should trigger eviction)
	for i := 1; i <= 5; i++ {
		jti := fmt.Sprintf("token%d", i)
		store.Record(jti, now) //nolint:errcheck
		time.Sleep(10 * time.Millisecond)
	}

	// Trigger eviction
	_, _ = store.Seen("token5") //nolint:errcheck

	// Store size should be limited to MaxSize
	if store.Size() > 3 {
		t.Errorf("Store size %d exceeds MaxSize 3 after LRU eviction", store.Size())
	}

	// Verify eviction stats (LRU policy should have triggered evictions)
	if store.evictionStats.LRUEvictions == 0 {
		t.Error("LRUEvictions should be > 0")
	}
}

// TestCompositeEviction verifies composite policy (TTL part working).
func TestCompositeEviction(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "replay_composite.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              200 * time.Millisecond,
		SnapshotInterval: 1 * time.Hour,
		EvictionPolicy: &CompositeEvictionPolicy{
			Policies: []EvictionPolicy{
				&TTLEvictionPolicy{TTL: 200 * time.Millisecond},
				&SizeBasedEvictionPolicy{MaxSize: 10}, // Large enough to not trigger
			},
		},
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	// Record 2 tokens
	store.Record("token1", time.Now()) //nolint:errcheck
	store.Record("token2", time.Now()) //nolint:errcheck

	// Verify both exist
	if seen, _ := store.Seen("token1"); !seen {
		t.Error("token1 should exist initially")
	}
	if seen, _ := store.Seen("token2"); !seen {
		t.Error("token2 should exist initially")
	}

	// Wait for TTL expiration
	time.Sleep(250 * time.Millisecond)

	// Both should be evicted by TTL policy in composite
	if seen, _ := store.Seen("token1"); seen {
		t.Error("token1 should be evicted by TTL in composite policy")
	}
	if seen, _ := store.Seen("token2"); seen {
		t.Error("token2 should be evicted by TTL in composite policy")
	}
}

// TestParseEvictionPolicy verifies policy parsing from env var strings.
func TestParseEvictionPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected string
	}{
		{"default", "", "ttl"},
		{"ttl", "ttl", "ttl"},
		{"lru", "lru", "lru"},
		{"size", "size", "size"},
		{"composite_ttl_size", "ttl+size", "composite(ttl,size)"},
		{"composite_ttl_lru", "ttl+lru", "composite(ttl,lru)"},
		{"none", "none", "ttl"},      // None falls back to TTL
		{"unknown", "foobar", "ttl"}, // Unknown falls back to TTL
		{"case_insensitive", "TTL", "ttl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := ParseEvictionPolicy(tt.policy, 10*time.Minute, 1000)
			if policy.Name() != tt.expected {
				t.Errorf("ParseEvictionPolicy(%q) = %q, want %q", tt.policy, policy.Name(), tt.expected)
			}
		})
	}
}

// TestEvictionMetrics verifies metrics are recorded correctly.
func TestEvictionMetrics(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "replay_metrics.wal")

	metrics := &TestReplayMetrics{
		evictions: make(map[string]int),
	}

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              50 * time.Millisecond,
		SnapshotInterval: 1 * time.Hour,
		Metrics:          metrics,
		EvictionPolicy:   &TTLEvictionPolicy{TTL: 50 * time.Millisecond},
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	// Record tokens
	store.Record("token1", time.Now()) //nolint:errcheck
	store.Record("token2", time.Now()) //nolint:errcheck

	// Verify cache miss for new token
	if metrics.cacheHits != 0 {
		t.Errorf("cacheHits should be 0 initially, got %d", metrics.cacheHits)
	}

	// Verify cache hit for existing token
	_, _ = store.Seen("token1") //nolint:errcheck
	if metrics.cacheHits != 1 {
		t.Errorf("cacheHits should be 1 after Seen(), got %d", metrics.cacheHits)
	}

	// Wait for TTL expiration
	time.Sleep(100 * time.Millisecond)

	// Trigger eviction
	_, _ = store.Seen("token1") //nolint:errcheck

	// Verify eviction metric
	if metrics.evictions["ttl"] == 0 {
		t.Error("TTL eviction metric should be recorded")
	}

	// Verify store size metric was called
	if metrics.storeSizeCalls == 0 {
		t.Error("SetReplayStoreSize should be called")
	}
}

// TestReplayMetrics is a test implementation of DurableReplayMetrics.
type TestReplayMetrics struct {
	errors           int
	latency          time.Duration
	snapshotDuration time.Duration
	flushLatency     time.Duration
	pending          int
	evictions        map[string]int
	storeSize        int
	storeSizeCalls   int
	cacheHits        int
	cacheMisses      int
}

func (m *TestReplayMetrics) IncReplayStoreErrors() {
	m.errors++
}

func (m *TestReplayMetrics) ObserveReplayStoreLatency(d time.Duration) {
	m.latency = d
}

func (m *TestReplayMetrics) ObserveReplayWALSnapshotDuration(d time.Duration) {
	m.snapshotDuration = d
}

func (m *TestReplayMetrics) ObserveReplayWALFlushLatency(d time.Duration) {
	m.flushLatency = d
}

func (m *TestReplayMetrics) SetReplayWALPending(n int) {
	m.pending = n
}

func (m *TestReplayMetrics) IncReplayEvictions(reason string) {
	m.evictions[reason]++
}

func (m *TestReplayMetrics) SetReplayStoreSize(size int) {
	m.storeSize = size
	m.storeSizeCalls++
}

func (m *TestReplayMetrics) IncReplayCacheHit() {
	m.cacheHits++
}

func (m *TestReplayMetrics) IncReplayCacheMiss() {
	m.cacheMisses++
}

// TestEvictionConcurrency verifies concurrent access during eviction.
func TestEvictionConcurrency(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "replay_concurrent.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              5 * time.Minute,
		SnapshotInterval: 1 * time.Hour,
		EvictionPolicy:   &SizeBasedEvictionPolicy{MaxSize: 100},
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 20; j++ {
				jti := fmt.Sprintf("token-%d-%d", id, j)
				store.Record(jti, time.Now()) //nolint:errcheck
				_, _ = store.Seen(jti)        //nolint:errcheck
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify store size is within bounds
	if store.Size() > 100 {
		t.Errorf("Store size %d exceeds max size 100 after concurrent access", store.Size())
	}

	// Verify no data races (test will fail with -race flag if races exist)
}

// TestEvictionPolicyNone verifies "none" policy defaults to TTL.
func TestEvictionPolicyNone(t *testing.T) {
	policy := ParseEvictionPolicy("none", 10*time.Minute, 1000)
	if _, ok := policy.(*TTLEvictionPolicy); !ok {
		t.Errorf("ParseEvictionPolicy('none') should default to TTLEvictionPolicy, got %T", policy)
	}
}
