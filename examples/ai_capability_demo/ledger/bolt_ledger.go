package ledger

// Experimental BoltDB-backed ledger (append-only) used when GAUTH_AI_DEMO_LEDGER_DB_PATH is set.
// For MVP we store entries sequentially in a single bucket and recompute roots in-memory on demand.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var boltBucket = []byte("ledger_entries")

// BoltLedger persists entries; roots kept in memory for simplicity.
type BoltLedger struct {
	mu    sync.RWMutex
	db    *bolt.DB
	roots []string
}

// NewBoltLedger opens (or creates) a BoltDB file at path.
func NewBoltLedger(path string) (*BoltLedger, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	// ensure bucket
	if err := db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(boltBucket)
		return e
	}); err != nil {
		db.Close()
		return nil, err
	}
	// preload roots by iterating entries (cost acceptable for demo small size)
	bl := &BoltLedger{db: db, roots: []string{}}
	bl.recomputeRootsLocked()
	return bl, nil
}

// Close closes underlying DB.
func (bl *BoltLedger) Close() error { return bl.db.Close() }

// Append stores an entry and updates root history.
func (bl *BoltLedger) Append(entryType, payload, digest string) (int, string, error) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	var id int
	if digest == "" {
		ph := sha256.Sum256([]byte(payload))
		digest = hex.EncodeToString(ph[:])
	}
	err := bl.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(boltBucket)
		if b == nil {
			return errors.New("bucket_missing")
		}
		id = b.Stats().KeyN
		entry := Entry{ID: id, Type: entryType, Digest: digest, Payload: payload}
		raw, _ := json.Marshal(entry)
		idKey := itob(id)
		return b.Put(idKey, raw)
	})
	if err != nil {
		return 0, "", err
	}
	root := bl.computeRootLocked()
	bl.roots = append(bl.roots, root)
	return id, root, nil
}

// Get returns entry by ID.
func (bl *BoltLedger) Get(id int) (Entry, bool) {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	var out Entry
	err := bl.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(boltBucket)
		if b == nil {
			return errors.New("bucket_missing")
		}
		v := b.Get(itob(id))
		if v == nil {
			return errors.New("not_found")
		}
		return json.Unmarshal(v, &out)
	})
	return out, err == nil
}

// Size returns entry count.
func (bl *BoltLedger) Size() int {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	var n int
	bl.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(boltBucket)
		if b != nil {
			n = b.Stats().KeyN
		}
		return nil
	})
	return n
}

// Proof builds an inclusion proof similar to in-memory ledger by loading all entries (demo scale acceptable).
func (bl *BoltLedger) Proof(id int) (Proof, error) {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	entries, err := bl.loadAllLocked()
	if err != nil {
		return Proof{}, err
	}
	if id < 0 || id >= len(entries) {
		return Proof{}, fmt.Errorf("entry_not_found")
	}
	layer := make([]string, len(entries))
	for i, e := range entries {
		layer[i] = leafHash(e.Digest)
	}
	path := []string{}
	orient := []string{}
	idx := id
	work := layer
	for len(work) > 1 {
		var sibIdx int
		if idx%2 == 0 {
			sibIdx = idx + 1
			if sibIdx >= len(work) {
				sibIdx = idx
			}
			orient = append(orient, "L")
		} else {
			sibIdx = idx - 1
			orient = append(orient, "R")
		}
		path = append(path, work[sibIdx])
		var next []string
		for i := 0; i < len(work); i += 2 {
			if i+1 == len(work) {
				next = append(next, hash(work[i], work[i]))
			} else {
				next = append(next, hash(work[i], work[i+1]))
			}
		}
		idx = idx / 2
		work = next
	}
	return Proof{EntryID: id, Digest: entries[id].Digest, Siblings: path, Orientations: orient, Root: work[0]}, nil
}

// HistoricalRoots returns copy of emitted roots.
func (bl *BoltLedger) HistoricalRoots() []string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	out := make([]string, len(bl.roots))
	copy(out, bl.roots)
	return out
}

// loadAllLocked reads all entries under lock.
func (bl *BoltLedger) loadAllLocked() ([]Entry, error) {
	var list []Entry
	err := bl.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(boltBucket)
		if b == nil {
			return errors.New("bucket_missing")
		}
		return b.ForEach(func(k, v []byte) error {
			var e Entry
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			list = append(list, e)
			return nil
		})
	})
	return list, err
}

// recomputeRootsLocked rebuilds roots history from scratch.
func (bl *BoltLedger) recomputeRootsLocked() {
	entries, err := bl.loadAllLocked()
	if err != nil {
		return
	}
	bl.roots = []string{}
	for i := range entries { // incremental emission semantics
		partial := entries[:i+1]
		layer := make([]string, len(partial))
		for j, e := range partial {
			layer[j] = leafHash(e.Digest)
		}
		for len(layer) > 1 {
			var next []string
			for k := 0; k < len(layer); k += 2 {
				if k+1 == len(layer) {
					next = append(next, hash(layer[k], layer[k]))
				} else {
					next = append(next, hash(layer[k], layer[k+1]))
				}
			}
			layer = next
		}
		bl.roots = append(bl.roots, layer[0])
	}
}

// computeRootLocked computes root for current full set without updating history.
func (bl *BoltLedger) computeRootLocked() string {
	entries, err := bl.loadAllLocked()
	if err != nil || len(entries) == 0 {
		return ""
	}
	layer := make([]string, len(entries))
	for i, e := range entries {
		layer[i] = leafHash(e.Digest)
	}
	for len(layer) > 1 {
		var next []string
		for i := 0; i < len(layer); i += 2 {
			if i+1 == len(layer) {
				next = append(next, hash(layer[i], layer[i]))
			} else {
				next = append(next, hash(layer[i], layer[i+1]))
			}
		}
		layer = next
	}
	return layer[0]
}

// itob converts int to big-endian 8-byte key (simplistic ordering by ID).
func itob(v int) []byte {
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = byte(v & 0xFF)
		v >>= 8
	}
	return b
}
