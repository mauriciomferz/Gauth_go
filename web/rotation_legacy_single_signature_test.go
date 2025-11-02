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

// TestRotationSummary_LegacySingleSignature verifies legacy single-sign path when multisig disabled.
func TestRotationSummary_LegacySingleSignature(t *testing.T) {
    os.Setenv("GAUTH_ROTATIONS_SIGN", "1")
    os.Unsetenv("GAUTH_ROTATIONS_MULTISIG")
    os.Unsetenv("GAUTH_ROTATIONS_THRESHOLD")
    tmp := t.TempDir()
    ledgerPath := tmp + "/ledger-legacy.json"
    led := notary.NewRotationLedger(ledgerPath)
    if err := led.Load(); err != nil { t.Fatalf("ledger load: %v", err) }
    // Build minimal non-empty chain
    _, pk1, _ := ed25519.GenerateKey(nil)
    _, pk2, _ := ed25519.GenerateKey(nil)
    r1 := &notary.KeyRotationDescriptor{EffectiveTime: time.Now().UTC().Format(time.RFC3339Nano), Reason: "init"}
    if err := notary.SignRotationDescriptor(pk1, pk2, r1); err != nil { t.Fatalf("sign r1: %v", err) }
    if _, err := led.AppendDescriptor(r1); err != nil { t.Fatalf("append r1: %v", err) }
    // Key manager single active key only (no rotate)
    m, _ := cryptoInt.NewManager(1 * time.Hour)
    cryptoInt.GlobalEdDSARegistry = m
    s := NewBetaServer("")
    s.rotationLedger = led
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/beta/rotations/summary", http.NoBody)
    s.router.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String()) }
    var resp struct {
        Success bool `json:"success"`
        Summary struct {
            Kid            string `json:"kid"`
            Signature      string `json:"signature"`
            Mode           string `json:"mode"`
            Threshold      int    `json:"threshold"`
            SatisfiedWeight int   `json:"satisfied_weight"`
            Signatures     []struct{ Kid, Mode, Signature string } `json:"signatures"`
        } `json:"summary"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil { t.Fatalf("unmarshal: %v", err) }
    if !resp.Success { t.Fatalf("success=false body=%s", w.Body.String()) }
    if resp.Summary.Kid == "" || resp.Summary.Signature == "" { t.Fatalf("expected single signature fields populated") }
    if resp.Summary.Threshold != 0 { t.Fatalf("threshold should be zero in legacy path") }
    if resp.Summary.SatisfiedWeight != 1 { t.Fatalf("satisfied_weight should be 1 for single-sign path") }
    if len(resp.Summary.Signatures) != 1 { t.Fatalf("signatures array should contain exactly one entry in legacy path") }
}
