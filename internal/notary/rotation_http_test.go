package notary

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRotationHTTPHandler(t *testing.T) {
	// Generate keys
	o1Pub, o1Priv, _ := ed25519.GenerateKey(nil)
	o2Pub, o2Priv, _ := ed25519.GenerateKey(nil)
	// Single rotation descriptor
	d1 := &KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := SignRotationDescriptor(o1Priv, o2Priv, d1); err != nil {
		t.Fatalf("sign rotation: %v", err)
	}
	// receipt hash for first rotation
	hashes := []string{"hash_rot_1"}
	rv := &RotationVerifier{Descriptors: []*KeyRotationDescriptor{d1}, ReceiptHashes: hashes, OldKeys: []ed25519.PublicKey{o1Pub}, NewKeys: []ed25519.PublicKey{o2Pub}}
	handler := rv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/rotations/verification", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	var parsed struct {
		GeneratedAt string                      `json:"generated_at"`
		Summary     RotationVerificationSummary `json:"summary"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if parsed.Summary.Total != 1 || parsed.Summary.Failures != 0 || !parsed.Summary.AllSignaturesOK || !parsed.Summary.AllContinuityOK {
		t.Fatalf("unexpected summary: %+v", parsed.Summary)
	}
}
