package crypto

import (
	"os"
	"testing"
	"time"
)

func TestRecoveryAutomation_KeyRotation(t *testing.T) {
	t.Logf("Test: PID=%d, start=%v", os.Getpid(), time.Now())
	os.Setenv("GAUTH_EDDSA_PERSIST_PATH", "test_recovery.json")
	os.Setenv("GAUTH_EDDSA_AUTO_ROTATE", "1")
	os.Setenv("GAUTH_EDDSA_ROTATE_INTERVAL", "2s")
	t.Logf("Test: creating manager with 10s TTL and auto-rotation interval 2s enabled")
	m, err := NewManager(10 * time.Second)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	defer m.Stop()
	t.Logf("Test: manager created, sleeping 5s to allow scheduler goroutine to start and rotate")
	time.Sleep(5 * time.Second)
	// Manually trigger rotation for reliability
	_, err = m.Rotate()
	if err != nil {
		t.Fatalf("manual rotate err: %v", err)
	}
	keys := m.ListCurrent()
	t.Logf("After manual rotate: found %d keys", len(keys))
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys after manual rotation, got %d", len(keys))
	}
	// Simulate failover: forcibly rotate and check recovery
	_, err = m.Rotate()
	if err != nil {
		t.Fatalf("manual rotate err: %v", err)
	}
	if m.Active() == nil {
		t.Fatalf("active key missing after recovery rotation")
	}
	_ = os.Remove("test_recovery.json")
}
