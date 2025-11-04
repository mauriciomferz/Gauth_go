package web

import (
	replaypkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/replay"
	"os"
	"testing"
	"time"
)

// TestAttestationExternalRedisBackend ensures Redis backend detects replay after Record.
func TestAttestationExternalRedisBackend(t *testing.T) {
	if os.Getenv("GAUTH_ATTEST_REPLAY_BACKEND_TEST_SKIP") == "1" {
		t.Skip("skipped via env")
	}
	addr := os.Getenv("GAUTH_ATTEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	backend, err := replaypkg.NewRedisReplayBackend(addr, 5*time.Minute)
	if err != nil {
		t.Skipf("redis unavailable: %v", err)
	}
	defer backend.Close()
	nonce := "redis-nonce-test-1"
	seen, _ := backend.Seen(nonce)
	if seen {
		t.Fatalf("expected nonce unseen initially")
	}
	if err := backend.Record(nonce); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	seen2, _ := backend.Seen(nonce)
	if !seen2 {
		t.Fatalf("expected nonce seen after record")
	}
}
