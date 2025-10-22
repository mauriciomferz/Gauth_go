package notary

import (
	"crypto/ed25519"
	"testing"
)

// TestDualSignatureRotationDescriptor covers signing & verification success and failure modes.
func TestDualSignatureRotationDescriptor(t *testing.T) {
	// Generate old/new keys
	oldPub, oldPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("old key gen: %v", err)
	}
	newPub, newPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("new key gen: %v", err)
	}
	rd := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := SignRotationDescriptor(oldPriv, newPriv, rd); err != nil {
		t.Fatalf("sign: %v", err)
	}
	valid, reason := VerifyRotationDescriptor(rd, oldPub, newPub)
	if !valid || reason != "" {
		t.Fatalf("expected valid rotation descriptor; valid=%v reason=%s", valid, reason)
	}

	// Tamper payload field (Reason) => signatures invalid
	rdT := *rd
	rdT.Reason = "tampered"
	valid, reason = VerifyRotationDescriptor(&rdT, oldPub, newPub)
	if valid || reason == "" {
		t.Fatalf("expected invalid after tamper; valid=%v reason=%s", valid, reason)
	}
	if reason != "old_sig_invalid" && reason != "new_sig_invalid" {
		t.Fatalf("expected sig invalid reason, got %s", reason)
	}

	// Remove old signature only
	rdMissing := *rd
	rdMissing.OldKeySignature = ""
	valid, reason = VerifyRotationDescriptor(&rdMissing, oldPub, newPub)
	if valid || reason != reasonMissingOldSig {
		t.Fatalf("expected %s; valid=%v reason=%s", reasonMissingOldSig, valid, reason)
	}

	// Kid mismatch (simulate wrong provided pub key)
	otherPub, _, _ := ed25519.GenerateKey(nil)
	valid, reason = VerifyRotationDescriptor(rd, otherPub, newPub)
	if valid || reason != "kid_mismatch_old" {
		t.Fatalf("expected kid_mismatch_old; valid=%v reason=%s", valid, reason)
	}
}
