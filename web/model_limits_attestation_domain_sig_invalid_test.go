package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestModelLimitsAttestationDomainSignatureInvalid ensures that when the domain signature is tampered
// the verification endpoint returns domain_signature_invalid (soft invalid) while primary remains valid.
func TestModelLimitsAttestationDomainSignatureInvalid(t *testing.T) {
    t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
    t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
    t.Setenv("GAUTH_ATTEST_DOMAIN_PREFIX", "EXTRA_ATTEST:")
    limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
    _, _ = limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
    limitsFile.Close()
    t.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())

    srv := NewBetaServer("")
    // Fetch attestation with dual signatures
    w := httptest.NewRecorder()
    req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
    srv.router.ServeHTTP(w, req)
    if w.Code != 200 { t.Fatalf("attestation status %d body=%s", w.Code, w.Body.String()) }
    var att map[string]any
    if err := json.Unmarshal(w.Body.Bytes(), &att); err != nil { t.Fatalf("unmarshal att: %v", err) }
    ds, _ := att["domain_signature"].(string)
    if ds == "" { t.Skip("domain signature absent (prefix not configured)") }
    // Tamper domain signature: decode, flip a bit, re-encode
    raw, err := base64.RawStdEncoding.DecodeString(ds)
    if err != nil { t.Fatalf("decode domain sig: %v", err) }
    if len(raw) > 0 { raw[0] ^= 0xFF }
    att["domain_signature"] = base64.RawStdEncoding.EncodeToString(raw)
    // Marshal mutated attestation
    mutated, _ := json.Marshal(att)
    w2 := httptest.NewRecorder()
    req2, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", nil)
    req2.Body = io.NopCloser(bytes.NewReader(mutated))
    req2.Header.Set("Content-Type", "application/json")
    srv.router.ServeHTTP(w2, req2)
    if w2.Code != 200 { t.Fatalf("expected 200 soft invalid path got %d body=%s", w2.Code, w2.Body.String()) }
    var v map[string]any
    if err := json.Unmarshal(w2.Body.Bytes(), &v); err != nil { t.Fatalf("unmarshal verify: %v", err) }
    if v["error"] != "domain_signature_invalid" { t.Fatalf("expected error domain_signature_invalid got %+v", v) }
    if valid, _ := v["valid"].(bool); valid { t.Fatalf("expected valid=false got %+v", v) }
}