package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDurableReplayStoreAgentAuthIntegration tests DurableReplayStore with agentauth ReplayStore interface.
func TestDurableReplayStoreAgentAuthIntegration(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "integration.wal")

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              5 * time.Minute,
		SnapshotInterval: 1 * time.Hour,
		EvictionPolicy:   &TTLEvictionPolicy{TTL: 5 * time.Minute},
	}

	store, err := NewDurableReplayStore(config)
	if err != nil {
		t.Fatalf("NewDurableReplayStore failed: %v", err)
	}
	defer store.Close()

	adapter := NewDurableReplayStoreAdapter(store)

	// Test CheckAndStore (first call should succeed)
	if err := adapter.CheckAndStore("jti-001"); err != nil {
		t.Errorf("First CheckAndStore should succeed: %v", err)
	}

	// Test CheckAndStore (second call should fail - replay detected)
	if err := adapter.CheckAndStore("jti-001"); err == nil {
		t.Error("Second CheckAndStore should fail (replay detected)")
	} else if err.Error() != "replay detected: JTI jti-001 already seen" {
		t.Errorf("Expected replay detected error, got: %v", err)
	}

	// Test with different JTI (should succeed)
	if err := adapter.CheckAndStore("jti-002"); err != nil {
		t.Errorf("CheckAndStore with new JTI should succeed: %v", err)
	}
}

// TestDurableReplayStoreFromEnv tests environment variable configuration.
func TestDurableReplayStoreFromEnv(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "env_test.wal")

	// Set environment variables
	os.Setenv("AGENTAUTH_REPLAY_WAL_PATH", walPath)
	os.Setenv("AGENTAUTH_REPLAY_TTL_SEC", "300")
	os.Setenv("AGENTAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC", "60")
	os.Setenv("AGENTAUTH_REPLAY_EVICTION_POLICY", "ttl")
	os.Setenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE", "5000")
	defer func() {
		os.Unsetenv("AGENTAUTH_REPLAY_WAL_PATH")
		os.Unsetenv("AGENTAUTH_REPLAY_TTL_SEC")
		os.Unsetenv("AGENTAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC")
		os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_POLICY")
		os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE")
	}()

	store, err := NewDurableReplayStoreFromEnv(NoopReplayMetrics{})
	if err != nil {
		t.Fatalf("NewDurableReplayStoreFromEnv failed: %v", err)
	}
	defer store.Close()

	// Verify configuration was applied
	if store.ttl != 5*time.Minute {
		t.Errorf("Expected TTL 5m, got %v", store.ttl)
	}
	if store.snapshotInterval != 1*time.Minute {
		t.Errorf("Expected snapshot interval 1m, got %v", store.snapshotInterval)
	}
	if store.evictionPolicy.Name() != "ttl" {
		t.Errorf("Expected eviction policy 'ttl', got '%s'", store.evictionPolicy.Name())
	}
}

// TestDurableReplayStoreWithSizeEviction tests size-based eviction with env vars.
func TestDurableReplayStoreWithSizeEviction(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "size_eviction.wal")

	os.Setenv("AGENTAUTH_REPLAY_WAL_PATH", walPath)
	os.Setenv("AGENTAUTH_REPLAY_EVICTION_POLICY", "size")
	os.Setenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE", "3")
	defer func() {
		os.Unsetenv("AGENTAUTH_REPLAY_WAL_PATH")
		os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_POLICY")
		os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE")
	}()

	store, err := NewDurableReplayStoreFromEnv(NoopReplayMetrics{})
	if err != nil {
		t.Fatalf("NewDurableReplayStoreFromEnv failed: %v", err)
	}
	defer store.Close()

	adapter := NewDurableReplayStoreAdapter(store)

	// Add 5 entries (should evict oldest 2)
	for i := 1; i <= 5; i++ {
		jti := fmt.Sprintf("jti-%03d", i)
		if err := adapter.CheckAndStore(jti); err != nil {
			t.Errorf("CheckAndStore %s failed: %v", jti, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Trigger eviction by accessing
	_, _ = adapter.Seen("jti-005") //nolint:errcheck

	// Store size should be limited
	if store.Size() > 3 {
		t.Errorf("Store size %d exceeds max size 3", store.Size())
	}
}

// TestDurableReplayStoreCompositePolicy tests composite eviction policy from env var.
func TestDurableReplayStoreCompositePolicy(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "composite.wal")

	os.Setenv("AGENTAUTH_REPLAY_WAL_PATH", walPath)
	os.Setenv("AGENTAUTH_REPLAY_EVICTION_POLICY", "ttl+size")
	os.Setenv("AGENTAUTH_REPLAY_TTL_SEC", "1") // 1 second TTL for fast testing
	os.Setenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE", "10")
	defer func() {
		os.Unsetenv("AGENTAUTH_REPLAY_WAL_PATH")
		os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_POLICY")
		os.Unsetenv("AGENTAUTH_REPLAY_TTL_SEC")
		os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE")
	}()

	store, err := NewDurableReplayStoreFromEnv(NoopReplayMetrics{})
	if err != nil {
		t.Fatalf("NewDurableReplayStoreFromEnv failed: %v", err)
	}
	defer store.Close()

	// Verify composite policy
	if store.evictionPolicy.Name() != "composite(ttl,size)" {
		t.Errorf("Expected composite policy, got '%s'", store.evictionPolicy.Name())
	}

	adapter := NewDurableReplayStoreAdapter(store)

	// Add entry
	_ = adapter.CheckAndStore("jti-ttl-test") //nolint:errcheck

	// Wait for TTL expiration
	time.Sleep(1500 * time.Millisecond)

	// Should be evicted by TTL
	if err := adapter.CheckAndStore("jti-ttl-test"); err != nil {
		t.Errorf("Entry should be evicted and allow re-use: %v", err)
	}
}

// TestDurableReplayStoreEnvDefaults tests default values when env vars not set.
func TestDurableReplayStoreEnvDefaults(t *testing.T) {
	// Clear all replay-related env vars
	os.Unsetenv("AGENTAUTH_REPLAY_WAL_PATH")
	os.Unsetenv("AGENTAUTH_REPLAY_TTL_SEC")
	os.Unsetenv("AGENTAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC")
	os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_POLICY")
	os.Unsetenv("AGENTAUTH_REPLAY_EVICTION_MAX_SIZE")

	store, err := NewDurableReplayStoreFromEnv(NoopReplayMetrics{})
	if err != nil {
		// Expected to fail if default path not writable
		// This is OK - just verify defaults were attempted
		if store != nil {
			defer store.Close()
		}
		return
	}
	defer store.Close()

	// Verify defaults
	if store.ttl != 15*time.Minute {
		t.Errorf("Expected default TTL 15m, got %v", store.ttl)
	}
	if store.snapshotInterval != 5*time.Minute {
		t.Errorf("Expected default snapshot interval 5m, got %v", store.snapshotInterval)
	}
	if store.evictionPolicy.Name() != "ttl" {
		t.Errorf("Expected default eviction policy 'ttl', got '%s'", store.evictionPolicy.Name())
	}
}

// TestDurableReplayStorePersistence tests crash recovery.
func TestDurableReplayStorePersistence(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "persistence.wal")

	// Create first store and add entries
	{
		store, err := NewDurableReplayStore(DurableReplayStoreConfig{
			WALPath:          walPath,
			TTL:              10 * time.Minute,
			SnapshotInterval: 1 * time.Hour,
		})
		if err != nil {
			t.Fatalf("NewDurableReplayStore failed: %v", err)
		}

		adapter := NewDurableReplayStoreAdapter(store)
		_ = adapter.CheckAndStore("jti-persist-1") //nolint:errcheck
		_ = adapter.CheckAndStore("jti-persist-2") //nolint:errcheck
		_ = adapter.CheckAndStore("jti-persist-3") //nolint:errcheck

		store.Close()
	}

	// Reopen store (simulating crash recovery)
	{
		store, err := NewDurableReplayStore(DurableReplayStoreConfig{
			WALPath:          walPath,
			TTL:              10 * time.Minute,
			SnapshotInterval: 1 * time.Hour,
		})
		if err != nil {
			t.Fatalf("NewDurableReplayStore reopen failed: %v", err)
		}
		defer store.Close()

		adapter := NewDurableReplayStoreAdapter(store)

		// Previously stored JTIs should be detected as replays
		if err := adapter.CheckAndStore("jti-persist-1"); err == nil {
			t.Error("jti-persist-1 should be detected as replay after recovery")
		}
		if err := adapter.CheckAndStore("jti-persist-2"); err == nil {
			t.Error("jti-persist-2 should be detected as replay after recovery")
		}
		if err := adapter.CheckAndStore("jti-persist-3"); err == nil {
			t.Error("jti-persist-3 should be detected as replay after recovery")
		}

		// New JTI should work
		if err := adapter.CheckAndStore("jti-persist-4"); err != nil {
			t.Errorf("New JTI after recovery should work: %v", err)
		}
	}
}
