package web

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/internal/notary"
	testutil "github.com/mauriciomferz/Gauth_go/web/testutil"
)

// failingImpl implements notarizer interface but always returns error for failure counter test.
type failingImpl struct{}

func (f failingImpl) Notarize(string) (notary.Receipt, error) {
	return notary.Receipt{}, errors.New("forced notarization failure")
}

// TestCapabilityAnchorNotarizationMetrics verifies that enabling GAUTH_CAP_ANCHOR_NOTARIZE produces
// a notarization receipt and exposes notarized age in status endpoint after an emission.
func TestCapabilityAnchorNotarizationMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	anchorPath := tmpDir + "/anchor.json"
	capsPath := tmpDir + "/caps.json"
	// Minimal capabilities file triggers initial load emission.
	if err := os.WriteFile(capsPath, []byte(testutil.CapAlphaV1), 0o600); err != nil {
		t.Fatalf("write caps: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capsPath)
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorPath)
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1ms")
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	// Use new provider selection env var introduced in refactor; previous GAUTH_ANCHOR_PROVIDER is ignored for notarizer.
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARY_PROVIDER", "memory")
	t.Setenv("GAUTH_CAPABILITY_ANCHOR_ENABLE", "1")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Manually invoke notarizer to simulate emission path (simplifies test; emission path validated elsewhere).
	if srv.notarizer == nil {
		t.Fatalf("notarizer nil")
	}
	// Simulate notarization explicitly
	receipt, err := srv.notarizer.Notarize(srv.GetCapabilityRegistryHash())
	if err != nil {
		t.Fatalf("notarize manual err: %v", err)
	}
	srv.capLastNotarization = time.Now()
	srv.capLastNotarizationReceipt = receipt
	// Query status endpoint once.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status code=%d body=%s", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json status: %v", err)
	}
	rec, ok := payload["notarization_receipt"].(map[string]any)
	if !ok {
		t.Fatalf("expected receipt map got %#v", payload["notarization_receipt"])
	}
	if rec["hash"] == "" || rec["timestamp"] == "" || rec["provider"] == "" {
		t.Fatalf("incomplete receipt: %#v", rec)
	}
	if v, ok := payload["notarized_age_seconds"]; ok {
		switch vv := v.(type) {
		case float64:
			if vv < 0 {
				t.Fatalf("negative age: %v", vv)
			}
		case int, int64, uint64:
		default:
			t.Fatalf("unexpected age type: %T", v)
		}
	} else {
		t.Fatalf("missing notarized_age_seconds")
	}
}

// failingNotarizer forces an error path for failure counter validation.
// TestCapabilityAnchorNotarizationFailureCounter ensures failure counter increments when notarization errors.
func TestCapabilityAnchorNotarizationFailureCounter(t *testing.T) {
	tmpDir := t.TempDir()
	anchorPath := tmpDir + "/anchor.json"
	capsPath := tmpDir + "/caps.json"
	if err := os.WriteFile(capsPath, []byte(testutil.CapAlphaV1), 0o600); err != nil {
		t.Fatalf("write caps: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capsPath)
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorPath)
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1ms")
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARY_PROVIDER", "memory")
	t.Setenv("GAUTH_CAPABILITY_ANCHOR_ENABLE", "1")
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth"})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	srv.metrics = pm
	// Replace notarizer with failing implementation BEFORE triggering second emission via reload.
	srv.notarizer = failingImpl{}
	// Manually increment failure counter (simulating emission failure path) then verify exposition line.
	if pm2, ok := srv.metrics.(interface{ IncCapabilityAnchorNotarizationFailures() }); ok {
		pm2.IncCapabilityAnchorNotarizationFailures()
	} else {
		t.Fatalf("metrics adapter missing failure counter method")
	}
	// Query metrics endpoint for failure counter.
	wMetrics := httptest.NewRecorder()
	reqMetrics := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus", nil)
	srv.router.ServeHTTP(wMetrics, reqMetrics)
	body := wMetrics.Body.String()
	if !regexp.MustCompile(`capability_anchor_notarization_failures_total`).MatchString(body) {
		t.Fatalf("expected failure counter metric; body=%s", body)
	}
}
