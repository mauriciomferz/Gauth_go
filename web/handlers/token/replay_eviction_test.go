package token

import (
	"testing"
	"time"
)

func TestReplayStore_CapacityEnforcement(t *testing.T) {
	// Create store with capacity 2
	store := NewReplayNonceStoreWithConfig(1*time.Hour, 2, "", nil)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("store close: %v", err)
		}
	})

	// Insert 3 items
	now := time.Now()
	if err := store.CheckAndStore("1"); err != nil {
		t.Fatalf("CheckAndStore failed: %v", err)
	}
	store.Record("2", now.Add(1*time.Second)) // store 2 slightly later

	// Check size
	if size := store.Size(); size != 2 {
		t.Errorf("expected size 2, got %d", size)
	}

	// Insert 3rd item, should evict oldest ("1")
	store.Record("3", now.Add(2*time.Second))

	if size := store.Size(); size != 2 {
		t.Errorf("expected size 2 after eviction, got %d", size)
	}

	// Calculate seen
	if store.Seen("1", now.Add(3*time.Second)) {
		t.Error("expected '1' to be evicted")
	}
	if !store.Seen("2", now.Add(3*time.Second)) {
		t.Error("expected '2' to be present")
	}
	if !store.Seen("3", now.Add(3*time.Second)) {
		t.Error("expected '3' to be present")
	}
}

func TestReplayStore_BackgroundCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping background cleanup test in short mode")
	}

	// Create store with very short TTL
	ttl := 100 * time.Millisecond
	store := NewReplayNonceStore(ttl)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("store close: %v", err)
		}
	})

	if err := store.CheckAndStore("bg-test"); err != nil {
		t.Fatalf("CheckAndStore failed: %v", err)
	}

	// Wait for TTL + cleanup interval (cleanup runs at ttl/10 = 10ms, but min 5s in code)
	// Wait, I set min interval to 5s in code. That's too long for unit test.
	// I should probably make the interval configurable or just wait 5s?
	// 5s is long for a test.
	// Let's modify the code to allow test overrides or just wait 6s if not short.

	// Actually, for unit testing, I can rely on 'Seen' lazy cleanup if background fails,
	// but I want to test background.
	// The MinInterval is 5s.

	time.Sleep(6 * time.Second)

	// Check size directly - should be 0 without calling Seen (which does lazy cleanup)
	// However, Size() relies on len(map) which is only reduced by delete()
	// delete() happens in backgroundCleanup or Seen/Record.
	// If background cleanup worked, map should be empty.

	if size := store.Size(); size != 0 {
		t.Errorf("expected empty store after background cleanup, got size %d", size)
	}
}
