package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestCombinedAnchorEmission exercises /api/v1/anchor/emitCombined and validates metrics & chain state.
func TestCombinedAnchorEmission(t *testing.T) {
	// Configure receipt store path (temp file) and seed capability hash via environment.
	tmpFile := "test_ext_anchor_receipts.json"
	_ = os.Remove(tmpFile)
	os.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH", tmpFile)
	os.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory") // ensure provider initialization triggers receipt store setup
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	// Seed capability registry hash directly (prototype placeholder).
	srv.capabilityRegistryHash = "deadbeefcafecafe"

	// First emission
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/anchor/emitCombined", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 emission status got %d body=%s", w.Code, w.Body.String())
	}
	combo := srv.capabilityRegistryHash + ":" + "" // rotation head empty
	expDigest := sha256.Sum256([]byte(combo))
	expHex := hex.EncodeToString(expDigest[:])

	// Chain query
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/anchor/chain", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 chain status got %d body=%s", w2.Code, w2.Body.String())
	}
	snap := mem.SnapshotEx()
	if snap.CombinedAnchorEmitted == 0 {
		t.Fatalf("expected CombinedAnchorEmitted > 0")
	}
	if snap.CombinedAnchorFailures != 0 {
		t.Fatalf("expected no failures; got %d", snap.CombinedAnchorFailures)
	}
	if !strings.Contains(w2.Body.String(), expHex) {
		t.Fatalf("chain response missing expected combined hash %s", expHex)
	}

	// Second emission
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/api/v1/anchor/emitCombined", nil)
	srv.router.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("expected 200 second emission got %d", w3.Code)
	}
	// Chain integrity verification
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("GET", "/api/v1/anchor/verifyChain", nil)
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 200 || !strings.Contains(wv.Body.String(), "\"status\":\"ok\"") {
		t.Fatalf("expected integrity ok response got code=%d body=%s", wv.Code, wv.Body.String())
	}
	snap2 := mem.SnapshotEx()
	if snap2.CombinedAnchorEmitted <= snap.CombinedAnchorEmitted {
		t.Fatalf("expected emission counter to increase (before=%d after=%d)", snap.CombinedAnchorEmitted, snap2.CombinedAnchorEmitted)
	}
	_ = os.Remove(tmpFile)
}

// containsSubstring simple helper (avoid pulling extra deps)
// (helpers removed; using strings.Contains)
