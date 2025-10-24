package crypto

import (
	"testing"
)

func TestSignerVerifier_Ed25519RoundTrip(t *testing.T) {
	prov, err := NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	signer, err := prov.ActiveSigner()
	if err != nil {
		t.Fatalf("active signer: %v", err)
	}
	msg := []byte("hello-digest")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := prov.VerifyWith(msg, sig, signer.KeyID()); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestSignerVerifier_InvalidSignature(t *testing.T) {
	prov, _ := NewInMemoryEd25519Provider()
	signer, _ := prov.ActiveSigner()
	msg := []byte("abc")
	sig, _ := signer.Sign(msg)
	// Flip a byte
	sig[0] ^= 0xFF
	if err := prov.VerifyWith(msg, sig, signer.KeyID()); err == nil {
		t.Fatalf("expected verification failure with tampered signature")
	}
}

func TestKeyProvider_RotationVisibility(t *testing.T) {
	prov, _ := NewInMemoryEd25519Provider()
	first, _ := prov.ActiveSigner()
	firstID := first.KeyID()
	if _, err := prov.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	// Old key should still verify existing signatures
	msg := []byte("rotation-check")
	sig, _ := first.Sign(msg)
	if err := prov.VerifyWith(msg, sig, firstID); err != nil {
		t.Fatalf("previous key verify failed: %v", err)
	}
}

func TestMultiSig_ThresholdPlaceholder(t *testing.T) {
	// Placeholder for future threshold / multi-signature implementation.
}

func TestBLSAggregatedSignature_Verification(t *testing.T) {
	// Placeholder for BLS aggregated signature verification test
	// TODO: Implement BLS aggregated signature test logic
}

func TestBatchSignature_Verification(t *testing.T) {
	// Placeholder for batch signature verification test
	// TODO: Implement batch signature test logic
}
