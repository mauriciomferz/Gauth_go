package anchor

import "testing"

const testHashValue = "abc123"

func TestMemoryAnchorBasic(t *testing.T) {
	m := NewMemoryAnchor()
	if m.TotalAnchors() != 0 {
		t.Fatalf("expected 0 anchors initially")
	}
	rec, err := m.Anchor(testHashValue)
	if err != nil {
		t.Fatalf("anchor error: %v", err)
	}
	if rec.Hash != testHashValue {
		t.Fatalf("hash mismatch")
	}
	if m.TotalAnchors() != 1 {
		t.Fatalf("expected 1 anchor")
	}
	// Idempotent duplicate
	rec2, err := m.Anchor(testHashValue)
	if err != nil {
		t.Fatalf("duplicate anchor error: %v", err)
	}
	if !rec2.AnchoredAt.Equal(rec.AnchoredAt) {
		t.Fatalf("expected same timestamp for duplicate anchor")
	}
	// Latest
	last, err := m.LatestAnchor()
	if err != nil {
		t.Fatalf("latest error: %v", err)
	}
	if last.Hash != testHashValue {
		t.Fatalf("latest hash mismatch")
	}
}

func TestMemoryAnchorMultiple(t *testing.T) {
	m := NewMemoryAnchor()
	if _, err := m.Anchor("h1"); err != nil {
		t.Fatalf("failed to anchor h1: %v", err)
	}
	if _, err := m.Anchor("h2"); err != nil {
		t.Fatalf("failed to anchor h2: %v", err)
	}
	if _, err := m.Anchor("h3"); err != nil {
		t.Fatalf("failed to anchor h3: %v", err)
	}
	if m.TotalAnchors() != 3 {
		t.Fatalf("expected 3 anchors")
	}
	latest, _ := m.LatestAnchor()
	if latest.Hash != "h3" {
		t.Fatalf("latest should be h3 got %s", latest.Hash)
	}
}

func TestMemoryAnchorEmptyHash(t *testing.T) {
	m := NewMemoryAnchor()
	if _, err := m.Anchor(""); err == nil {
		t.Fatalf("expected error on empty hash")
	}
}

func TestMemoryAnchorPersistence(t *testing.T) {
	// Use temp dir
	dir := t.TempDir()
	path := dir + "/anchors.json"
	m := NewMemoryAnchor()
	if err := m.EnablePersistence(path); err != nil {
		t.Fatalf("enable persistence: %v", err)
	}
	if _, err := m.Anchor("alpha"); err != nil {
		t.Fatalf("failed to anchor alpha: %v", err)
	}
	if _, err := m.Anchor("beta"); err != nil {
		t.Fatalf("failed to anchor beta: %v", err)
	}
	// Recreate new instance and load
	m2 := NewMemoryAnchor()
	if err := m2.EnablePersistence(path); err != nil {
		t.Fatalf("reload persistence: %v", err)
	}
	if m2.TotalAnchors() != 2 {
		t.Fatalf("expected 2 anchors after reload got %d", m2.TotalAnchors())
	}
	latest, _ := m2.LatestAnchor()
	if latest.Hash != "beta" {
		t.Fatalf("expected latest beta got %s", latest.Hash)
	}
}
