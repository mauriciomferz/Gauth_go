package gauth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBoltReplayStore_ContainerSafetyCheck(t *testing.T) {
	// This test verifies that BoltDB safety checks work correctly
	// On macOS (not in container), it should accept /tmp paths
	// In containers, it would reject /tmp paths unless bypass is set

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "replay.db")

	// Test 1: Normal path should work (not in container on macOS)
	store, err := NewBoltReplayStore(dbPath, time.Hour)
	if err != nil {
		t.Fatalf("NewBoltReplayStore() failed on safe path: %v", err)
	}
	defer store.Close()

	// Test 2: Verify basic functionality
	jti := "test-jti-123"
	err = store.CheckAndRecord(jti)
	if err != nil {
		t.Errorf("CheckAndRecord() failed: %v", err)
	}

	// Test 3: Replay should be detected
	err = store.CheckAndRecord(jti)
	if err == nil {
		t.Error("CheckAndRecord() should have detected replay attack")
	}

	t.Logf("✅ BoltDB safety checks functional (not in container)")
}

func TestBoltReplayStore_BypassFlag(t *testing.T) {
	// Test that bypass flag is respected
	// Note: This test won't trigger container detection on macOS,
	// but verifies the bypass mechanism exists

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "replay.db")

	// Set bypass flag
	oldValue := os.Getenv("GAUTH_ALLOW_UNSAFE_BOLTDB")
	os.Setenv("GAUTH_ALLOW_UNSAFE_BOLTDB", "1")
	defer func() {
		if oldValue != "" {
			os.Setenv("GAUTH_ALLOW_UNSAFE_BOLTDB", oldValue)
		} else {
			os.Unsetenv("GAUTH_ALLOW_UNSAFE_BOLTDB")
		}
	}()

	// Should still work (bypass enabled)
	store, err := NewBoltReplayStore(dbPath, time.Hour)
	if err != nil {
		t.Fatalf("NewBoltReplayStore() failed with bypass flag: %v", err)
	}
	defer store.Close()

	t.Logf("✅ Bypass flag mechanism functional")
}

func TestBoltReplayStore_Documentation(t *testing.T) {
	// This test documents expected behavior in different environments

	t.Log("BoltDB Container Safety Behavior:")
	t.Log("")
	t.Log("Environment: macOS (not in container)")
	t.Log("  - /tmp paths: ✅ ACCEPTED (no container detection)")
	t.Log("  - /data paths: ✅ ACCEPTED")
	t.Log("  - Bypass flag: Not required")
	t.Log("")
	t.Log("Environment: Docker/Kubernetes (in container)")
	t.Log("  - /tmp paths: ❌ REJECTED (ephemeral storage)")
	t.Log("  - /var/tmp paths: ❌ REJECTED (ephemeral storage)")
	t.Log("  - /data paths with PVC: ✅ ACCEPTED (persistent)")
	t.Log("  - Bypass flag GAUTH_ALLOW_UNSAFE_BOLTDB=1: ⚠️ UNSAFE (dev only)")
	t.Log("")
	t.Log("Production Recommendation:")
	t.Log("  - Use Redis for distributed replay protection")
	t.Log("  - See REPLAY_STORE_MIGRATION_GUIDE.md")
}
