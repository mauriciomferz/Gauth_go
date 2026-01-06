package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreAppendAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy_store.json")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new filestore: %v", err)
	}
	ctx := context.Background()
	if h, _ := fs.Head(ctx); h != nil {
		t.Fatalf("expected empty head")
	}
	// Append bundle
	b1, err := fs.AppendBundle(ctx, Bundle{ID: "b1", Policies: []Policy{{ID: "p1", Subjects: []string{"alice"}, Rules: []Rule{{Actions: []string{"read"}, Resources: []string{"res"}, Effect: Allow}}}}})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if b1.Hash == "" {
		t.Fatalf("hash not set")
	}
	// Reload new store instance
	fs2, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	head, err := fs2.Head(ctx)
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if head == nil || head.Hash != b1.Hash {
		t.Fatalf("expected head hash %s got %+v", b1.Hash, head)
	}
	if verifyErr := fs2.VerifyChain(ctx); verifyErr != nil {
		t.Fatalf("verify chain: %v", verifyErr)
	}
	// Append second bundle and check linkage
	b2, err := fs2.AppendBundle(ctx, Bundle{ID: "b2", Policies: []Policy{{ID: "p2", Subjects: []string{"alice"}, Rules: []Rule{{Actions: []string{"read"}, Resources: []string{"res2"}, Effect: Allow}}}}})
	if err != nil {
		t.Fatalf("append2: %v", err)
	}
	if b2.PrevHash != b1.Hash {
		t.Fatalf("expected prev hash %s got %s", b1.Hash, b2.PrevHash)
	}
	// Corrupt file and expect reload failure
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("corrupt write: %v", err)
	}
	if _, err := NewFileStore(path); err == nil {
		t.Fatalf("expected error on corrupt file")
	}
}

func TestFileStoreConcurrencySafety(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy_store.json")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new filestore: %v", err)
	}
	done := make(chan struct{})
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			_, _ = fs.AppendBundle(ctx, Bundle{ID: "b" + time.Now().Format("150405.000") + string(rune('a'+i)), Policies: []Policy{{ID: "p", Subjects: []string{"u"}, Rules: []Rule{{Actions: []string{"read"}, Resources: []string{"r"}, Effect: Allow}}}}})
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if h, _ := fs.Head(ctx); h == nil {
		t.Fatalf("expected head after concurrent appends")
	}
	if err := fs.VerifyChain(ctx); err != nil {
		t.Fatalf("verify chain: %v", err)
	}
}
