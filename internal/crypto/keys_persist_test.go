package crypto

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerPersistenceCycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eddsa_keys.json")
	os.Setenv("GAUTH_EDDSA_PERSIST_PATH", path)
	defer os.Unsetenv("GAUTH_EDDSA_PERSIST_PATH")
	m, err := NewManager(2 * time.Hour)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	active := m.Active()
	if active == nil {
		t.Fatalf("active key expected")
	}
	// Force rotation to produce history key
	if _, rotErr := m.Rotate(); rotErr != nil {
		t.Fatalf("rotate: %v", rotErr)
	}
	if len(m.ListCurrent()) < 2 {
		t.Fatalf("expected at least 2 keys after rotation")
	}
	// Reload via new manager instance
	m2, err := NewManager(2 * time.Hour) // will attempt load before fresh rotation
	if err != nil {
		t.Fatalf("new manager 2: %v", err)
	}
	if m2.Active() == nil {
		t.Fatalf("expected active key after load")
	}
	// Ensure original active (or rotated) kid present in loaded set
	found := false
	for _, k := range m2.ListCurrent() {
		if k.ID == active.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected original active kid %s present after load", active.ID)
	}
}
