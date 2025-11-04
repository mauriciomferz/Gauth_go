package ledger

import (
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"
)

// IndexEntry represents an indexed record in BoltDB.
type IndexEntry struct {
	Key       []byte
	Timestamp int64
	TTL       int64 // Time-to-live in seconds
}

// BoltDBIndexer provides indexing and TTL pruning for BoltDB.
type BoltDBIndexer struct {
	DB *bbolt.DB
}

func NewBoltDBIndexer(db *bbolt.DB) *BoltDBIndexer {
	return &BoltDBIndexer{DB: db}
}

// AddIndexEntry adds an index entry with optional TTL.
func (idx *BoltDBIndexer) AddIndexEntry(bucket string, key []byte, ttlSeconds int64) error {
	return idx.DB.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		entry := IndexEntry{Key: key, Timestamp: time.Now().Unix(), TTL: ttlSeconds}
		return b.Put(key, encodeIndexEntry(entry))
	})
}

// PruneExpiredEntries removes entries past their TTL.
func (idx *BoltDBIndexer) PruneExpiredEntries(bucket string) error {
	return idx.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			entry := decodeIndexEntry(v)
			if entry.TTL > 0 && time.Now().Unix()-entry.Timestamp > entry.TTL {
				if err := b.Delete(k); err != nil {
					// Log or handle delete error as appropriate
					continue
				}
			}
		}
		return nil
	})
}

// encodeIndexEntry and decodeIndexEntry are stubs for serialization.
func encodeIndexEntry(entry IndexEntry) []byte {
	// JSON encoding for IndexEntry
	data, _ := json.Marshal(entry)
	return data
}

func decodeIndexEntry(data []byte) IndexEntry {
	var entry IndexEntry
	_ = json.Unmarshal(data, &entry)
	return entry
}
