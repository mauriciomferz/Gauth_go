package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// TestCombinedAnchorEmissionWithRotation ensures rotation ledger head participates in combined hash.
func TestCombinedAnchorEmissionWithRotation(t *testing.T) {
	tmpReceipts := "test_ext_anchor_receipts_rotation.json"
	_ = os.Remove(tmpReceipts)
	os.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory")
	os.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH", tmpReceipts)
	// Enable notarization to initialize rotation ledger.
	tmpLedger := "test_rotation_ledger.json"
	_ = os.Remove(tmpLedger)
	os.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	os.Setenv("GAUTH_ROTATION_LEDGER_PATH", tmpLedger)

	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	// Seed capability hash
	srv.capabilityRegistryHash = "feedfacecafedead"
	// Append a rotation descriptor to ledger directly (simulate key rotation event)
	if srv.rotationLedger == nil {
		t.Fatalf("rotation ledger not initialized")
	}
	// Minimal descriptor (OldKeyID/NewKeyID will be assigned upon signing when implemented). For ledger chaining
	// we just need EffectiveTime/Reason; optional PrevRotationHash left empty for first entry.
	rd := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-25T00:00:00Z", Reason: "scheduled"}
	if _, err := srv.rotationLedger.AppendDescriptor(rd); err != nil {
		t.Fatalf("append rotation descriptor failed: %v", err)
	}
	rotHead := srv.rotationLedger.HeadHash()
	if rotHead == "" {
		t.Fatalf("expected non-empty rotation head after append")
	}

	// Emit combined anchor
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/anchor/emitCombined", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "rotation_head") || !strings.Contains(body, rotHead) {
		t.Fatalf("response missing rotation_head %s body=%s", rotHead, body)
	}
	// Validate digest matches cap:rotHead
	combo := srv.capabilityRegistryHash + ":" + rotHead
	exp := sha256.Sum256([]byte(combo))
	expHex := hex.EncodeToString(exp[:])
	if !strings.Contains(body, expHex) {
		t.Fatalf("response missing expected combined hash %s", expHex)
	}

	// Chain retrieval should include combined hash
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/anchor/chain", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("chain status %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), expHex) {
		t.Fatalf("chain output missing combined hash %s", expHex)
	}

	_ = os.Remove(tmpReceipts)
	_ = os.Remove(tmpLedger)
}
