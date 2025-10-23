package ledger

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestBoltStoreAppendAndVerify(t *testing.T) {
	// Create temp file
	f, err := os.CreateTemp("", "ledger-bolt-*.db")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	defer func() { _ = os.Remove(path) }()

	storeIface, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("new bolt store: %v", err)
	}
	bs := storeIface.(*boltStore)
	defer func() { _ = bs.Close() }()

	ctx := context.Background()
	// Append a few entries
	for i := 0; i < 3; i++ {
		e := &Entry{ID: fmt.Sprintf("e%d", i), TS: time.Now(), Type: "test", Subject: "alice", Object: "obj", Metadata: map[string]interface{}{"i": i}}
		if err := bs.Append(ctx, e); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	vr, err := bs.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if vr.Count != 3 {
		t.Fatalf("expected 3 entries, got %d", vr.Count)
	}
	if vr.Mismatches != 0 {
		t.Fatalf("unexpected mismatches: %d", vr.Mismatches)
	}
	// Query by subject
	subjEntries, err := bs.QueryBySubject(ctx, "alice")
	if err != nil {
		t.Fatalf("query subject: %v", err)
	}
	if len(subjEntries) != 3 {
		t.Fatalf("expected 3 subject entries, got %d", len(subjEntries))
	}
}
