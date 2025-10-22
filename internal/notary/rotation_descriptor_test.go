package notary

import (
	"encoding/json"
	"testing"
)

func TestRotationDescriptorSerialization(t *testing.T) {
	rd := &KeyRotationDescriptor{OldKeyID: "ed25519:old", NewKeyID: "ed25519:new", EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled", PrevRotationHash: "abcdef"}
	r := Receipt{Hash: "sha256:rot", Timestamp: "2025-10-20T12:00:00Z", Provider: "memory", Version: 1, Success: true, LatencySeconds: 0.001, Rotation: rd}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	// Unmarshal back
	var out Receipt
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.Rotation == nil {
		t.Fatalf("rotation missing after round-trip")
	}
	if out.Rotation.OldKeyID != rd.OldKeyID || out.Rotation.NewKeyID != rd.NewKeyID || out.Rotation.PrevRotationHash != rd.PrevRotationHash {
		t.Fatalf("rotation fields mismatch: %+v", out.Rotation)
	}
}
