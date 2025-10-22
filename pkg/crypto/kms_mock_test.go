package crypto

import "testing"

func TestMockKMSRotationAndSnapshot(t *testing.T) {
	kms, err := NewMockKMS()
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	// Initial list
	keys, err := kms.ListKeys()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || !keys[0].Active {
		t.Fatalf("expected 1 active key, got %+v", keys)
	}
	snap := kms.Snapshot()
	if snap["list_calls"] != 1 {
		t.Fatalf("expected list_calls=1, got %v", snap["list_calls"])
	}
	// Rotate
	id, err := kms.Rotate()
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if id == keys[0].ID {
		t.Fatalf("rotation did not change active id")
	}
	keys2, err := kms.ListKeys()
	if err != nil {
		t.Fatalf("list2: %v", err)
	}
	if len(keys2) != 2 {
		t.Fatalf("expected 2 keys after rotation, got %d", len(keys2))
	}
	// Ensure exactly one active
	var activeCount int
	for _, k := range keys2 {
		if k.Active {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("expected 1 active key, got %d", activeCount)
	}
	snap2 := kms.Snapshot()
	if snap2["rotate_calls"] != 1 {
		t.Fatalf("expected rotate_calls=1 got %v", snap2["rotate_calls"])
	}
	if snap2["list_calls"] != 2 {
		t.Fatalf("expected list_calls=2 got %v", snap2["list_calls"])
	}
}
