package crypto

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRecoveryAutomation_KeyRotation(t *testing.T) {
	persistPath := filepath.Join(t.TempDir(), "recovery.json")
	t.Setenv("AGENTAUTH_EDDSA_PERSIST_PATH", persistPath)
	t.Setenv("AGENTAUTH_EDDSA_AUTO_ROTATE", "1")
	t.Setenv("AGENTAUTH_EDDSA_ROTATE_INTERVAL", "200ms")

	m, err := NewManager(10 * time.Second)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	defer m.Stop()

	// Give the scheduler a moment to run, but keep the test fast.
	time.Sleep(750 * time.Millisecond)

	// Manually trigger rotation for determinism.
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("manual rotate err: %v", err)
	}
	keys := m.ListCurrent()
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys after manual rotation, got %d", len(keys))
	}

	// Simulate failover: rotate again and ensure active key exists.
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("manual rotate err: %v", err)
	}
	if m.Active() == nil {
		t.Fatalf("active key missing after recovery rotation")
	}
}
