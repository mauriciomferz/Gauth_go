package ledger

// Audit hash-chain ledger (separate from Merkle ledger) provides sequential tamper-evident chaining.
// Each entry links to previous via entry_hash = SHA256(prev_hash || canonical_entry_json).

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "time"
    "sync"
    bolt "go.etcd.io/bbolt"
)

var auditBucket = []byte("audit_ledger")

// AuditEntry is a single hash-chain record.
type AuditEntry struct {
    ID        int               `json:"id"`
    At        time.Time         `json:"at"`
    Type      string            `json:"type"`
    POAID     string            `json:"poa_id,omitempty"`
    Actor     string            `json:"actor,omitempty"`
    PrevHash  string            `json:"prev_hash"`
    EntryHash string            `json:"entry_hash"`
    Metadata  map[string]any    `json:"metadata,omitempty"`
}

// canonicalBytes returns deterministic JSON for hashing (excluding EntryHash itself).
func (ae *AuditEntry) canonicalBytes() []byte {
    // We build minimal stable map (exclude EntryHash, include PrevHash for linkage).
    obj := map[string]any{
        "id": ae.ID,
        "at": ae.At.UTC().Format(time.RFC3339Nano),
        "type": ae.Type,
        "poa_id": ae.POAID,
        "actor": ae.Actor,
        "prev_hash": ae.PrevHash,
        "metadata": ae.Metadata,
    }
    b, _ := json.Marshal(obj)
    return b
}

// computeHash derives EntryHash given PrevHash + canonical JSON.
func (ae *AuditEntry) computeHash() string {
    h := sha256.New()
    h.Write([]byte(ae.PrevHash))
    h.Write(ae.canonicalBytes())
    return hex.EncodeToString(h.Sum(nil))
}

// AuditLedger persistent implementation using BoltDB.
type AuditLedger struct {
    mu  sync.RWMutex
    db  *bolt.DB
    size int
    head string // latest entry hash
}

// NewAuditLedger opens or creates the BoltDB file & bucket.
func NewAuditLedger(path string) (*AuditLedger, error) {
    db, err := bolt.Open(path, 0600, nil)
    if err != nil { return nil, err }
    if err := db.Update(func(tx *bolt.Tx) error { _, e := tx.CreateBucketIfNotExists(auditBucket); return e }); err != nil { db.Close(); return nil, err }
    al := &AuditLedger{db: db}
    al.rebuildState()
    return al, nil
}

// rebuildState loads last ID + head hash.
func (al *AuditLedger) rebuildState() {
    al.mu.Lock(); defer al.mu.Unlock()
    var last AuditEntry
    _ = al.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(auditBucket); if b == nil { return errors.New("bucket_missing") }
        c := b.Cursor()
        k, v := c.Last()
        if k == nil { return nil }
        _ = json.Unmarshal(v, &last)
        return nil
    })
    al.head = last.EntryHash
    al.size = last.ID + 1
}

// Append creates a new audit entry.
func (al *AuditLedger) Append(entryType string, actor string, poaID string, metadata map[string]any) (AuditEntry, error) {
    al.mu.Lock(); defer al.mu.Unlock()
    id := al.size
    ae := AuditEntry{ID: id, At: time.Now().UTC(), Type: entryType, POAID: poaID, Actor: actor, PrevHash: al.head, Metadata: metadata}
    ae.EntryHash = ae.computeHash()
    raw, _ := json.Marshal(ae)
    err := al.db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket(auditBucket); if b == nil { return errors.New("bucket_missing") }
        key := itob(id)
        return b.Put(key, raw)
    })
    if err != nil { return AuditEntry{}, err }
    al.head = ae.EntryHash
    al.size++
    return ae, nil
}

// Get returns entry by ID.
func (al *AuditLedger) Get(id int) (AuditEntry, bool) {
    al.mu.RLock(); defer al.mu.RUnlock()
    var ae AuditEntry
    err := al.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(auditBucket); if b == nil { return errors.New("bucket_missing") }
        v := b.Get(itob(id)); if v == nil { return errors.New("not_found") }
        return json.Unmarshal(v, &ae)
    })
    return ae, err == nil
}

// List returns last N entries (limit applied from tail).
func (al *AuditLedger) List(limit int) ([]AuditEntry, error) {
    al.mu.RLock(); defer al.mu.RUnlock()
    if limit <= 0 || limit > 10000 { limit = 100 } // safety bounds
    start := 0
    if al.size > limit { start = al.size - limit }
    var list []AuditEntry
    err := al.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(auditBucket); if b == nil { return errors.New("bucket_missing") }
        c := b.Cursor()
        for k, v := c.Seek(itob(start)); k != nil; k, v = c.Next() {
            var ae AuditEntry
            if err := json.Unmarshal(v, &ae); err != nil { return err }
            list = append(list, ae)
        }
        return nil
    })
    return list, err
}

// VerifyChain scans entries in order and recomputes each hash to detect tampering; returns first mismatch index or -1.
func (al *AuditLedger) VerifyChain() (int, error) {
    al.mu.RLock(); defer al.mu.RUnlock()
    var idxMismatch = -1
    err := al.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(auditBucket); if b == nil { return errors.New("bucket_missing") }
        prev := ""
        c := b.Cursor()
        for k, v := c.First(); k != nil; k, v = c.Next() {
            var ae AuditEntry
            if err := json.Unmarshal(v, &ae); err != nil { return err }
            expectedPrev := prev
            if ae.PrevHash != expectedPrev { idxMismatch = ae.ID; return errors.New("prev_hash_mismatch") }
            // recompute hash
            tmp := AuditEntry{ID: ae.ID, At: ae.At, Type: ae.Type, POAID: ae.POAID, Actor: ae.Actor, PrevHash: ae.PrevHash, Metadata: ae.Metadata}
            calc := tmp.computeHash()
            if calc != ae.EntryHash { idxMismatch = ae.ID; return errors.New("entry_hash_mismatch") }
            prev = ae.EntryHash
        }
        return nil
    })
    if err != nil { return idxMismatch, err }
    return -1, nil
}

// HeadHash returns latest entry hash.
func (al *AuditLedger) HeadHash() string { al.mu.RLock(); defer al.mu.RUnlock(); return al.head }

// Size returns number of entries.
func (al *AuditLedger) Size() int { al.mu.RLock(); defer al.mu.RUnlock(); return al.size }

// Close closes DB.
func (al *AuditLedger) Close() error { return al.db.Close() }
