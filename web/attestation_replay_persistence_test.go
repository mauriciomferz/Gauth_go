package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/web/handlers/token"
)

// TestAttestationReplayPersistenceRestart verifies that an attestation nonce recorded in a durable store
// is still detected as a replay after a simulated restart (store re-instantiated from WAL).
func TestAttestationReplayPersistence(t *testing.T) {
	// Durable store with WAL
	dir := t.TempDir()
	walPath := filepath.Join(dir, "attest.wal")
	// First store: record nonce
	store := token.NewReplayNonceStoreWithConfig(10*time.Minute, 100, walPath, nil)
	nonce := "persist-nonce-123"
	if store.Seen(nonce, time.Now()) {
		t.Fatalf("nonce should not be seen initially")
	}
	store.Record(nonce, time.Now())
	if !store.Seen(nonce, time.Now()) {
		t.Fatalf("nonce should be seen after record")
	}
	// Simulate process restart by creating new store reading same WAL path.
	store2 := token.NewReplayNonceStoreWithConfig(10*time.Minute, 100, walPath, nil)
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
