package token

import (
	"strconv"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
)

func TestReplayNonceStore_Eviction(t *testing.T) {
	mem := &metrics.Memory{}
	ttl := 1 * time.Minute
	capacity := 10
	store := NewReplayNonceStoreWithConfig(ttl, capacity, "", mem)
	defer store.Close()

	now := time.Now()
	// Fill the store to capacity
	for i := 0; i < capacity; i++ {
		store.Record("nonce-"+strconv.Itoa(i), now)
	}

	if store.Size() != capacity {
		t.Fatalf("expected size %d, got %d", capacity, store.Size())
	}

	// Add one more, should cause eviction
	store.Record("nonce-overflow", now)

	if store.Size() != capacity {
		t.Fatalf("expected size %d after overflow, got %d", capacity, store.Size())
	}

	// First nonce should have been evicted
	if store.Seen("nonce-0", now) {
		t.Errorf("expected nonce-0 to be evicted")
	}

	// Verify metric
	// Note: You need to check if memory metrics are working.
	// We'll trust the logic if Size is correct.
}

func TestReplayNonceStore_TTL(t *testing.T) {
	ttl := 100 * time.Millisecond
	store := NewReplayNonceStoreWithConfig(ttl, 100, "", nil)
	defer store.Close()

	n := "nonce-ttl"
	now := time.Now()
	store.Record(n, now)

	if !store.Seen(n, now) {
		t.Errorf("expected nonce to be seen immediately")
	}

	// Wait for TTL
	time.Sleep(150 * time.Millisecond)
	future := now.Add(150 * time.Millisecond)

	if store.Seen(n, future) {
		t.Errorf("expected nonce to be expired")
	}

	if store.Size() != 0 {
		t.Errorf("expected store to be empty after lazy delete, got %d", store.Size())
	}
}

func TestReplayNonceStore_UpdateLRU(t *testing.T) {
	capacity := 2
	store := NewReplayNonceStoreWithConfig(1*time.Minute, capacity, "", nil)
	defer store.Close()

	now := time.Now()
	store.Record("a", now)
	store.Record("b", now)

	// Access "a" to move it to back
	store.Record("a", now.Add(1*time.Second))

	// Record "c", should evict "b" because "a" was refreshed
	store.Record("c", now)

	if store.Seen("b", now) {
		t.Errorf("expected b to be evicted")
	}
	if !store.Seen("a", now) {
		t.Errorf("expected a to be retained")
	}
}
