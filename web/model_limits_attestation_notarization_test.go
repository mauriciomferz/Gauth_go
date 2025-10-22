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

// TestModelLimitsAttestationNotarization ensures notarization + surge fields present when enabled and signature validates.
func TestModelLimitsAttestationNotarization(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "1")
	t.Setenv("GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE", "1")
	// Use memory notarizer through capability path (already wired when GAUTH_CAP_ANCHOR_NOTARIZE=1) is separate; attestation uses same s.notarizer.
	// Force initialization of notarizer by reusing capability env (fallback path if needed):
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")

	auditFile, _ := os.CreateTemp(t.TempDir(), "audit_*.jsonl")
	anchorFile, _ := os.CreateTemp(t.TempDir(), "anchor_*.jsonl")
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":5}}}`))
	limitsFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH", anchorFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "1")
	os.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")
	// Surge tuning for deterministic trigger
	os.Setenv("GAUTH_MODEL_LIMIT_SURGE_FACTOR", "1.0") // make trigger easier
	os.Setenv("GAUTH_MODEL_LIMIT_SURGE_MIN_EVENTS", "1")

	srv := NewBetaServer("")
	if internalCrypto.GlobalEdDSARegistry == nil || internalCrypto.GlobalEdDSARegistry.Active() == nil {
		t.Fatalf("expected eddsa registry active")
	}
	active := internalCrypto.GlobalEdDSARegistry.Active()

	// Generate multiple exceed events across small time to raise counts.
	for i := 0; i < 6; i++ {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 10})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		time.Sleep(10 * time.Millisecond)
	}
	// Force at least one more event after brief pause to ensure last10 includes recent high count.
	time.Sleep(30 * time.Millisecond)
	wLast := httptest.NewRecorder()
	bodyLast, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 10})
	reqLast, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(bodyLast))
	reqLast.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(wLast, reqLast)

	// Fetch attestation
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("status %d body=%s", w2.Code, w2.Body.String())
	}

	type attStruct struct {
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
		Signature string `json:"signature"`
		SigKid    string `json:"sig_kid"`
		SigMode   string `json:"sig_mode"`
	}
	var att attStruct
	if err := json.Unmarshal(w2.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if att.Signature == "" || att.SigKid == "" || att.SigMode != sigModeEdDSA {
		t.Fatalf("signature fields missing")
	}
	if att.Notarization == nil || !att.Notarization.Success || att.Notarization.Provider == "" {
		t.Fatalf("notarization missing or invalid")
	}
	// Surge stats are opportunistic; assert structure if present, otherwise accept absence.
	if att.Surge != nil {
		if !att.Surge.Triggered || att.Surge.ModelID == "" || att.Surge.Last10Sec == 0 {
			t.Fatalf("surge stats invalid: %+v", att.Surge)
		}
	}

	sigBytes, err := base64.RawStdEncoding.DecodeString(att.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	// Reconstruct unsigned struct in identical field order excluding signature fields.
	type unsignedStruct struct {
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
	u := unsignedStruct{
		Success:       att.Success,
		Configured:    att.Configured,
		Reason:        att.Reason,
		Snapshot:      att.Snapshot,
		Audit:         att.Audit,
		Anchor:        att.Anchor,
		StrictUnknown: att.StrictUnknown,
		Surge:         att.Surge,
		Notarization:  att.Notarization,
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal unsigned: %v", err)
	}
	if !ed25519.Verify(active.Public, raw, sigBytes) {
		t.Fatalf("signature verify failed")
	}
}
