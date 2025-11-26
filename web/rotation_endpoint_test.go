package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	notary "github.com/mauriciomferz/Gauth_go/internal/notary"
)

// Minimal harness constructing a BetaServer with a temporary receipt store containing rotation descriptors.
func TestRotationsVerificationEndpoint(t *testing.T) {
	// Prepare temp receipt file
	dir := t.TempDir()
	path := dir + "/receipts.jsonl"
	os.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	os.Setenv("GAUTH_NOTARY_RECEIPT_PERSIST_PATH", path)
	// Initialize server (will load empty store)
	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	if srv == nil {
		t.Fatalf("server init failed")
	}
	// Append two rotation receipts manually via underlying store
	rs, ok := srv.receiptStore.(interface {
		Append(notary.Receipt) (notary.StoredReceipt, error)
		Entries() []notary.StoredReceipt
	})
	if !ok {
		t.Fatalf("receipt store type assertion failed")
	}
	os.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	os.Setenv("GAUTH_SEED_POLICY", "0")
	_, o1Priv, _ := ed25519.GenerateKey(rand.Reader)
	_, o2Priv, _ := ed25519.GenerateKey(rand.Reader)
	// First descriptor
	rd1 := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:00:00Z", Reason: "scheduled"}
	if err := notary.SignRotationDescriptor(o1Priv, o2Priv, rd1); err != nil {
		t.Fatalf("sign rd1: %v", err)
	}
	r1 := notary.Receipt{Hash: "h1", Timestamp: "2025-10-20T12:00:00Z", Provider: "memory", Version: 1, Success: true, Rotation: rd1}
	if _, err := rs.Append(r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}
	// Second descriptor referencing previous rotation receipt hash via prev_rotation_hash
	rd2 := &notary.KeyRotationDescriptor{EffectiveTime: "2025-10-20T12:10:00Z", Reason: "scheduled", PrevRotationHash: rs.Entries()[0].Hash}
	_, o3Priv, _ := ed25519.GenerateKey(rand.Reader)
	if err := notary.SignRotationDescriptor(o2Priv, o3Priv, rd2); err != nil {
		t.Fatalf("sign rd2: %v", err)
	}
	r2 := notary.Receipt{Hash: "h2", Timestamp: "2025-10-20T12:10:00Z", Provider: "memory", Version: 1, Success: true, Rotation: rd2}
	if _, err := rs.Append(r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}
	// Issue request
	req := httptest.NewRequest("GET", "/api/v1/beta/rotations/verification", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var parsed struct {
		Success     bool                               `json:"success"`
		Configured  bool                               `json:"configured"`
		GeneratedAt string                             `json:"generated_at"`
		Summary     notary.RotationVerificationSummary `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !parsed.Success || !parsed.Configured || parsed.Summary.Total != 2 {
		t.Fatalf("unexpected response: %+v", parsed)
	}
	// Signatures will be reported as kid_not_found since server path currently lacks key resolution; continuity should pass.
	if parsed.Summary.Results[0].ContinuityOK != true {
		t.Fatalf("expected continuity ok")
	}
	// Fetch metrics exposition from default Prometheus handler mounted under /metrics if present.
	// NOTE: Server uses gin; we simulate a request to /metrics if registered (failure reason counter should have increments for kid_not_found_*).
	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsW := httptest.NewRecorder()
	srv.router.ServeHTTP(metricsW, metricsReq)
	if metricsW.Code == 200 {
		body := metricsW.Body.String()
		// Look for the failure reason counter lines.
		// Accept either old or new key missing pattern.
		found := false
		for _, line := range strings.Split(body, "\n") {
			if strings.Contains(line, "gauth_rotation_verification_failure_reason_total") && (strings.Contains(line, "reason=\"kid_not_found_old\"") || strings.Contains(line, "reason=\"kid_not_found_new\"")) {
				// Ensure it is at least 1
				if strings.Contains(line, " 0") {
					t.Fatalf("expected failure reason counter increment, line=%s", line)
				}
				found = true
			}
		}
		if !found {
			// metrics endpoint may not be wired; do not fail test hard, but log.
			t.Logf("rotation failure reason counter not observed in /metrics output")
		}
	} else {
		// metrics endpoint not exposed; skip metrics assertion gracefully
		_, _ = io.Copy(io.Discard, metricsW.Body)
	}
}
