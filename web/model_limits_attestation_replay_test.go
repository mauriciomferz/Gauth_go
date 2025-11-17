package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestModelLimitsAttestationReplay ensures a second verification attempt with identical nonce triggers replay error.
func TestModelLimitsAttestationReplay(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	auditFile, _ := os.CreateTemp(t.TempDir(), "audit_*.jsonl")
	anchorFile, _ := os.CreateTemp(t.TempDir(), "anchor_*.jsonl")
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	_, _ = limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
	limitsFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH", anchorFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "1")

	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// Generate exceed event to populate audit/anchor
	w1 := httptest.NewRecorder()
	body1, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 10})
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w1, req1)

	// Fetch attestation
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("attestation status=%d", w2.Code)
	}

	// First verify (should succeed)
	attBytes := w2.Body.Bytes()
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", bytes.NewReader(attBytes))
	req3.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("first verify status=%d body=%s", w3.Code, w3.Body.String())
	}

	// Second verify (replay of same nonce) should fail with replay code
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", bytes.NewReader(attBytes))
	req4.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w4, req4)
	if w4.Code == 200 {
		t.Fatalf("expected replay failure, got 200 body=%s", w4.Body.String())
	}
	if !bytes.Contains(w4.Body.Bytes(), []byte("attestation_nonce_replay")) {
		t.Fatalf("expected attestation_nonce_replay error; body=%s", w4.Body.String())
	}
}
