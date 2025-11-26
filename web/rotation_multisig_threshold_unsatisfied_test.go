package web

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	cryptoInt "github.com/mauriciomferz/Gauth_go/internal/crypto"
	notary "github.com/mauriciomferz/Gauth_go/internal/notary"
)

// TestRotationSummary_MultiSignatureThresholdUnsatisfied expects error when threshold exceeds signatures.
func TestRotationSummary_MultiSignatureThresholdUnsatisfied(t *testing.T) {
	os.Setenv("GAUTH_ROTATIONS_SIGN", "1")
	os.Setenv("GAUTH_ROTATIONS_MULTISIG", "1")
	os.Setenv("GAUTH_ROTATIONS_THRESHOLD", "5") // deliberately higher than available signatures
	tmp := t.TempDir()
	ledgerPath := tmp + "/ledger-ms-unsat.json"
	led := notary.NewRotationLedger(ledgerPath)
	if err := led.Load(); err != nil {
		t.Fatalf("ledger load: %v", err)
	}
	// Create only two rotations, produce at most 2 signatures
	_, priv1, _ := ed25519.GenerateKey(nil)
	_, priv2, _ := ed25519.GenerateKey(nil)
	r1 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().UTC().Format(time.RFC3339Nano), Reason: "scheduled"}
	if err := notary.SignRotationDescriptor(priv1, priv2, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	// second rotation
	_, priv3, _ := ed25519.GenerateKey(nil)
	r2 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().Add(1 * time.Minute).UTC().Format(time.RFC3339Nano), Reason: "scheduled", PrevRotationHash: led.HeadHash()}
	if err := notary.SignRotationDescriptor(priv2, priv3, r2); err != nil {
		t.Fatalf("sign r2: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	// Key manager with single rotation (rotate once) gives 2 keys total for signing
	m, _ := cryptoInt.NewManager(1 * time.Hour)
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("manager rotate: %v", err)
	}
	cryptoInt.GlobalEdDSARegistry = m
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	s.rotationLedger = led
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/beta/rotations/summary", http.NoBody)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d body=%s", w.Code, w.Body.String())
	}
	var errResp struct {
		Code   string         `json:"code"`
		RFC    string         `json:"rfc_ref"`
		Detail map[string]any `json:"detail"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Code != "rotation_threshold_unsatisfied" {
		t.Fatalf("expected rotation_threshold_unsatisfied got %s", errResp.Code)
	}
}
