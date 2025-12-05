package crypto

import (
	"os"
	"testing"
	"time"
)

func TestSchedulerGoroutineLaunch(t *testing.T) {
	os.Setenv("GAUTH_EDDSA_AUTO_ROTATE", "1")
	t.Logf("Test: creating manager with 10s TTL and auto-rotation enabled")
	m, err := NewManager(10 * time.Second)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	defer m.Stop()
	t.Logf("Test: manager created, sleeping 3s to allow scheduler goroutine to start and rotate")
	time.Sleep(3 * time.Second)
	keys := m.ListCurrent()
	t.Logf("Test: after sleep, found %d keys", len(keys))
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys after scheduler, got %d", len(keys))
	}
}
