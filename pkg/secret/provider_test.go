package secret

import (
	"context"
	"testing"
)

func TestMemoryProviderCRUD(t *testing.T) {
	mp := NewMemory()
	ctx := context.Background()
	// Get on missing
	if _, err := mp.Get(ctx, "a"); err == nil {
		t.Fatalf("expected not found")
	}
	if err := mp.Set(ctx, "a", "value1"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, err := mp.Get(ctx, "a")
	if err != nil || v != "value1" {
		t.Fatalf("get mismatch v=%s err=%v", v, err)
	}
	// IfNotExists should fail
	if err2 := mp.Set(ctx, "a", "value2", IfNotExists()); err2 == nil {
		t.Fatalf("expected already exist error")
	}
	// Overwrite without flag
	if err2 := mp.Set(ctx, "a", "value2"); err2 != nil {
		t.Fatalf("overwrite: %v", err2)
	}
	if v, _ = mp.Get(ctx, "a"); v != "value2" {
		t.Fatalf("expected overwrite value2 got %s", v)
	}
	// List
	_ = mp.Set(ctx, "b/c", "x")
	keys, err := mp.List(ctx, "")
	if err != nil || len(keys) < 2 {
		t.Fatalf("list expected >=2 got %v err=%v", keys, err)
	}
	pref, _ := mp.List(ctx, "b/")
	if len(pref) != 1 || pref[0] != "b/c" {
		t.Fatalf("prefix list mismatch %v", pref)
	}
	// Delete
	if err := mp.Delete(ctx, "a"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := mp.Get(ctx, "a"); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

func TestVaultStubName(t *testing.T) {
	v := NewVaultStub()
	if v.Name() != "vaultstub" {
		t.Fatalf("expected vaultstub got %s", v.Name())
	}
}
