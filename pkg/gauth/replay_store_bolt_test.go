// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package gauth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoltReplayStore_CheckAndRecord(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "replay_test.db")

	store, err := NewBoltReplayStore(dbPath, 1*time.Hour)
	require.NoError(t, err)
	defer store.Close()

	// First use should succeed
	err = store.CheckAndRecord("jti-12345")
	assert.NoError(t, err)

	// Second use should fail (replay detected)
	err = store.CheckAndRecord("jti-12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "replay detected")

	// Different JTI should succeed
	err = store.CheckAndRecord("jti-67890")
	assert.NoError(t, err)
}

func TestBoltReplayStore_Expiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "replay_exp_test.db")

	// Use very short TTL for testing
	store, err := NewBoltReplayStore(dbPath, 100*time.Millisecond)
	require.NoError(t, err)
	defer store.Close()

	// Record JTI
	err = store.CheckAndRecord("jti-expire")
	assert.NoError(t, err)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be able to use again after expiration
	err = store.CheckAndRecord("jti-expire")
	assert.NoError(t, err)
}

func TestBoltReplayStore_Count(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "replay_count_test.db")

	store, err := NewBoltReplayStore(dbPath, 1*time.Hour)
	require.NoError(t, err)
	defer store.Close()

	// Should start empty
	count, err := store.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add some JTIs
	_ = store.CheckAndRecord("jti-1")
	_ = store.CheckAndRecord("jti-2")
	_ = store.CheckAndRecord("jti-3")

	count, err = store.Count()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestBoltReplayStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "replay_persist_test.db")

	// Create store and record JTI
	store1, err := NewBoltReplayStore(dbPath, 1*time.Hour)
	require.NoError(t, err)

	err = store1.CheckAndRecord("jti-persist")
	assert.NoError(t, err)
	store1.Close()

	// Reopen store and verify JTI still exists
	store2, err := NewBoltReplayStore(dbPath, 1*time.Hour)
	require.NoError(t, err)
	defer store2.Close()

	err = store2.CheckAndRecord("jti-persist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "replay detected")
}

func TestBoltReplayStore_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "replay.db")

	store, err := NewBoltReplayStore(nestedPath, 1*time.Hour)
	require.NoError(t, err)
	defer store.Close()

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(nestedPath))
	assert.NoError(t, err)
}
