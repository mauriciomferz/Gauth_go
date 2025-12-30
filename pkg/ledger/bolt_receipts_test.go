package ledger

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/anchor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltReceiptStore_AppendAndRetrieve(t *testing.T) {
	// 1. Setup DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_receipts.db")
	db, err := bolt.Open(dbPath, 0o600, nil)
	require.NoError(t, err)
	defer db.Close()

	// 2. Init Store
	store, err := NewBoltReceiptStore(db)
	require.NoError(t, err)

	// 3. Append Receipt
	r1 := anchor.ExternalAnchorReceipt{
		Hash:           "hash-1",
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Provider:       "test-provider",
		Version:        1,
		LatencySeconds: 0.1,
	}
	stored1, err := store.Append(r1)
	require.NoError(t, err)
	assert.Equal(t, r1.Hash, stored1.Hash)
	assert.NotEmpty(t, stored1.ChainHash)
	assert.Empty(t, stored1.PrevHash)

	// 4. Append Second Receipt
	r2 := anchor.ExternalAnchorReceipt{
		Hash:           "hash-2",
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Provider:       "test-provider",
		Version:        1,
		LatencySeconds: 0.2,
	}
	stored2, err := store.Append(r2)
	require.NoError(t, err)
	assert.Equal(t, r2.Hash, stored2.Hash)
	assert.Equal(t, stored1.ChainHash, stored2.PrevHash)

	// 5. Verify Latest
	latest := store.Latest()
	assert.Equal(t, stored2.ChainHash, latest.ChainHash)

	// 6. Verify Persistence (New Store instance)
	store2, err := NewBoltReceiptStore(db)
	require.NoError(t, err)
	latest2 := store2.Latest()
	assert.Equal(t, latest.ChainHash, latest2.ChainHash)

	// 7. Verify Incremental Stub (should be OK)
	status, _, _ := store2.VerifyIncremental()
	assert.Equal(t, anchor.ExternalReceiptStatusOK, status)
}

func TestBoltReceiptStore_ManualVerification(t *testing.T) {
	// 1. Setup DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "verify_test.db")
	db, err := bolt.Open(dbPath, 0o600, nil)
	require.NoError(t, err)
	defer db.Close()

	store, _ := NewBoltReceiptStore(db)

	r := anchor.ExternalAnchorReceipt{
		Hash:      "test",
		Timestamp: time.Now().UTC().String(),
	}
	_, err = store.Append(r)
	require.NoError(t, err)

	// Check bucket content manually
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketExternalReceipts))
		require.NotNil(t, b)
		stats := b.Stats()
		assert.Equal(t, 1, stats.KeyN)

		c := b.Cursor()
		k, v := c.First()
		assert.NotNil(t, k)

		var loaded anchor.StoredExternalAnchorReceipt
		json.Unmarshal(v, &loaded)
		assert.Equal(t, "test", loaded.Hash)
		return nil
	})
	require.NoError(t, err)
}
