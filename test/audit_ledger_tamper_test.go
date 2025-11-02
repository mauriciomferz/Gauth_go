package test

import (
    "encoding/json"
    "path/filepath"
    "testing"

    bolt "go.etcd.io/bbolt"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/examples/ai_capability_demo/ledger"
)

// TestAuditLedgerTamper verifies that modifying a stored entry breaks chain verification.
func TestAuditLedgerTamper(t *testing.T) {
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "audit.db")
    al, err := ledger.NewAuditLedger(dbPath)
    if err != nil { t.Fatalf("init ledger: %v", err) }
    // Append two entries
    if _, err := al.Append("poa_issue", "actorA", "poa-1", map[string]any{"scope": []string{"x"}}); err != nil { t.Fatalf("append1: %v", err) }
    if _, err := al.Append("poa_sign", "actorB", "poa-1", map[string]any{"signatures": 1}); err != nil { t.Fatalf("append2: %v", err) }
    if mismatch, err := al.VerifyChain(); err != nil || mismatch != -1 { t.Fatalf("expected clean chain got mismatch=%d err=%v", mismatch, err) }
    al.Close()
    // Tamper second entry bytes directly via Bolt
    db, err := boltOpenRW(dbPath)
    if err != nil { t.Fatalf("reopen rw: %v", err) }
    err = db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("audit_ledger"))
        if b == nil { return nil }
        v := b.Get(itob(1)) // id=1 second entry
        if v == nil { return nil }
        // Decode JSON, alter prev_hash
        var entry map[string]any
        if err := json.Unmarshal(v, &entry); err != nil { return err }
        entry["prev_hash"] = "deadbeef" // invalid linkage
        tampered, _ := json.Marshal(entry)
        return b.Put(itob(1), tampered)
    })
    db.Close()
    if err != nil { t.Fatalf("tamper update: %v", err) }
    al2, err := ledger.NewAuditLedger(dbPath)
    if err != nil { t.Fatalf("rewrap: %v", err) }
    mismatch, verr := al2.VerifyChain()
    al2.Close()
    if verr == nil || mismatch == -1 { t.Fatalf("expected tamper detection, got mismatch=%d err=%v", mismatch, verr) }
}

// Helpers
func boltOpenRW(path string) (*bolt.DB, error) { return bolt.Open(path, 0600, nil) }
// itob replicates the 8-byte big-endian conversion used by ledger internals.
func itob(v int) []byte {
    b := make([]byte, 8)
    for i := 7; i >= 0; i-- { b[i] = byte(v & 0xFF); v >>= 8 }
    return b
}
