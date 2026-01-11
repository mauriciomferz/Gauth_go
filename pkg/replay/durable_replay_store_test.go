package replay

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDurableReplayStore_BasicOperations verifies Seen/Record functionality.
func TestDurableReplayStore_BasicOperations(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	jti := "test-jti-123"

	// Should not be seen initially
	seen, err := store.Seen(jti)
	if err != nil {
		t.Fatalf("Seen() error: %v", err)
	}
	if seen {
		t.Error("expected JTI not seen initially")
	}

	// Record JTI
	err = store.Record(jti, time.Now())
	if err != nil {
		t.Fatalf("Record() error: %v", err)
	}

	// Should be seen after recording
	seen, err = store.Seen(jti)
	if err != nil {
		t.Fatalf("Seen() error: %v", err)
	}
	if !seen {
		t.Error("expected JTI to be seen after Record()")
	}
}

// TestDurableReplayStore_Persistence verifies WAL persistence across restarts.
func TestDurableReplayStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	// Create first store instance
	store1, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store1: %v", err)
	}

	jti1 := "persistent-jti-1"
	jti2 := "persistent-jti-2"

	// Record JTIs
	if err2 := store1.Record(jti1, time.Now()); err2 != nil {
		t.Fatalf("failed to record jti1: %v", err2)
	}
	if err2 := store1.Record(jti2, time.Now()); err2 != nil {
		t.Fatalf("failed to record jti2: %v", err2)
	}

	// Close first instance
	if closeErr := store1.Close(); closeErr != nil {
		t.Fatalf("close store1: %v", closeErr)
	}

	// Create second instance (should recover from WAL)
	store2, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store2: %v", err)
	}
	t.Cleanup(func() { _ = store2.Close() })

	// Verify JTIs recovered
	seen1, _ := store2.Seen(jti1)
	seen2, _ := store2.Seen(jti2)

	if !seen1 {
		t.Error("expected jti1 to be recovered from WAL")
	}
	if !seen2 {
		t.Error("expected jti2 to be recovered from WAL")
	}
}

// TestDurableReplayStore_TTLExpiration verifies TTL-based expiration.
func TestDurableReplayStore_TTLExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              100 * time.Millisecond, // Short TTL for testing
		SnapshotInterval: 10 * time.Second,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	jti := "expiring-jti"

	// Record JTI
	if err := store.Record(jti, time.Now()); err != nil {
		t.Fatalf("failed to record jti: %v", err)
	}

	// Should be seen immediately
	seen, _ := store.Seen(jti)
	if !seen {
		t.Error("expected JTI to be seen before expiration")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should not be seen after expiration
	seen, _ = store.Seen(jti)
	if seen {
		t.Error("expected JTI to be expired after TTL")
	}
}

// TestDurableReplayStore_Snapshot verifies snapshot creation and recovery.
func TestDurableReplayStore_Snapshot(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	store1, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store1: %v", err)
	}

	// Record multiple JTIs
	for i := 0; i < 10; i++ {
		jti := "snapshot-jti-" + string(rune('0'+i))
		if err2 := store1.Record(jti, time.Now()); err2 != nil {
			t.Fatalf("failed to record jti: %v", err2)
		}
	}

	// Create manual snapshot
	err = store1.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}

	// Verify snapshot file exists
	snapshotPath := walPath + ".snapshot"
	if _, err2 := os.Stat(snapshotPath); os.IsNotExist(err2) {
		t.Error("expected snapshot file to exist")
	}

	if closeErr := store1.Close(); closeErr != nil {
		t.Fatalf("close store1: %v", closeErr)
	}

	// Recover from snapshot
	store2, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store2: %v", err)
	}
	t.Cleanup(func() { _ = store2.Close() })

	// Verify all JTIs recovered
	for i := 0; i < 10; i++ {
		jti := "snapshot-jti-" + string(rune('0'+i))
		seen, _ := store2.Seen(jti)
		if !seen {
			t.Errorf("expected %s to be recovered from snapshot", jti)
		}
	}
}

// TestDurableReplayStore_ConcurrentAccess verifies thread-safe operations.
func TestDurableReplayStore_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Concurrent Record operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			jti := "concurrent-jti-" + string(rune('0'+id))
			_ = store.Record(jti, time.Now()) //nolint:errcheck
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all JTIs recorded
	for i := 0; i < 10; i++ {
		jti := "concurrent-jti-" + string(rune('0'+i))
		seen, _ := store.Seen(jti)
		if !seen {
			t.Errorf("expected %s to be seen after concurrent Record()", jti)
		}
	}
}

// TestDurableReplayStore_Stats verifies stats reporting.
func TestDurableReplayStore_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Record JTIs
	if err := store.Record("jti-1", time.Now()); err != nil {
		t.Fatalf("failed to record jti-1: %v", err)
	}
	if err := store.Record("jti-2", time.Now()); err != nil {
		t.Fatalf("failed to record jti-2: %v", err)
	}
	if err := store.Record("jti-3", time.Now()); err != nil {
		t.Fatalf("failed to record jti-3: %v", err)
	}

	stats := store.Stats()
	if stats.TotalEntries != 3 {
		t.Errorf("expected 3 total entries, got %d", stats.TotalEntries)
	}
	if stats.ActiveEntries != 3 {
		t.Errorf("expected 3 active entries, got %d", stats.ActiveEntries)
	}
	if stats.WALPath != walPath {
		t.Errorf("expected WAL path %s, got %s", walPath, stats.WALPath)
	}
}

// TestDurableReplayStoreAdapter_AAP001Integration verifies adapter compatibility.
func TestDurableReplayStoreAdapter_AAP001Integration(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	adapter := NewDurableReplayStoreAdapter(store)

	jti := "adapter-jti-123"

	// Test adapter Seen
	seen, err := adapter.Seen(jti)
	if err != nil {
		t.Fatalf("adapter.Seen() error: %v", err)
	}
	if seen {
		t.Error("expected JTI not seen initially via adapter")
	}

	// Test adapter Record
	err = adapter.Record(jti, time.Now())
	if err != nil {
		t.Fatalf("adapter.Record() error: %v", err)
	}

	// Verify via adapter
	seen, err = adapter.Seen(jti)
	if err != nil {
		t.Fatalf("adapter.Seen() error: %v", err)
	}
	if !seen {
		t.Error("expected JTI to be seen after adapter.Record()")
	}
}

// TestDurableReplayStore_AutomaticSnapshot verifies periodic snapshot scheduling.
func TestDurableReplayStore_AutomaticSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 200 * time.Millisecond, // Fast for testing
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Record JTIs
	_ = store.Record("auto-snapshot-jti-1", time.Now()) //nolint:errcheck
	_ = store.Record("auto-snapshot-jti-2", time.Now()) //nolint:errcheck

	// Wait for automatic snapshot
	time.Sleep(300 * time.Millisecond)

	// Verify snapshot file exists
	snapshotPath := walPath + ".snapshot"
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		t.Error("expected automatic snapshot file to exist")
	}
}

// TestDurableReplayStore_GracefulShutdown verifies final snapshot on Close().
func TestDurableReplayStore_GracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Hour, // Long interval (won't trigger automatically)
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Record JTI
	_ = store.Record("shutdown-jti", time.Now()) //nolint:errcheck

	// Close (should trigger final snapshot)
	if closeErr := store.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	// Verify snapshot created on shutdown
	snapshotPath := walPath + ".snapshot"
	if _, err2 := os.Stat(snapshotPath); os.IsNotExist(err2) {
		t.Error("expected final snapshot file after Close()")
	}

	// Verify recovery after shutdown
	store2, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store2: %v", err)
	}
	t.Cleanup(func() { _ = store2.Close() })

	seen, _ := store2.Seen("shutdown-jti")
	if !seen {
		t.Error("expected shutdown-jti to be recovered after final snapshot")
	}
}

// TestDurableReplayStore_Size verifies Size() reporting.
func TestDurableReplayStore_Size(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "replay.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              1 * time.Hour,
		SnapshotInterval: 10 * time.Second,
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Initial size should be 0
	if size := store.Size(); size != 0 {
		t.Errorf("expected initial size 0, got %d", size)
	}

	// Record JTIs
	store.Record("size-jti-1", time.Now()) //nolint:errcheck
	store.Record("size-jti-2", time.Now()) //nolint:errcheck
	store.Record("size-jti-3", time.Now()) //nolint:errcheck

	// Size should be 3
	if size := store.Size(); size != 3 {
		t.Errorf("expected size 3, got %d", size)
	}
}
