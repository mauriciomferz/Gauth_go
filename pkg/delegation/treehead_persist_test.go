package delegation

import (
	"os"
	"testing"
	"time"

	cryptoInt "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

func TestSignedTreeHeadPersistence(t *testing.T) {
	// Ensure multi-sig threshold not set so single signature verification works predictably
	os.Unsetenv("GAUTH_MULTI_SIG_THRESHOLD")
	tmpFile, err := os.CreateTemp("", "sth_persist_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	os.Setenv("GAUTH_STH_PERSIST_PATH", tmpFile.Name())
	km, err := cryptoInt.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	chain := NewRevocationChain(WithKeyProvider(km))
	// Append an event and sign tree head (auto-save expected)
	if _, err2 := chain.Append(RevocationEvent{ID: "rev-a", DelegationID: "del-a"}); err2 != nil {
		t.Fatalf("append: %v", err2)
	}
	if _, err2 := chain.SignTreeHead(); err2 != nil {
		t.Fatalf("sign: %v", err2)
	}
	// Confirm file non-empty
	info, err := os.Stat(tmpFile.Name())
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected persisted file non-empty")
	}
	// New chain instance loads from file
	chain2 := NewRevocationChain(WithKeyProvider(km))
	if err := chain2.LoadSignedTreeHeads(tmpFile.Name()); err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(chain2.TreeHeads()) != 1 {
		t.Fatalf("expected 1 loaded tree head, got %d", len(chain2.TreeHeads()))
	}
}

func TestSignedTreeHeadPersistenceMultiSig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sth_persist_ms_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	os.Setenv("GAUTH_STH_PERSIST_PATH", tmpFile.Name())
	os.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "2")
	km, err := cryptoInt.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	// Rotate to produce at least 2 keys
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate1: %v", err)
	}
	chain := NewRevocationChain(WithKeyProvider(km))
	if _, err := chain.Append(RevocationEvent{ID: "rev-x", DelegationID: "del-x"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Load and verify multi-sig properties
	chain2 := NewRevocationChain(WithKeyProvider(km))
	if err := chain2.LoadSignedTreeHeads(tmpFile.Name()); err != nil {
		t.Fatalf("load: %v", err)
	}
	sth := chain2.LatestTreeHead()
	if sth == nil {
		t.Fatalf("expected sth")
	}
	if sth.Threshold != 2 {
		t.Fatalf("threshold mismatch loaded=%d", sth.Threshold)
	}
	if len(sth.Signatures) < 2 {
		t.Fatalf("expected >=2 signatures loaded")
	}
}
