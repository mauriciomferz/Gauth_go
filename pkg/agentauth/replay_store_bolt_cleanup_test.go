package agentauth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoltReplayStore_Cleanup(t *testing.T) {
	// Setup temporary DB path
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cleanup_test.db")

	// Create store with reasonable TTL (store uses second precision)
	ttl := 2 * time.Second

	// Allow unsafe bolt for testing
	t.Setenv("AGENTAUTH_ALLOW_UNSAFE_BOLTDB", "1")

	store, err := NewBoltReplayStore(dbPath, ttl)
	require.NoError(t, err)
	defer store.Close()

	// 1. Record a JTI
	jti := "test-jti-cleanup"
	err = store.CheckAndRecord(jti)
	require.NoError(t, err)

	// Verify it exists
	err = store.CheckAndRecord(jti)
	assert.Error(t, err) // Should return error as it already exists/replayed

	count, err := store.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 2. Wait for expiration
	time.Sleep(3 * time.Second)

	// 3. Run Cleanup
	evicted, err := store.Cleanup(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, evicted, "Should evict 1 item")

	// 4. Verify it's gone
	count, err = store.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify we can use the same JTI again (since it expired and was cleaned)
	err = store.CheckAndRecord(jti)
	require.NoError(t, err, "Should be able to record JTI again after cleanup")
}

func TestBoltReplayStore_Lifecycle(t *testing.T) {
	// Verify no goroutine leaks on rapid create/close
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "lifecycle_test.db")
	t.Setenv("AGENTAUTH_ALLOW_UNSAFE_BOLTDB", "1")

	store, err := NewBoltReplayStore(dbPath, time.Minute)
	require.NoError(t, err)

	// Close immediately
	start := time.Now()
	err = store.Close()
	require.NoError(t, err)

	// Should allow closing within reasonable time (waiting for cleanup goroutine)
	assert.WithinDuration(t, start, time.Now(), 1*time.Second)
}
