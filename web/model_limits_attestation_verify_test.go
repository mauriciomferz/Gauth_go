package web

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	internalCrypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
)

// TestModelLimitsAttestationVerify exercises the verification endpoint with a valid and tampered attestation.
func TestModelLimitsAttestationVerify(t *testing.T) {
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
	if internalCrypto.GlobalEdDSARegistry == nil || internalCrypto.GlobalEdDSARegistry.Active() == nil {
		t.Fatalf("eddsa registry inactive")
	}

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
	var att map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal att: %v", err)
	}
	sig := att["signature"].(string)
	kid := att["sig_kid"].(string)
	if sig == "" || kid == "" {
		t.Fatalf("signature fields missing")
	}

	// Post to verify endpoint
	attBytes := w2.Body.Bytes()
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", bytes.NewReader(attBytes))
	req3.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("verify status=%d", w3.Code)
	}
	var vresp struct {
		Success  bool   `json:"success"`
		Valid    bool   `json:"valid"`
		Kid      string `json:"kid"`
		Combined string `json:"combined_hash"`
	}
	if err := json.Unmarshal(w3.Body.Bytes(), &vresp); err != nil {
		t.Fatalf("unmarshal verify: %v body=%s", err, w3.Body.String())
	}
	if !vresp.Success || !vresp.Valid || vresp.Kid != kid || vresp.Combined == "" {
		t.Fatalf("verify response invalid %+v", vresp)
	}

	// Tamper snapshot hash and expect invalid
	var attStruct struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Snapshot   struct {
			Hash        string `json:"hash"`
			GeneratedAt string `json:"generated_at"`
		} `json:"snapshot"`
		Audit *struct {
			HeadHash string `json:"head_hash"`
			Entries  int    `json:"entries"`
		} `json:"audit,omitempty"`
		Anchor *struct {
			LatestHash string `json:"latest_hash"`
			Entries    int    `json:"entries"`
			Interval   int    `json:"interval"`
		} `json:"anchor,omitempty"`
		StrictUnknown bool   `json:"strict_unknown"`
		Signature     string `json:"signature"`
		SigKid        string `json:"sig_kid"`
		SigMode       string `json:"sig_mode"`
	}
	if err := json.Unmarshal(attBytes, &attStruct); err != nil {
		t.Fatalf("unmarshal struct: %v", err)
	}
	attStruct.Snapshot.Hash += "tamper"
	tamperedBytes, _ := json.Marshal(attStruct)
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPost, "/api/v1/model/limits/attestation/verify", bytes.NewReader(tamperedBytes))
	req4.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w4, req4)
	var vresp2 struct {
		Success bool   `json:"success"`
		Valid   bool   `json:"valid"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(w4.Body.Bytes(), &vresp2); err != nil {
		t.Fatalf("unmarshal tampered verify: %v body=%s", err, w4.Body.String())
	}
	if !vresp2.Success || vresp2.Valid {
		t.Fatalf("expected invalid signature: %+v", vresp2)
	}
}

// TestModelLimitsAttestationKeys validates keys endpoint output.
func TestModelLimitsAttestationKeys(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	srv := NewBetaServer("")
	if internalCrypto.GlobalEdDSARegistry == nil || internalCrypto.GlobalEdDSARegistry.Active() == nil {
		t.Fatalf("eddsa registry inactive")
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation/keys", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
	var resp struct {
		Success bool `json:"success"`
		Keys    []struct {
			Kid       string `json:"kid"`
			PublicB64 string `json:"public_b64"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal keys: %v", err)
	}
	if !resp.Success || len(resp.Keys) == 0 || resp.Keys[0].Kid == "" || resp.Keys[0].PublicB64 == "" {
		t.Fatalf("keys response invalid %+v", resp)
	}
	// Basic public key size check
	pb, err := base64.RawStdEncoding.DecodeString(resp.Keys[0].PublicB64)
	if err != nil || len(pb) != ed25519.PublicKeySize {
		t.Fatalf("public key decode/size invalid")
	}
}
