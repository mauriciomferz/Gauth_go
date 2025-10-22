package notary

import (
	"crypto/ed25519"
	"testing"
)

// TestRotationSummarySignatureValid ensures a signed summary verifies successfully.
func TestRotationSummarySignatureValid(t *testing.T) {
	// Build minimal ledger with two descriptors.
	dir := t.TempDir()
	path := dir + "/ledger.json"
	led := NewRotationLedger(path)
	if err := led.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	// Generate sequential keypairs for two rotations (o1->o2, o2->o3)
	_, o1Priv, _ := ed25519.GenerateKey(nil)
	_, o2Priv, _ := ed25519.GenerateKey(nil)
	o3Pub, o3Priv, _ := ed25519.GenerateKey(nil)
	// Rotation 1
	r1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := SignRotationDescriptor(o1Priv, o2Priv, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	// Rotation 2 (prev hash = head after r1)
	r2 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T13:00:00Z", Reason: "scheduled", PrevRotationHash: led.HeadHash()}
	if err := SignRotationDescriptor(o2Priv, o3Priv, r2); err != nil {
		t.Fatalf("sign r2: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	sum := BuildRotationSummary(led)
	// Sign using active (latest new key = o3Priv)
	kid := computeKeyID(o3Pub)
	if err := SignRotationSummary(&sum, o3Priv, kid); err != nil {
		t.Fatalf("sign summary: %v", err)
	}
	ok, reason := VerifyRotationSummary(&sum, o3Pub)
	if !ok || reason != "" {
		t.Fatalf("expected summary valid got ok=%v reason=%s", ok, reason)
	}
}

// TestRotationSummarySignatureTamper modifies aggregate hash post-signature causing verification failure.
func TestRotationSummarySignatureTamper(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ledger.json"
	led := NewRotationLedger(path)
	if err := led.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	// Single rotation sufficient.
	_, o1Priv, _ := ed25519.GenerateKey(nil)
	o2Pub, o2Priv, _ := ed25519.GenerateKey(nil)
	r1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := SignRotationDescriptor(o1Priv, o2Priv, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	sum := BuildRotationSummary(led)
	kid := computeKeyID(o2Pub)
	if err := SignRotationSummary(&sum, o2Priv, kid); err != nil {
		t.Fatalf("sign summary: %v", err)
	}
	// Tamper aggregate hash
	sum.AggregateHash = "deadbeef" + sum.AggregateHash
	ok, reason := VerifyRotationSummary(&sum, o2Pub)
	if ok || reason != "signature_invalid" {
		t.Fatalf("expected signature_invalid after tamper got ok=%v reason=%s", ok, reason)
	}
}
