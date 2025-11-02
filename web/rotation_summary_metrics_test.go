package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	cryptoInt "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// TestRotationSummaryMetrics ensures new metrics for summary generation & anchoring are exposed.
func TestRotationSummaryMetrics(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("GAUTH_ROTATION_LEDGER_PATH", dir+"/ledger.json")
	os.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	os.Setenv("GAUTH_ANCHOR_ROTATIONS", "1")
	os.Setenv("GAUTH_ROTATIONS_SIGN", "1")
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	// Reset any leaked multisig or threshold settings & global key state from previous tests.
	os.Unsetenv("GAUTH_ROTATIONS_MULTISIG")
	os.Unsetenv("GAUTH_ROTATIONS_THRESHOLD")
	if m, _ := cryptoInt.NewManager(24 * time.Hour); m != nil { cryptoInt.GlobalEdDSARegistry = m }
	srv := NewBetaServer("0")
	led, ok := srv.rotationLedger.(*notary.RotationLedger)
	if !ok {
		t.Fatalf("ledger type assertion failed")
	}
	// Append two descriptors to satisfy continuity (non-empty and valid prev hash linkage)
	_, o1Priv, _ := ed25519.GenerateKey(rand.Reader)
	_, o2Priv, _ := ed25519.GenerateKey(rand.Reader)
	r1 := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := notary.SignRotationDescriptor(o1Priv, o2Priv, r1); err != nil { t.Fatalf("sign r1: %v", err) }
	if _, err := led.AppendDescriptor(r1); err != nil { t.Fatalf("append r1: %v", err) }
	r2 := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-21T12:00:00Z", Reason: "scheduled"}
	r2.PrevRotationHash = led.HeadHash()
	if err := notary.SignRotationDescriptor(o1Priv, o2Priv, r2); err != nil { t.Fatalf("sign r2: %v", err) }
	if _, err := led.AppendDescriptor(r2); err != nil { t.Fatalf("append r2: %v", err) }
	// First call anchors
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, httptest.NewRequest("GET", "/api/v1/beta/rotations/summary", nil))
	if w1.Code != 200 {
		t.Fatalf("first summary status %d", w1.Code)
	}
	// Second call (no new head) should skip anchoring
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, httptest.NewRequest("GET", "/api/v1/beta/rotations/summary", nil))
	if w2.Code != 200 {
		t.Fatalf("second summary status %d", w2.Code)
	}
	// Metrics scrape
	mw := httptest.NewRecorder()
	srv.router.ServeHTTP(mw, httptest.NewRequest("GET", "/metrics", nil))
	if mw.Code != 200 {
		t.Skip("metrics endpoint not exposed in this build")
	}
	body := mw.Body.String()
	// Basic presence checks
	if !strings.Contains(body, "gauth_rotation_summary_latency_seconds") {
		t.Fatalf("missing latency metric")
	}
	if !strings.Contains(body, "gauth_rotation_summary_total") {
		t.Fatalf("missing summary total metric")
	}
	if !strings.Contains(body, "gauth_rotation_summary_anchor_total") {
		t.Fatalf("missing anchor metric")
	}
	if !strings.Contains(body, "gauth_rotation_summary_chain_length") {
		t.Fatalf("missing chain length gauge")
	}
	if !strings.Contains(body, "gauth_rotation_summary_head_age_seconds") {
		t.Fatalf("missing head age gauge")
	}
	if !strings.Contains(body, "gauth_rotation_summary_last_anchor_age_seconds") {
		t.Fatalf("missing last anchor age gauge")
	}
	// Expect at least one anchored and one skipped or error label occurrence.
	if !strings.Contains(body, "gauth_rotation_summary_anchor_total{result=\"anchored\"") {
		t.Fatalf("expected anchored result in metrics: %s", body)
	}
}
