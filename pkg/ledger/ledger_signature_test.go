package ledger

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
	"time"
)

func TestMemoryStoreEntrySignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	ms := NewMemoryStore().(*memoryStore)
	ms.ConfigureEd25519Signer(priv, pub, "testkey")
	ctx := context.Background()
	e := &Entry{ID: "sig1", TS: time.Now(), Type: "test", Subject: "alice", Object: "obj"}
	if err := ms.Append(ctx, e); err != nil {
		t.Fatalf("append: %v", err)
	}
	if e.Signature == nil {
		t.Fatalf("expected signature present")
	}
	vr, err := ms.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if vr.Mismatches != 0 {
		t.Fatalf("unexpected mismatches: %d", vr.Mismatches)
	}
}

func TestBoltStoreEntrySignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	f, err := os.CreateTemp("", "ledger-bolt-sig-*.db")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)
	storeIface, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("new bolt store: %v", err)
	}
	bs := storeIface.(*boltStore)
	bs.ConfigureEd25519Signer(priv, pub, "testkey")
	defer bs.Close()
	ctx := context.Background()
	e := &Entry{ID: "sig2", TS: time.Now(), Type: "test", Subject: "alice", Object: "obj"}
	if err := bs.Append(ctx, e); err != nil {
		t.Fatalf("append: %v", err)
	}
	if e.Signature == nil {
		t.Fatalf("expected signature present")
	}
	vr, err := bs.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if vr.Mismatches != 0 {
		t.Fatalf("unexpected mismatches: %d", vr.Mismatches)
	}
}
