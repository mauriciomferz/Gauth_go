package modellimits

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	internalCrypto "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// TestModelLimitsAttestationStreamDualDomainSignature subscribes to the SSE stream and validates
// both primary and domain signatures on the initial emitted attestation.
func TestModelLimitsAttestationStreamDualDomainSignature(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	t.Setenv("GAUTH_ATTEST_DOMAIN_PREFIX", "EXTRA_ATTEST:")
	t.Setenv("GAUTH_ATTEST_STREAM_ENABLE", "1")
	t.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")

	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	limitsFile.WriteString(`{"model_limits":{"demo":{"max_input_tokens":5}}}`)
	limitsFile.Close()

	auditFile, _ := os.CreateTemp(t.TempDir(), "audit_*.jsonl")
	auditFile.Close()
	anchorFile, _ := os.CreateTemp(t.TempDir(), "anchor_*.jsonl")
	anchorFile.Close()
	t.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "1")

	km, _ := internalCrypto.NewManager(1 * time.Hour)
	if km.Active() == nil {
		t.Fatalf("expected eddsa key active")
	}
	active := km.Active()

	h := NewHandler(limitsFile.Name(), auditFile.Name(), anchorFile.Name())
	h.KeyManager = km
	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	ch := h.SubscribeAttestation()
	defer h.UnsubscribeAttestation(ch)

	h.EmitAttestation("test")

	var attVal ModelLimitsAttestation
	select {
	case attVal = <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for attestation on channel")
	}

	// Verify signatures manually
	// Reconstruct unsigned object with identical field order
	type unsignedStructTest struct {
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
	}

	u := unsignedStructTest{
		Success: attVal.Success, Configured: attVal.Configured, Reason: attVal.Reason, Nonce: attVal.Nonce,
		Snapshot: attVal.Snapshot, Audit: attVal.Audit, Anchor: attVal.Anchor,
		StrictUnknown: attVal.StrictUnknown, Surge: attVal.Surge, Notarization: attVal.Notarization,
	}

	raw, _ := json.Marshal(u)
	primarySig, _ := base64.RawStdEncoding.DecodeString(attVal.Signature)
	domainSig, _ := base64.RawStdEncoding.DecodeString(attVal.DomainSignature)

	primaryMsg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
	if !ed25519.Verify(active.Public, primaryMsg, primarySig) {
		t.Fatalf("primary signature invalid")
	}

	domainMsg := append([]byte(attVal.DomainPrefix), raw...)
	if !ed25519.Verify(active.Public, domainMsg, domainSig) {
		t.Fatalf("domain signature invalid")
	}
}
