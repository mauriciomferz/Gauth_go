package replay

import (
	"path/filepath"
	"testing"
	"time"
)

// TestWALAppendAndRecover ensures records written are recoverable after close/reopen.
func TestWALAppendAndRecover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "replay.wal")
	wal, err := NewWALStore(path)
	if err != nil { t.Fatalf("open wal: %v", err) }
	defer wal.Close()
	recs := []WALRecord{
		{Op: "Record", Key: []byte("alpha"), TS: time.Now().Unix()},
		{Op: "Record", Key: []byte("beta"), TS: time.Now().Add(1 * time.Second).Unix()},
		{Op: "Record", Key: []byte("alpha"), TS: time.Now().Add(2 * time.Second).Unix()}, // duplicate key acceptable
	}
	for i, r := range recs {
		if err := wal.AppendRecord(r); err != nil { t.Fatalf("append %d: %v", i, err) }
	}
	if err := wal.Close(); err != nil { t.Fatalf("close wal: %v", err) }
	// Reopen and recover
	wal2, err := NewWALStore(path)
	if err != nil { t.Fatalf("reopen wal: %v", err) }
	defer wal2.Close()
	var recovered []WALRecord
	apply := func(r WALRecord) error { recovered = append(recovered, r); return nil }
	applied, skipped, err := wal2.RecoverWithStats(apply)
	if err != nil { t.Fatalf("recover: %v", err) }
	if skipped != 0 { t.Fatalf("expected 0 skipped, got %d", skipped) }
	if applied != len(recs) { t.Fatalf("applied mismatch: want %d got %d", len(recs), applied) }
	if len(recovered) != len(recs) { t.Fatalf("recovered slice mismatch") }
	// Order and content assertions
	for i := range recs {
		if string(recovered[i].Key) != string(recs[i].Key) || recovered[i].Op != recs[i].Op {
			// TS may differ slightly due to unix seconds reading but should match exact int64
			if recovered[i].TS != recs[i].TS { t.Fatalf("record %d mismatch", i) }
		}
	}
}

// TestWALRecoverSkipsMalformed verifies malformed JSON lines are skipped without aborting recovery.
func TestWALRecoverSkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "replay.wal")
	wal, err := NewWALStore(path)
	if err != nil { t.Fatalf("open wal: %v", err) }
	// Append one good record
	if err := wal.AppendRecord(WALRecord{Op: "Record", Key: []byte("good"), TS: time.Now().Unix()}); err != nil { t.Fatalf("append good: %v", err) }
	// Manually inject a malformed line
	if _, err := wal.file.Write([]byte("{this is not json}\n")); err != nil { t.Fatalf("inject malformed: %v", err) }
	// Append another good record
	if err := wal.AppendRecord(WALRecord{Op: "Record", Key: []byte("good2"), TS: time.Now().Unix()}); err != nil { t.Fatalf("append good2: %v", err) }
	if err := wal.Close(); err != nil { t.Fatalf("close wal: %v", err) }
	wal2, err := NewWALStore(path)
	if err != nil { t.Fatalf("reopen: %v", err) }
	defer wal2.Close()
	var keys []string
	apply := func(r WALRecord) error { keys = append(keys, string(r.Key)); return nil }
	applied, skipped, err := wal2.RecoverWithStats(apply)
	if err != nil { t.Fatalf("recover: %v", err) }
	if applied != 2 { t.Fatalf("expected 2 applied got %d", applied) }
	if skipped != 1 { t.Fatalf("expected 1 skipped got %d", skipped) }
	if len(keys) != 2 || keys[0] != "good" || keys[1] != "good2" { t.Fatalf("unexpected keys %v", keys) }
}

// TestNewReplayNonceStoreWithConfigRecovery ensures ReplayNonceStore integrates WAL recovery.
// Integration with ReplayNonceStore tested in package web; here we focus solely on WAL mechanics.
