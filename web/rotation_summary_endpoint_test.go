package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	notary "github.com/mauriciomferz/AgentAuth/internal/notary"
	cryptoInt "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// TestRotationSummaryEndpointAnchoring verifies anchoring only occurs once for same head hash when GAUTH_ANCHOR_ROTATIONS=1.
func TestRotationSummaryEndpointAnchoring(t *testing.T) {
	dir := t.TempDir()
	ledgerPath := dir + "/ledger.json"
	t.Setenv("GAUTH_ROTATION_LEDGER_PATH", ledgerPath)
	t.Setenv("GAUTH_ANCHOR_ROTATIONS", "1")
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")  // ensure notarizer & ledger initialization path executes
	t.Setenv("GAUTH_ANCHOR_PROVIDER", "memory") // initialize memory anchor client
	// Enable signing to exercise signature path (optional)
	t.Setenv("GAUTH_ROTATIONS_SIGN", "1")
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	// Ensure isolation from previous tests that may have mutated global registry or multisig env.
	os.Unsetenv("GAUTH_ROTATIONS_MULTISIG")
	os.Unsetenv("GAUTH_ROTATIONS_THRESHOLD")
	m, _ := cryptoInt.NewManager(24 * time.Hour)
	// Initialize server
	srv := NewBetaServer("0", WithKeyProvider(m))
	t.Cleanup(func() { srv.Shutdown() })
	// Append two descriptors through ledger directly (simulate rotation activity).
	led, ok := srv.rotationLedger.(*notary.RotationLedger)
	if !ok {
		t.Fatalf("ledger type conversion failed")
	}
	_, o1Priv, _ := ed25519.GenerateKey(rand.Reader)
	_, o2Priv, _ := ed25519.GenerateKey(rand.Reader)
	// First rotation descriptor (dual signature)
	r1 := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := notary.SignRotationDescriptor(o1Priv, o2Priv, r1); err != nil {
		t.Fatalf("sign r1: %v", err)
	}
	if _, err := led.AppendDescriptor(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	// Second descriptor to avoid empty-chain edge cases and continuity gap false positives
	r2 := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-21T12:00:00Z", Reason: "scheduled"}
	// Set PrevRotationHash to previous head hash for continuity correctness
	r2.PrevRotationHash = led.HeadHash()
	if err := notary.SignRotationDescriptor(o1Priv, o2Priv, r2); err != nil {
		t.Fatalf("sign r2: %v", err)
	}
	if _, err := led.AppendDescriptor(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	// Fetch summary first time (should anchor)
	req1 := httptest.NewRequest("GET", "/api/v1/beta/rotations/summary", nil)
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected 200 got %d", w1.Code)
	}
	var resp1 struct {
		Success    bool                   `json:"success"`
		Configured bool                   `json:"configured"`
		Anchored   bool                   `json:"anchored"`
		Summary    notary.RotationSummary `json:"summary"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal resp1: %v", err)
	}
	if !resp1.Success || !resp1.Configured {
		t.Fatalf("unexpected resp1: %+v", resp1)
	}
	if !resp1.Anchored {
		t.Fatalf("expected first call anchored=true")
	}
	head := resp1.Summary.HeadHash
	if head == "" {
		t.Fatalf("expected head hash non-empty")
	}
	// Second call without new descriptor should not anchor again
	req2 := httptest.NewRequest("GET", "/api/v1/beta/rotations/summary", nil)
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 got %d", w2.Code)
	}
	var resp2 struct {
		Success    bool                   `json:"success"`
		Configured bool                   `json:"configured"`
		Anchored   bool                   `json:"anchored"`
		Summary    notary.RotationSummary `json:"summary"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal resp2: %v", err)
	}
	if !resp2.Success || !resp2.Configured {
		t.Fatalf("unexpected resp2: %+v", resp2)
	}
	if resp2.Anchored {
		t.Fatalf("expected second call anchored=false")
	}
	if resp2.Summary.HeadHash != head {
		t.Fatalf("head hash changed unexpectedly head1=%s head2=%s", head, resp2.Summary.HeadHash)
	}
}
