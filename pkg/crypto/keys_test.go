package crypto

import (
	"os"
	"testing"
	"time"
)

func TestKeyArchivalAndRetrieval(t *testing.T) {
	// Create temp file for persistence
	f, err := os.CreateTemp("", "gauth-keys-*.json")
	if err != nil {
		t.Fatal(err)
	}
	persistPath := f.Name()
	f.Close()
	defer os.Remove(persistPath)

	t.Setenv("AGENTAUTH_EDDSA_PERSIST_PATH", persistPath)

	// 1. Init Manager with short TTL
	ttl := 200 * time.Millisecond
	m, err := NewManager(ttl)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	firstKey := m.Active()
	if firstKey == nil {
		t.Fatal("expected active key")
	}
	t.Logf("First Key: %s (Expires: %v)", firstKey.ID, firstKey.ExpiresAt)

	// 2. Wait for expiration
	time.Sleep(ttl + 50*time.Millisecond)

	// 3. Rotate (First key moves to History, but since it's expired, logic might prune/archive immediately or on next)
	// Current logic: rotateLocked appends prev (firstKey) to history. Then prune checks history.
	// If firstKey is expired, it moves to archived. active is new.
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("Rotate 1: %v", err)
	}

	secondKey := m.Active()
	t.Logf("Second Key: %s", secondKey.ID)

	// Verify first key is NOT in Current list (active+history)
	current := m.ListCurrent()
	for _, k := range current {
		if k.ID == firstKey.ID {
			t.Errorf("expired key %s should not be in ListCurrent", k.ID)
		}
	}

	// Verify first key IS retrievable by ID (from archive)
	found := m.FindByID(firstKey.ID)
	if found == nil {
		t.Errorf("expired key %s not found via FindByID (should be in archive)", firstKey.ID)
	}

	// 4. Persistence Check: Load a new manager from the same file
	// Wait a bit to ensure second key also potentially expires or just to separate state
	time.Sleep(10 * time.Millisecond)

	m2, err := NewManager(ttl)
	if err != nil {
		t.Fatalf("NewManager (reload): %v", err)
	}

	// Verify first key is still retrievable from loaded archive
	found2 := m2.FindByID(firstKey.ID)
	if found2 == nil {
		t.Errorf("reloaded manager failed to find archived key %s", firstKey.ID)
	}

	// ListCurrent on m2 should distinct active from archive
	current2 := m2.ListCurrent()
	for _, k := range current2 {
		if k.ID == firstKey.ID {
			t.Errorf("reloaded ListCurrent contained archived key %s", k.ID)
		}
	}
}
