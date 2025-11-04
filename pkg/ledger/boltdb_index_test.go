package ledger

import (
	"os"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func TestBoltDBIndexer_AddAndPruneIndexEntry(t *testing.T) {
	file := "test_boltdb_index.db"
	defer os.Remove(file)
	
db, err := bbolt.Open(file, 0600, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	indexer := NewBoltDBIndexer(db)
	bucket := "testbucket"
	key := []byte("key1")
	// Add entry with short TTL
	if err := indexer.AddIndexEntry(bucket, key, 1); err != nil {
		t.Fatalf("add index: %v", err)
	}
	// Wait for TTL to expire
	time.Sleep(2 * time.Second)
	if err := indexer.PruneExpiredEntries(bucket); err != nil {
		t.Fatalf("prune: %v", err)
	}
	// Check that entry is deleted
	if err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			t.Fatalf("bucket missing")
		}
		v := b.Get(key)
		if v != nil {
			t.Fatalf("expected entry to be pruned, got %v", v)
		}
		return nil
	}); err != nil {
		t.Fatalf("view error: %v", err)
	}
}
