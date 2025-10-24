package web

import (
	"os"
	"testing"
	"time"
)

func TestReplayNonceStore_WALIntegration(t *testing.T) {
	walPath := "test_replay_wal.log"
	_ = os.Remove(walPath)
	os.Setenv("GAUTH_REPLAY_WAL", walPath)
	store := NewReplayNonceStore(2 * time.Second)
	store.Record("nonce1", time.Now())
	store.Record("nonce2", time.Now())
	if !store.Seen("nonce1", time.Now()) {
		t.Fatalf("nonce1 should be seen after record")
	}
	if !store.Seen("nonce2", time.Now()) {
		t.Fatalf("nonce2 should be seen after record")
	}
	// Simulate restart and recovery
	os.Setenv("GAUTH_REPLAY_WAL", walPath)
	store2 := NewReplayNonceStore(2 * time.Second)
	if !store2.Seen("nonce1", time.Now()) {
		t.Fatalf("nonce1 should be recovered from WAL")
	}
	if !store2.Seen("nonce2", time.Now()) {
		t.Fatalf("nonce2 should be recovered from WAL")
	}
	_ = os.Remove(walPath)
}
