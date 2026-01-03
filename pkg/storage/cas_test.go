package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalCAS_StoreAndRetrieve(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewLocalCAS(tempDir)
	require.NoError(t, err)

	ctx := context.Background()
	data := []byte("hello world")

	// Store
	hash, err := store.Put(ctx, data)
	require.NoError(t, err)

	// Verify Hash
	expectedHash := sha256.Sum256(data)
	assert.Equal(t, hex.EncodeToString(expectedHash[:]), hash)

	// Retrieve
	retrieved, err := store.Get(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestLocalCAS_Deduplication(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewLocalCAS(tempDir)
	require.NoError(t, err)
	ctx := context.Background()

	data := []byte("duplicate data")

	hash1, err := store.Put(ctx, data)
	require.NoError(t, err)

	hash2, err := store.Put(ctx, data)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2)

	// Check only one file exists
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(files))
}

func TestLocalCAS_IntegrityCheck(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewLocalCAS(tempDir)
	require.NoError(t, err)
	ctx := context.Background()

	data := []byte("integrity test data")
	hash, err := store.Put(ctx, data)
	require.NoError(t, err)

	// Verify intact
	err = store.VerifyIntegrity(ctx, hash)
	assert.NoError(t, err)

	// Tamper with file
	path := filepath.Join(tempDir, hash)
	// Must change permission back to write
	err = os.Chmod(path, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(path, []byte("tampered data"), 0o644)
	require.NoError(t, err)

	// Verify failure
	err = store.VerifyIntegrity(ctx, hash)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integrity failure")
}

func TestLocalCAS_Get_InvalidHash(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewLocalCAS(tempDir)
	require.NoError(t, err)

	_, err = store.Get(context.Background(), "invalid-hash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hash format")
}

func TestLocalCAS_Get_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewLocalCAS(tempDir)
	require.NoError(t, err)

	// Valid hash format (64 chars) but doesn't exist
	fakeHash := "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = store.Get(context.Background(), fakeHash)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence not found")
}
