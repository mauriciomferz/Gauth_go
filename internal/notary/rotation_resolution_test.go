package notary

import (
	"crypto/ed25519"
	"testing"
)

// TestRotationVerificationWithResolvedKeys ensures signatures validate when key IDs map to manager public keys.
func TestRotationVerificationWithResolvedKeys(t *testing.T) {
	// Generate three sequential key pairs to simulate two rotations.
	o1Pub, o1Priv, _ := ed25519.GenerateKey(nil)
	o2Pub, o2Priv, _ := ed25519.GenerateKey(nil)
	o3Pub, o3Priv, _ := ed25519.GenerateKey(nil)
	// First rotation: o1 -> o2
	r1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := SignRotationDescriptor(o1Priv, o2Priv, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	// Second rotation: o2 -> o3 (prev hash simulated as h1)
	r2 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T13:00:00Z", Reason: "scheduled", PrevRotationHash: "h1"}
	if err := SignRotationDescriptor(o2Priv, o3Priv, r2); err != nil {
		t.Fatalf("sign r2: %v", err)
	}
	descriptors := []*KeyRotationDescriptor{r1, r2}
	receiptHashes := []string{"h1", "h2"} // continuity: r2 expects previous hash h1
	oldPubs := []ed25519.PublicKey{o1Pub, o2Pub}
	newPubs := []ed25519.PublicKey{o2Pub, o3Pub}
	summary := VerifyAllRotations(descriptors, receiptHashes, oldPubs, newPubs)
	if summary.Total != 2 || summary.Failures != 0 || !summary.AllSignaturesOK || !summary.AllContinuityOK {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
