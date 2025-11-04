package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestModelLimitsAttestationDomainSignaturePrefixMissing ensures missing domain_prefix triggers soft invalid.
func TestModelLimitsAttestationDomainSignaturePrefixMissing(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	t.Setenv("GAUTH_ATTEST_DOMAIN_PREFIX", "EXTRA_ATTEST:")
	lf, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	_, _ = lf.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
	lf.Close()
	t.Setenv("GAUTH_MODEL_LIMITS_PATH", lf.Name())
	srv := NewBetaServer("")
	// Fetch attestation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("attestation status %d body=%s", w.Code, w.Body.String())
	}
	var att map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal att: %v", err)
	}
	if att["domain_signature"] == nil {
		t.Skip("domain signature absent")
	}
	// Remove domain_prefix field only
	delete(att, "domain_prefix")
	mutated, _ := json.Marshal(att)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", nil)
	req2.Body = io.NopCloser(bytes.NewReader(mutated))
	req2.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", w2.Code, w2.Body.String())
	}
	var v map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &v); err != nil {
		t.Fatalf("unmarshal verify: %v", err)
	}
	if v["error"] != "domain_signature_prefix_missing" {
		t.Fatalf("expected domain_signature_prefix_missing got %+v", v)
	}
	if valid, _ := v["valid"].(bool); valid {
		t.Fatalf("expected valid=false got %+v", v)
	}
}
