package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntry(t *testing.T) {
	t.Run("Create Entry", func(t *testing.T) {
		entry := NewEntry(TypeAuth).
			WithActor("user123", ActorUser).
			WithAction(ActionLogin).
			WithTarget("webapp", "application").
			WithResult(ResultSuccess).
			WithMetadata("ip", "192.168.1.1")

		assert.Equal(t, TypeAuth, entry.Type)
		assert.Equal(t, "user123", entry.ActorID)
		assert.Equal(t, ActorUser, entry.ActorType)
		assert.Equal(t, ActionLogin, entry.Action)
		assert.Equal(t, ResultSuccess, entry.Result)
		assert.Equal(t, "192.168.1.1", entry.Metadata["ip"])
	})

	t.Run("Hash Chain", func(t *testing.T) {
		entry1 := NewEntry(TypeAuth)
		entry2 := NewEntry(TypeAuth)
		entry2.PrevHash = entry1.CalculateHash()

		hash1 := entry1.CalculateHash()
		hash2 := entry2.CalculateHash()

		assert.NotEqual(t, hash1, hash2)
		assert.Equal(t, hash1, entry2.PrevHash)
	})
}

func TestFileStorage(t *testing.T) {
	t.Run("Store and Retrieve", func(t *testing.T) {
		dir := t.TempDir()
		storage, err := NewFileStorage(FileConfig{
			Directory: dir,
		})
		require.NoError(t, err)
		defer storage.Close()
		ctx := context.Background()

		entry := NewEntry(TypeAuth).
			WithActor("user123", ActorUser).
			WithAction(ActionLogin).
			WithResult(ResultSuccess)

		err = storage.Store(ctx, entry)
		require.NoError(t, err)

		retrieved, err := storage.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entry.ID, retrieved.ID)
		assert.Equal(t, entry.Type, retrieved.Type)
		assert.Equal(t, entry.ActorID, retrieved.ActorID)
	})

	t.Run("Search", func(t *testing.T) {
		dir := t.TempDir()
		storage, err := NewFileStorage(FileConfig{
			Directory: dir,
		})
		require.NoError(t, err)
		defer storage.Close()
		ctx := context.Background()

		// Create test entries
		entries := []*Entry{
			NewEntry(TypeAuth).WithActor("user1", ActorUser),
			NewEntry(TypeToken).WithActor("user1", ActorUser),
			NewEntry(TypeAuth).WithActor("user2", ActorUser),
		}

		for _, entry := range entries {
			require.NoError(t, storage.Store(ctx, entry))
		}

		// Search by type
		results, err := storage.Search(ctx, &Filter{
			Types: []string{TypeAuth},
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// Search by actor
		results, err = storage.Search(ctx, &Filter{
			ActorIDs: []string{"user1"},
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Chain", func(t *testing.T) {
		dir := t.TempDir()
		storage, err := NewFileStorage(FileConfig{
			Directory: dir,
		})
		require.NoError(t, err)
		defer storage.Close()
		ctx := context.Background()

		chainID := "test-chain"
		entries := []*Entry{
			NewEntry(TypeAuth).WithAction(ActionLogin),
			NewEntry(TypeResource).WithAction(ActionResourceAccess),
		}
		for _, entry := range entries {
			entry.ChainID = chainID
			require.NoError(t, storage.Store(ctx, entry))
		}

		chain, err := storage.GetChain(ctx, chainID)
		require.NoError(t, err)
		assert.Len(t, chain, 2)
	})

	t.Run("Cleanup", func(t *testing.T) {
		dir := t.TempDir()
		storage, err := NewFileStorage(FileConfig{
			Directory: dir,
		})
		require.NoError(t, err)
		defer storage.Close()
		ctx := context.Background()

		old := NewEntry(TypeAuth)
		old.Timestamp = time.Now().Add(-24 * time.Hour)
		require.NoError(t, storage.Store(ctx, old))

		// Force log rotation by updating the file's mod time to be old
		files, err := filepath.Glob(filepath.Join(dir, "audit-*.log"))
		require.NoError(t, err)
		require.NotEmpty(t, files)
		require.NoError(t, os.Chtimes(files[0], time.Now().Add(-24*time.Hour), time.Now().Add(-24*time.Hour)))

		recent := NewEntry(TypeAuth)
		require.NoError(t, storage.Store(ctx, recent))

		_ = storage.Cleanup(ctx, time.Now().Add(-12*time.Hour))
		// File-based cleanup only removes files older than cutoff, so no error expected

		// Old entry should be gone (not found)
		_, err = storage.GetByID(ctx, old.ID)
		assert.Error(t, err)

		// Recent entry should still exist
		_, err = storage.GetByID(ctx, recent.ID)
		assert.NoError(t, err)
	})
}

func TestRedisStorage(t *testing.T) {
	// Skip if no Redis available
	storage, err := NewRedisStorage(RedisConfig{
		Addresses: []string{"localhost:6379"},
		KeyPrefix: "test:",
	})
	if err != nil {
		t.Skip("Redis not available:", err)
	}
	defer storage.Close()

	ctx := context.Background()

	t.Run("Store and Retrieve", func(t *testing.T) {
		entry := NewEntry(TypeAuth).
			WithActor("user123", ActorUser).
			WithAction(ActionLogin).
			WithResult(ResultSuccess)

		err := storage.Store(ctx, entry)
		require.NoError(t, err)

		retrieved, err := storage.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entry.ID, retrieved.ID)
		assert.Equal(t, entry.Type, retrieved.Type)
		assert.Equal(t, entry.ActorID, retrieved.ActorID)
	})

	// Add more Redis-specific tests...
}

func TestSQLStorage(t *testing.T) {
	// Skip if no PostgreSQL available
	storage, err := NewSQLStorage(SQLConfig{
		Driver: "postgres",
		DSN:    "postgres://localhost/test?sslmode=disable",
	})
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer storage.Close()

	ctx := context.Background()

	t.Run("Store and Retrieve", func(t *testing.T) {
		entry := NewEntry(TypeAuth).
			WithActor("user123", ActorUser).
			WithAction(ActionLogin).
			WithResult(ResultSuccess)

		err := storage.Store(ctx, entry)
		require.NoError(t, err)

		retrieved, err := storage.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entry.ID, retrieved.ID)
		assert.Equal(t, entry.Type, retrieved.Type)
		assert.Equal(t, entry.ActorID, retrieved.ActorID)
	})

	// Add more SQL-specific tests...
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics("test")

	t.Run("Observe Entry", func(t *testing.T) {
		entry := NewEntry(TypeAuth).
			WithAction(ActionLogin).
			WithResult(ResultSuccess)

		metrics.ObserveEntry(entry, time.Millisecond)
		// Note: Can't easily test Prometheus metrics directly
	})

	t.Run("Storage Operations", func(t *testing.T) {
		metrics.ObserveStorageOperation("store", "redis", time.Millisecond)
		metrics.ObserveStorageError("store", "connection_failed")
		metrics.SetBatchSize(10)
		metrics.ObserveChainLength("auth", 5)
	})
}
