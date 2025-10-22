package notary

import (
	"crypto/ed25519"
	"testing"
)

// helper to create signed descriptor
func makeSignedDescriptor(oldPriv, newPriv ed25519.PrivateKey, prevHash string) *KeyRotationDescriptor {
	rd := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled", PrevRotationHash: prevHash}
	if err := SignRotationDescriptor(oldPriv, newPriv, rd); err != nil {
		return nil
	}
	return rd
}

func TestVerifyAllRotations(t *testing.T) {
	// Generate key pairs for three rotations (old/new progression)
	o1Pub, o1Priv, _ := ed25519.GenerateKey(nil)
	o2Pub, o2Priv, _ := ed25519.GenerateKey(nil)
	o3Pub, o3Priv, _ := ed25519.GenerateKey(nil)
	// Descriptors chain: (o1->o2), (o2->o3)
	d1 := makeSignedDescriptor(o1Priv, o2Priv, "")
	if d1 == nil {
		t.Fatalf("descriptor1 signing failed")
	}
	// Fake receipt hashes for rotation events
	h1 := "hash_rot_1"
	d2 := makeSignedDescriptor(o2Priv, o3Priv, h1)
	if d2 == nil {
		t.Fatalf("descriptor2 signing failed")
	}
	descriptors := []*KeyRotationDescriptor{d1, d2}
	hashes := []string{h1, "hash_rot_2"}
	oldPubs := []ed25519.PublicKey{o1Pub, o2Pub}
	newPubs := []ed25519.PublicKey{o2Pub, o3Pub}
	summary := VerifyAllRotations(descriptors, hashes, oldPubs, newPubs)
	if summary.Failures != 0 || !summary.AllContinuityOK || !summary.AllSignaturesOK {
		t.Fatalf("expected all OK: %+v", summary)
	}
	// Continuity failure: break prev hash of second descriptor
	d2Bad := *d2
	d2Bad.PrevRotationHash = "wrong" // keep signatures
	summary2 := VerifyAllRotations([]*KeyRotationDescriptor{d1, &d2Bad}, hashes, oldPubs, newPubs)
	if summary2.Failures != 1 || summary2.Results[1].Reason != "continuity_failure" {
		t.Fatalf("expected continuity failure: %+v", summary2)
	}
	// Signature failure: tamper signature of first descriptor
	d1Bad := *d1
	d1Bad.OldKeySignature = "" // remove old signature
	summary3 := VerifyAllRotations([]*KeyRotationDescriptor{&d1Bad, d2}, hashes, oldPubs, newPubs)
	if summary3.Failures != 1 || summary3.Results[0].Reason != "missing_old_signature" {
		t.Fatalf("expected signature failure: %+v", summary3)
	}
}
