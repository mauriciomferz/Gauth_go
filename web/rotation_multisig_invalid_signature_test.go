package web

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	notary "github.com/mauriciomferz/AgentAuth/internal/notary"
	cryptoInt "github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/verification"
)

// TestRotationSummary_MultiSignatureInvalidSignature ensures tampering one signature causes verification failure.
func TestRotationSummary_MultiSignatureInvalidSignature(t *testing.T) {
	t.Setenv("AGENTAUTH_ROTATIONS_SIGN", "1")
	t.Setenv("AGENTAUTH_ROTATIONS_MULTISIG", "1")
	t.Setenv("AGENTAUTH_ROTATIONS_THRESHOLD", "2")
	tmp := t.TempDir()
	ledgerPath := tmp + "/ledger-invalidsig.json"
	led := notary.NewRotationLedger(ledgerPath)
	if err := led.Load(); err != nil {
		t.Fatalf("ledger load: %v", err)
	}
	// Build two rotation descriptors for non-empty chain
	_, rk1, _ := ed25519.GenerateKey(nil)
	_, rk2, _ := ed25519.GenerateKey(nil)
	r1 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().UTC().Format(time.RFC3339Nano), Reason: "init"}
	if err := notary.SignRotationDescriptor(rk1, rk2, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	_, rk3, _ := ed25519.GenerateKey(nil)
	r2 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano), Reason: "rotate", PrevRotationHash: led.HeadHash()}
	if err := notary.SignRotationDescriptor(rk2, rk3, r2); err != nil {
		t.Fatalf("sign r2: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	// Key manager with two keys for multisig
	m, _ := cryptoInt.NewManager(1 * time.Hour)
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("manager rotate1: %v", err)
	}
	if _, err := m.Rotate(); err != nil {
		t.Fatalf("manager rotate2: %v", err)
	}
	s := NewBetaServer("", WithKeyProvider(m))
	t.Cleanup(func() { s.Shutdown() })
	s.rotationLedger = led
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/beta/rotations/summary", http.NoBody)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var apiResp struct {
		Summary *verification.RotationSummary `json:"summary"`
		Success bool                          `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if apiResp.Summary == nil || len(apiResp.Summary.Signatures) < 2 {
		t.Fatalf("expected multisig signatures >=2")
	}
	// Tamper first signature bytes (truncate or replace) -> verification should fail
	apiResp.Summary.Signatures[0].Signature = "aapAA"
	err := verification.VerifyRotationSummarySignature(apiResp.Summary, m)
	if err == nil || !strings.Contains(err.Error(), "signature_invalid") {
		t.Fatalf("expected signature_invalid error, got=%v", err)
	}
}
