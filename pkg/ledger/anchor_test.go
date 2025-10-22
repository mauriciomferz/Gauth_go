package ledger

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

type mockAnchor struct{ called int }

func (m *mockAnchor) Anchor(hash string) error {
	if hash != "" {
		m.called++
	}
	return nil
}

func TestAnchoringStoreAppend(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	ms := NewMemoryStore().(*memoryStore)
	ms.ConfigureEd25519Signer(priv, pub, "k")
	ma := &mockAnchor{}
	store := NewAnchoringStore(ms, ma)
	ctx := context.Background()
	if err := store.Append(ctx, &Entry{ID: "a1", TS: time.Now(), Type: "t", Subject: "s", Object: "o"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if ma.called == 0 {
		t.Fatalf("expected anchor invocation")
	}
}
