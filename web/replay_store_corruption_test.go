package web

import (
	"fmt"
	"os"
	"testing"
	"time"

	metricsPkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestReplayNonceStore_CorruptionRecovery ensures malformed WAL lines are skipped and counted.
func TestReplayNonceStore_CorruptionRecovery(t *testing.T) {
    walPath := "test_replay_wal_corrupt.log"
    _ = os.Remove(walPath)
    // Create a WAL file with: valid, corrupt, valid lines.
    f, err := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
    if err != nil {
        t.Fatalf("unable to create wal file: %v", err)
    }
    now := time.Now().Unix()
    // Valid record 1
    _, _ = fmt.Fprintf(f, `{"Op":"Record","Key":"bm9uY2VfY29yMV9rZXk=","Value":null,"TS":%d}`+"\n", now)
    // Corrupt line (truncated JSON)
    _, _ = fmt.Fprintln(f, `{"Op":"Record","Key":"bmFsZm9ybWVk","Value":null,"TS":1700000001`)
    // Valid record 2
    _, _ = fmt.Fprintf(f, `{"Op":"Record","Key":"bm9uY2VfY29yMl9rZXk=","Value":null,"TS":%d}`+"\n", now+1)
    _ = f.Close()

    os.Setenv("GAUTH_REPLAY_WAL", walPath)
    memMetrics := metricsPkg.NewMemory()
    store := NewReplayNonceStoreWithMetrics(10*time.Minute, memMetrics)

    // Base64 keys decoded by WAL recovery will be stored as raw bytes; our test used plain base64 of strings.
    // We inserted keys: nonce_cor1_key and nonce_cor2_key (encoded). Validate they are seen.
    if !store.Seen("nonce_cor1_key", time.Now()) {
        t.Fatalf("expected first valid key recovered")
    }
    if !store.Seen("nonce_cor2_key", time.Now()) {
        t.Fatalf("expected second valid key recovered")
    }

    // Snapshot metrics and ensure at least one replay store error was recorded for the corrupt line.
    snap := memMetrics.SnapshotEx()
    if snap.ReplayStoreErrors == 0 {
        t.Fatalf("expected replay store errors > 0 for corrupt line; got %d", snap.ReplayStoreErrors)
    }

    _ = os.Remove(walPath)
}
