package web

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/web/handlers/token"
)

func TestReplayNonceStore_WALIntegration(t *testing.T) {
	walPath := "test_replay_wal.log"
	_ = os.Remove(walPath)
	t.Setenv("GAUTH_REPLAY_WAL", walPath)
	store := token.NewReplayNonceStore(2 * time.Second)
	store.Record("nonce1", time.Now())
	store.Record("nonce2", time.Now())
	if !store.Seen("nonce1", time.Now()) {
		t.Fatalf("nonce1 should be seen after record")
	}
	if !store.Seen("nonce2", time.Now()) {
		t.Fatalf("nonce2 should be seen after record")
	}
	// Simulate restart and recovery
	t.Setenv("GAUTH_REPLAY_WAL", walPath)
	store2 := token.NewReplayNonceStore(2 * time.Second)
	if !store2.Seen("nonce1", time.Now()) {
		t.Fatalf("nonce1 should be recovered from WAL")
	}
	if !store2.Seen("nonce2", time.Now()) {
		t.Fatalf("nonce2 should be recovered from WAL")
	}
	_ = os.Remove(walPath)
}
