package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestModelLimitsAttestationMutation mutates the nonce after signature issuance to force a signature mismatch
// while avoiding replay detection (nonce is new). Expect soft invalid signature_invalid.
func TestModelLimitsAttestationMutation(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	t.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")
	// minimal limits file to satisfy snapshot generation
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	_, _ = limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
	limitsFile.Close()
	t.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())

	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Acquire attestation via router (mirrors other tests)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var att map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sig, _ := att["signature"].(string)
	kid, _ := att["sig_kid"].(string)
	if sig == "" || kid == "" {
		t.Fatalf("missing signature fields: %+v", att)
	}
	originalNonce, _ := att["nonce"].(string)
	if originalNonce == "" {
		t.Skip("nonce not present (replay protection disabled)")
	}
	// Mutate nonce to a new value; keep signature (now invalid)
	att["nonce"] = "MUTATED-NONCE-XYZ"
	mutated, _ := json.Marshal(att)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", strings.NewReader(string(mutated)))
	req2.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 soft invalid path got %d body=%s", w2.Code, w2.Body.String())
	}
	var v map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &v); err != nil {
		t.Fatalf("verify decode: %v", err)
	}
	if suc, _ := v["success"].(bool); !suc {
		t.Fatalf("expected success true got %v", v)
	}
	if valid, _ := v["valid"].(bool); valid {
		t.Fatalf("expected valid false got %v", v)
	}
	if errStr, _ := v["error"].(string); errStr != "signature_invalid" {
		t.Fatalf("expected signature_invalid got %s full=%v", errStr, v)
	}
}
