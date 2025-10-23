package secrets

import (
	"crypto/rand"
	"os"
	"testing"
)

func randKey(t *testing.T) []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return b
}

func TestFilesystemProviderLifecycle(t *testing.T) {
	dir := t.TempDir()
	mk := randKey(t)
	p, err := NewFilesystemProvider(dir, mk)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if p.Backend() != "filesystem" {
		t.Fatalf("backend mismatch")
	}
	// store
	if err := p.Store("api_key", []byte("super-secret")); err != nil {
		t.Fatalf("store: %v", err)
	}
	// get
	val, err := p.Get("api_key")
	if err != nil || string(val) != "super-secret" {
		t.Fatalf("get mismatch: %v val=%s", err, string(val))
	}
	// rotate
	newKey := randKey(t)
	if err := p.Rotate(newKey); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	val2, err := p.Get("api_key")
	if err != nil || string(val2) != "super-secret" {
		t.Fatalf("post-rotate mismatch: %v val=%s", err, string(val2))
	}
	// delete
	if err := p.Delete("api_key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := p.Get("api_key"); err == nil {
		t.Fatalf("expected error after delete")
	}
}

func TestFilesystemProviderAutoKey(t *testing.T) {
	dir := t.TempDir()
	p, err := NewFilesystemProvider(dir, nil)
	if err != nil {
		t.Fatalf("auto init: %v", err)
	}
	// touch backend to avoid unused variable
	if p.Backend() == "" {
		t.Fatalf("backend should be set")
	}
	// ensure master.key exists
	if _, err := os.Stat(dir + "/master.key"); err != nil {
		t.Fatalf("missing master key file: %v", err)
	}
}
