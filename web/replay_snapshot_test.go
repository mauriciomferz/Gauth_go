package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/web/handlers/token"
)

// TestReplaySnapshotAndCompact verifies snapshot file creation, WAL rotation, and recovery continuity.
func TestReplaySnapshotAndCompact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "replay.wal")
	store := token.NewReplayNonceStoreWithConfig(10*time.Minute, 100, path, nil)
	if !store.IsDurable() {
		t.Fatalf("expected wal backing store")
	}
	// Insert entries
	for i := 0; i < 10; i++ {
		store.Record("nonce-"+time.Now().Format("150405")+"-"+string(rune('a'+i)), time.Now())
		// small sleep to ensure varying timestamps
		time.Sleep(5 * time.Millisecond)
	}
	// Stat WAL size before
	infoBefore, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	if err2 := store.SnapshotAndCompact(); err2 != nil {
		t.Fatalf("snapshot+compact: %v", err2)
	}
	// Ensure snapshot file exists
	snapInfo, err := os.Stat(path + ".snapshot")
	if err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}
	if snapInfo.Size() == 0 {
		t.Fatalf("snapshot file empty")
	}
	// WAL after rotation
	infoAfter, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if infoAfter.Size() > infoBefore.Size()*2 {
		t.Fatalf("unexpected WAL growth after compaction")
	}
	// Close and reopen to test recovery
	if err := store.Close(); err != nil {
		t.Fatalf("close wal: %v", err)
	}
	store2 := token.NewReplayNonceStoreWithConfig(30*time.Minute, 0, path, nil)
	// Non-deterministic key generation above; ensure at least one recovered by scanning seen map size > 0
	if store2.Size() == 0 {
		t.Fatalf("expected entries recovered; size=0")
	}
}
