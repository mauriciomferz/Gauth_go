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

// TestModelLimitsAttestationSignature verifies optional signing of the attestation payload.
func TestModelLimitsAttestationSignature(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")

	// Prepare temp config + audit/anchor paths
	auditFile, _ := os.CreateTemp(t.TempDir(), "audit_*.jsonl")
	anchorFile, _ := os.CreateTemp(t.TempDir(), "anchor_*.jsonl")
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	_, _ = limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
	limitsFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH", anchorFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "1")
	os.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")

	srv := NewBetaServer("")
	if internalCrypto.GlobalEdDSARegistry == nil || internalCrypto.GlobalEdDSARegistry.Active() == nil {
		t.Fatalf("expected active eddsa registry")
	}
	active := internalCrypto.GlobalEdDSARegistry.Active()
	pub := active.Public
	kid := active.ID

	// Trigger exceed to populate audit + anchor
	w1 := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 10})
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w1, req1)

	// Fetch attestation
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("status %d body=%s", w2.Code, w2.Body.String())
	}

	// Define struct mirroring server representation (including signature fields)
	type attStruct struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Nonce      string `json:"nonce"`
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
		Signature     string `json:"signature,omitempty"`
		SigKid        string `json:"sig_kid,omitempty"`
		SigMode       string `json:"sig_mode,omitempty"`
	}

	var att attStruct
	if err := json.Unmarshal(w2.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal attestation: %v", err)
	}
	if att.Signature == "" || att.SigKid == "" || att.SigMode != sigModeEdDSA {
		t.Fatalf("signature fields missing or invalid: %+v", att)
	}
	if att.SigKid != kid {
		t.Fatalf("kid mismatch expected %s got %s", kid, att.SigKid)
	}
	sigBytes, err := base64.RawStdEncoding.DecodeString(att.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	// Reconstruct unsigned canonical JSON
	unsigned := att
	unsigned.Signature = ""
	unsigned.SigKid = ""
	unsigned.SigMode = ""
	raw, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf("marshal unsigned: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("public key size bad")
	}
	// Domain-separated message must match server signing (prefix + canonical JSON)
	prefix := []byte("GAUTH_MODEL_LIMIT_ATTEST:")
	msg := append(prefix, raw...)
	if !ed25519.Verify(pub, msg, sigBytes) {
		t.Fatalf("signature verify failed (domain-separated)")
	}
	// Tamper check
	unsigned.Snapshot.Hash += testTamper
	tamperedRaw, _ := json.Marshal(unsigned)
	tamperedMsg := append(prefix, tamperedRaw...)
	if ed25519.Verify(pub, tamperedMsg, sigBytes) {
		t.Fatalf("tamper should fail verification (domain-separated)")
	}
}

const testTamper = "tamper"
