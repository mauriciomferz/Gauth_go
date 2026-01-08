package crypto

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSchedulerGoroutineLaunch(t *testing.T) {
	// Ensure this test does not depend on global env state or write into the repo.
	t.Setenv("AGENTAUTH_EDDSA_PERSIST_PATH", filepath.Join(t.TempDir(), "eddsa_keys.json"))
	t.Setenv("AGENTAUTH_EDDSA_AUTO_ROTATE", "1")
	t.Setenv("AGENTAUTH_EDDSA_ROTATE_INTERVAL", "500ms")

	t.Logf("Test: creating manager with 10s TTL and auto-rotation enabled")
	m, err := NewManager(10 * time.Second)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	defer m.Stop()

	// Wait long enough for at least one scheduler tick.
	t.Logf("Test: manager created, sleeping to allow scheduler goroutine to rotate")
	time.Sleep(2 * time.Second)

	keys := m.ListCurrent()
	t.Logf("Test: after sleep, found %d keys", len(keys))
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys after scheduler, got %d", len(keys))
	}
}
