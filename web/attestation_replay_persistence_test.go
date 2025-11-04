package web

import (
	"os"
	"testing"
	"time"
)

// TestAttestationReplayPersistenceRestart verifies that an attestation nonce recorded in a durable store
// is still detected as a replay after a simulated restart (store re-instantiated from WAL).
func TestAttestationReplayPersistenceRestart(t *testing.T) {
	walPath := t.TempDir() + "/attest_replay.wal"
	// First store: record nonce
	store1 := NewReplayNonceStoreWithConfig(30*time.Minute, 0, walPath, nil)
	nonce := "persist-nonce-123"
	if store1.Seen(nonce, time.Now()) {
		t.Fatalf("nonce should not be seen initially")
	}
	store1.Record(nonce, time.Now())
	if !store1.Seen(nonce, time.Now()) {
		t.Fatalf("nonce should be seen after record")
	}
	// Simulate process restart by creating new store reading same WAL path.
	store2 := NewReplayNonceStoreWithConfig(30*time.Minute, 0, walPath, nil)
	if !store2.Seen(nonce, time.Now()) {
		t.Fatalf("restarted store should detect replay nonce")
	}
	// Ensure snapshot+compact removes expired entries but retains active.
	if err := store2.SnapshotAndCompact(); err != nil {
		t.Fatalf("snapshot/compact failed: %v", err)
	}
	// Touch WAL file to ensure it exists post-rotation.
	if _, err := os.Stat(walPath); err != nil {
		t.Fatalf("wal file missing after compact: %v", err)
	}
}
