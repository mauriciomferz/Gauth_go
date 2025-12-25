package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexedDelegationStore_Pruning_Background(t *testing.T) {
	// Setup temp db
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "delegation_test.db")
	t.Setenv("GAUTH_ALLOW_UNSAFE_BOLTDB", "1")

	// Config with short intervals for testing
	config := PruneConfig{
		Enabled:          true,
		Interval:         500 * time.Millisecond,
		RetentionPeriod:  100 * time.Millisecond, // Expires quickly
		InactivityPeriod: 200 * time.Millisecond,
	}

	store, err := NewIndexedDelegationStore(dbPath, &config)
	require.NoError(t, err)
	defer store.Close()

	// 1. Add expired record
	expiredRecord := &DelegationRecord{
		ID:        "expired-1",
		Subject:   "sub1",
		Delegate:  "del1",
		Status:    "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired long ago
		IssuedAt:  time.Now(),
	}
	err = store.Store(expiredRecord)
	require.NoError(t, err)

	// 2. Add active record
	activeRecord := &DelegationRecord{
		ID:        "active-1",
		Subject:   "sub2",
		Delegate:  "del2",
		Status:    "active",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		IssuedAt:  time.Now(),
	}
	err = store.Store(activeRecord)
	require.NoError(t, err)

	// Verify both exist
	rec, err := store.Get("expired-1")
	require.NoError(t, err)
	assert.NotNil(t, rec)

	rec, err = store.Get("active-1")
	require.NoError(t, err)
	assert.NotNil(t, rec)

	// 3. Wait for pruning interval
	// Interval is 500ms, wait 1s to be safe
	time.Sleep(1 * time.Second)

	// 4. Verify expired is gone
	rec, err = store.Get("expired-1")
	assert.Error(t, err) // Should be not found
	assert.Nil(t, rec)

	// 5. Verify active remains
	rec, err = store.Get("active-1")
	require.NoError(t, err)
	assert.NotNil(t, rec)

	// Check stats
	stats := store.GetStats()
	assert.Greater(t, stats.PrunedRecords, int64(0))
}

func TestIndexedDelegationStore_Lifecycle(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "lifecycle_test.db")
	t.Setenv("GAUTH_ALLOW_UNSAFE_BOLTDB", "1")

	config := PruneConfig{
		Enabled:  true,
		Interval: 10 * time.Second, // Long interval
	}

	store, err := NewIndexedDelegationStore(dbPath, &config)
	require.NoError(t, err)

	start := time.Now()
	err = store.Close()
	require.NoError(t, err)

	// Should return quickly
	assert.WithinDuration(t, start, time.Now(), 1*time.Second)
}
