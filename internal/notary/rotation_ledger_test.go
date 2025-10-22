package notary

import (
	"crypto/ed25519"
	"os"
	"testing"
)

// TestRotationLedgerPersistence verifies append then reload preserves head hash and entry chain.
func TestRotationLedgerPersistence(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ledger.json"
	led := NewRotationLedger(path)
	if err := led.Load(); err != nil {
		t.Fatalf("initial load: %v", err)
	}
	// Generate three sequential key pairs and descriptors
	_, o1Priv, _ := ed25519.GenerateKey(nil)
	_, o2Priv, _ := ed25519.GenerateKey(nil)
	_, o3Priv, _ := ed25519.GenerateKey(nil)
	d1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := SignRotationDescriptor(o1Priv, o2Priv, d1); err != nil {
		t.Fatalf("sign d1: %v", err)
	}
	if _, err := led.AppendDescriptor(d1); err != nil {
		t.Fatalf("append d1: %v", err)
	}
	d2 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T13:00:00Z", Reason: "scheduled", PrevRotationHash: led.HeadHash()}
	if err := SignRotationDescriptor(o2Priv, o3Priv, d2); err != nil {
		t.Fatalf("sign d2: %v", err)
	}
	if _, err := led.AppendDescriptor(d2); err != nil {
		t.Fatalf("append d2: %v", err)
	}
	headBefore := led.HeadHash()
	entriesBefore := led.Entries()
	if len(entriesBefore) != 2 {
		t.Fatalf("expected 2 entries got %d", len(entriesBefore))
	}
	// Reload new instance from disk
	led2 := NewRotationLedger(path)
	if err := led2.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	headAfter := led2.HeadHash()
	if headAfter != headBefore {
		t.Fatalf("head mismatch after reload before=%s after=%s", headBefore, headAfter)
	}
	if len(led2.Entries()) != 2 {
		t.Fatalf("expected 2 entries after reload got %d", len(led2.Entries()))
	}
	// Ensure chain linkage intact
	e0 := led2.Entries()[0]
	e1 := led2.Entries()[1]
	if e1.PrevHash != e0.Hash {
		t.Fatalf("continuity break: e1.prev=%s e0.hash=%s", e1.PrevHash, e0.Hash)
	}
	// Confirm file exists non-empty
	if st, err := os.Stat(path); err != nil || st.Size() == 0 {
		t.Fatalf("ledger file missing or empty")
	}
}
