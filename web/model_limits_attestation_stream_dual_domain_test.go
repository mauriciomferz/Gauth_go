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

	internalCrypto "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// TestModelLimitsAttestationStreamDualDomainSignature subscribes to the SSE stream and validates
// both primary and domain signatures on the initial emitted attestation. The stream path may
// inject a Reason field post-sign; we reconstruct unsigned bytes excluding signature fields and
// omitting Reason to mirror the server's signed form.
func TestModelLimitsAttestationStreamDualDomainSignature(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	t.Setenv("GAUTH_ATTEST_DOMAIN_PREFIX", "EXTRA_ATTEST:")
	t.Setenv("GAUTH_ATTEST_STREAM_ENABLE", "1")
	t.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")
	// minimal audit/anchor/limits data
	auditFile, _ := os.CreateTemp(t.TempDir(), "audit_*.jsonl")
	anchorFile, _ := os.CreateTemp(t.TempDir(), "anchor_*.jsonl")
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	_, _ = limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
	limitsFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH", anchorFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "1")

	km, _ := internalCrypto.NewManager(1 * time.Hour)
	srv := NewBetaServer("", WithKeyProvider(km))
	t.Cleanup(func() { srv.Shutdown() })
	if km.Active() == nil {
		t.Fatalf("expected eddsa key active")
	}
	active := km.Active()

	// Generate a couple exceed events before streaming so snapshot/audit not empty
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 10})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		time.Sleep(5 * time.Millisecond)
	}

	// Subscribe via internal channel to avoid blocking HTTP streaming semantics.
	ch := srv.subscribeAttestation()
	defer srv.unsubscribeAttestation(ch)
	// Manually emit an attestation (reason=test)
	srv.emitAttestation("test")
	var attVal modelLimitsAttestation
	select {
	case attVal = <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for attestation on channel")
	}

	// Serialize full attestation to reuse same reconstruction code style as HTTP path test.
	bFull, _ := json.Marshal(attVal)
	attJSON := string(bFull)

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
		Nonce         string `json:"nonce"`
		Success       bool   `json:"success"`
		Configured    bool   `json:"configured"`
		Reason        string `json:"reason"`
		StrictUnknown bool   `json:"strict_unknown"`
	}
	if err := json.Unmarshal([]byte(attJSON), &att); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if att.Signature == "" || att.DomainSignature == "" || att.DomainPrefix != "EXTRA_ATTEST:" {
		t.Fatalf("missing domain signature fields: %+v", att)
	}

	// Rebuild unsigned struct (exclude signature fields + domain prefix/signatures). Exclude Reason (added post-sign).
	unsigned := struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
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
		Notarization  *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
	}{Success: att.Success, Configured: att.Configured, Nonce: att.Nonce, Snapshot: att.Snapshot, Audit: att.Audit, Anchor: att.Anchor, StrictUnknown: att.StrictUnknown, Notarization: att.Notarization}
	raw, _ := json.Marshal(unsigned)

	primarySig, _ := base64.RawStdEncoding.DecodeString(att.Signature)
	domainSig, _ := base64.RawStdEncoding.DecodeString(att.DomainSignature)
	primaryMsg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
	if !ed25519.Verify(active.Public, primaryMsg, primarySig) {
		t.Fatalf("primary signature invalid")
	}
	domainMsg := append([]byte(att.DomainPrefix), raw...)
	if !ed25519.Verify(active.Public, domainMsg, domainSig) {
		t.Fatalf("domain signature invalid")
	}
}
