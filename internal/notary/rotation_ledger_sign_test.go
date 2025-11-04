package notary

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
)

// TestRotationLedgerEntrySigning verifies RB5 signature inclusion and verification.
func TestRotationLedgerEntrySigning(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key gen: %v", err)
	}
	f, err := os.CreateTemp("", "rot-ledger-*.json")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	defer os.Remove(path)
	led := NewRotationLedger(path)
	if err := led.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	led.ConfigureEd25519Signer(privKey, "ed25519:test")
	// Append two descriptors
	r1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-27T12:00:00Z", Reason: "scheduled"}
	r2 := &KeyRotationDescriptor{EffectiveTime: "2025-10-27T13:00:00Z", Reason: "scheduled", PrevRotationHash: "h1"}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	entries := led.Entries()
	if entries[0].Signature == "" || entries[1].Signature == "" {
		t.Fatalf("expected signatures present")
	}
	mismatches, invalid := VerifyRotationLedger(entries, false, func(kid string) ed25519.PublicKey {
		// single kid mapping
		return pubKey
	})
	if mismatches != 0 {
		t.Fatalf("unexpected mismatches=%d", mismatches)
	}
	if invalid != 0 {
		t.Fatalf("unexpected invalid sigs=%d", invalid)
	}
}

// TestRotationLedgerTamperDetect ensures hash tamper is detected.
func TestRotationLedgerTamperDetect(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key gen: %v", err)
	}
	f, err := os.CreateTemp("", "rot-ledger-tamper-*.json")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	defer os.Remove(path)
	led := NewRotationLedger(path)
	if err := led.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	led.ConfigureEd25519Signer(privKey, "ed25519:test")
	r1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-27T12:00:00Z", Reason: "scheduled"}
	r2 := &KeyRotationDescriptor{EffectiveTime: "2025-10-27T13:00:00Z", Reason: "scheduled", PrevRotationHash: "h1"}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	entries := led.Entries()
	// Tamper second descriptor reason
	entries[1].Descriptor.Reason = "tampered"
	mismatches, _ := VerifyRotationLedger(entries, false, func(kid string) ed25519.PublicKey { return nil })
	if mismatches == 0 {
		t.Fatalf("expected mismatches after tamper")
	}
}
