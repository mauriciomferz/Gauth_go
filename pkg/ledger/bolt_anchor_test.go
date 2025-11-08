package ledger

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBoltAnchorFileEmission verifies periodic anchor file updates and signature presence.
func TestBoltAnchorFileEmission(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ledger.db")
	stRaw, err := NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("bolt store create: %v", err)
	}
	st, ok := stRaw.(*boltStore)
	if !ok {
		t.Fatalf("unexpected store type")
	}
	// Generate key pair for signing chain tip.
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	st.ConfigureEd25519Signer(priv, pub, "test-key")
	anchorFile := filepath.Join(dir, "anchor.json")
	if err2 := st.EnableAnchorFile(anchorFile, 10*time.Millisecond); err2 != nil {
		t.Fatalf("enable anchor file: %v", err2)
	}
	// Append several entries spaced out to trigger writes.
	for i := 0; i < 3; i++ {
		e := &Entry{
			ID:      time.Now().Format("20060102150405") + "-" + string(rune('a'+i)),
			TS:      time.Now().UTC(),
			Type:    "test",
			Subject: "sub",
			Object:  "obj",
		}
		if err2 := st.Append(context.TODO(), e); err2 != nil {
			t.Fatalf("append %d: %v", i, err2)
		}
		time.Sleep(15 * time.Millisecond)
	}
	// Read anchor file.
	data, err := os.ReadFile(anchorFile)
	if err != nil {
		t.Fatalf("read anchor file: %v", err)
	}
	var anchor struct {
		Hash       string `json:"hash"`
		AnchoredAt string `json:"anchored_at"`
		KeyID      string `json:"key_id"`
		Signature  string `json:"signature"`
		Writes     uint64 `json:"writes"`
	}
	if err := json.Unmarshal(data, &anchor); err != nil {
		t.Fatalf("unmarshal anchor: %v", err)
	}
	if anchor.Hash == "" {
		t.Fatalf("expected hash in anchor file")
	}
	if anchor.KeyID != "test-key" {
		t.Fatalf("expected key id test-key got %s", anchor.KeyID)
	}
	if anchor.Signature == "" {
		t.Fatalf("expected signature present")
	}
	if anchor.Writes < 2 {
		t.Fatalf("expected multiple writes recorded, got %d", anchor.Writes)
	}
}
