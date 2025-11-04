package web

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	cryptoInt "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// TestRotationSummary_MultiSignature ensures multi-signature mode emits multiple signatures and threshold fields.
func TestRotationSummary_MultiSignature(t *testing.T) {
	os.Setenv("GAUTH_ROTATIONS_SIGN", "1")
	os.Setenv("GAUTH_ROTATIONS_MULTISIG", "1")
	os.Setenv("GAUTH_ROTATIONS_THRESHOLD", "2")
	tmp := t.TempDir()
	ledgerPath := tmp + "/ledger-ms.json"
	led := notary.NewRotationLedger(ledgerPath)
	if err := led.Load(); err != nil {
		t.Fatalf("ledger load: %v", err)
	}
	// Create two sequential rotations
	_, priv1, _ := ed25519.GenerateKey(nil)
	_, priv2, _ := ed25519.GenerateKey(nil)
	r1 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().UTC().Format(time.RFC3339Nano), Reason: "scheduled"}
	if err := notary.SignRotationDescriptor(priv1, priv2, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	// Generate key pair for second rotation descriptor (public key unused outside descriptor signing)
	_, priv3, _ := ed25519.GenerateKey(nil)
	r2 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339Nano), Reason: "scheduled", PrevRotationHash: led.HeadHash()}
	if err := notary.SignRotationDescriptor(priv2, priv3, r2); err != nil {
		t.Fatalf("sign r2: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	// Prepare key manager with active + history key by performing a manual rotation.
	m, _ := cryptoInt.NewManager(1 * time.Hour)
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("manager rotate: %v", err)
	}
	cryptoInt.GlobalEdDSARegistry = m
	s := NewBetaServer("")
	s.rotationLedger = led
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/beta/rotations/summary", http.NoBody)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Summary struct {
			Threshold       int `json:"threshold"`
			SatisfiedWeight int `json:"satisfied_weight"`
			Signatures      []struct {
				Kid       string `json:"kid"`
				Signature string `json:"signature"`
				Mode      string `json:"mode"`
			} `json:"signatures"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success false")
	}
	if resp.Summary.Threshold != 2 {
		t.Fatalf("expected threshold 2 got %d", resp.Summary.Threshold)
	}
	if resp.Summary.SatisfiedWeight < resp.Summary.Threshold {
		t.Fatalf("satisfied_weight < threshold: %+v", resp.Summary)
	}
	if len(resp.Summary.Signatures) < 2 {
		t.Fatalf("expected >=2 signatures in multisig mode got %d", len(resp.Summary.Signatures))
	}
}
