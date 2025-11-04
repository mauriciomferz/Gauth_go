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
	"time"

	internalCrypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
)

// TestModelLimitsAttestationDualDomainSignature ensures both primary and domain signatures verify
// when GAUTH_ATTEST_DOMAIN_PREFIX is configured.
func TestModelLimitsAttestationDualDomainSignature(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	t.Setenv("GAUTH_ATTEST_DOMAIN_PREFIX", "EXTRA_ATTEST:")
	// minimal files
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
		t.Fatalf("expected eddsa registry active")
	}
	active := internalCrypto.GlobalEdDSARegistry.Active()

	// Produce some audit events (exceed) to ensure snapshot not empty
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 10})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		time.Sleep(5 * time.Millisecond)
	}

	// Fetch attestation
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("status %d body=%s", w2.Code, w2.Body.String())
	}

	var att struct {
		Signature       string `json:"signature"`
		DomainSignature string `json:"domain_signature"`
		DomainPrefix    string `json:"domain_prefix"`
		SigKid          string `json:"sig_kid"`
		Snapshot        struct {
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
		Notarization *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
		Nonce      string `json:"nonce"`
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if att.Signature == "" || att.DomainSignature == "" || att.DomainPrefix != "EXTRA_ATTEST:" {
		t.Fatalf("domain signature fields missing: %+v", att)
	}

	// Reconstruct unsigned payload excluding signature fields using struct to mirror ordering.
	unsigned := struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Nonce      string `json:"nonce,omitempty"`
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
		StrictUnknown bool `json:"strict_unknown"`
		Surge         *struct {
			ModelID   string  `json:"model_id"`
			Last10Sec int     `json:"last_10s_exceed_events"`
			AvgActive float64 `json:"avg_active_seconds"`
			Factor    float64 `json:"factor"`
			MinEvents int     `json:"min_events"`
			Triggered bool    `json:"triggered"`
			At        string  `json:"triggered_at,omitempty"`
		} `json:"surge,omitempty"`
		Notarization *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
	}{Success: att.Success, Configured: att.Configured, Reason: "", Nonce: att.Nonce, Snapshot: att.Snapshot, Audit: att.Audit, Anchor: att.Anchor, StrictUnknown: true, Surge: nil, Notarization: att.Notarization}
	raw, _ := json.Marshal(unsigned)
	// Raw bytes length intentionally not logged (noise); if debugging needed, reintroduce logging locally.
	// Do NOT canonicalize here; server signs the direct struct serialization ordering.

	primarySig, _ := base64.RawStdEncoding.DecodeString(att.Signature)
	domainSig, _ := base64.RawStdEncoding.DecodeString(att.DomainSignature)
	// Primary signature is over the constant AttestationDomainPrefix.
	primaryMsg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
	if !ed25519.Verify(active.Public, primaryMsg, primarySig) {
		t.Fatalf("primary signature invalid")
	}
	// Domain signature is over the env-provided prefix.
	domainMsg := append([]byte(att.DomainPrefix), raw...)
	if !ed25519.Verify(active.Public, domainMsg, domainSig) {
		t.Fatalf("domain signature invalid")
	}
}
