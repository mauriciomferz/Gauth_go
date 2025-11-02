package web

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cryptoInt "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	"github.com/gin-gonic/gin"
)

// TestAttestationIntegrity_Success verifies successful verification path with valid Ed25519 signature.
func TestAttestationIntegrity_Success(t *testing.T) {
    gin.SetMode(gin.TestMode)
    s := NewBetaServer("")
    // Register ephemeral key into global registry for test
    pub, priv, _ := ed25519.GenerateKey(nil)
    // Construct temporary manager with single active key if Manager type exposes a simple constructor; fall back to manual minimal struct if not.
    // Use NewManager from internal crypto; rotate once to ensure an active key present then overwrite active key with our generated pair.
    m, _ := cryptoInt.NewManager(1 * time.Hour)
    if ak := m.Active(); ak != nil {
        ak.Private = priv
        ak.Public = pub
        ak.ID = "test-key"
    }
    cryptoInt.GlobalEdDSARegistry = m
    // Build unsigned portion
    payload := struct {
        Success    bool   `json:"success"`
        Configured bool   `json:"configured"`
        Nonce      string `json:"nonce"`
        Snapshot   struct {
            Hash        string `json:"hash"`
            GeneratedAt string `json:"generated_at"`
        } `json:"snapshot"`
        SigKid    string `json:"sig_kid"`
        SigMode   string `json:"sig_mode"`
        Signature string `json:"signature"`
    }{Success: true, Configured: true, Nonce: "n1"}
    payload.Snapshot.Hash = "sha256:deadbeef"
    payload.Snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
    payload.SigKid = "test-key"
    payload.SigMode = sigModeEdDSA
    // Reconstruct unsigned struct for signing (mirror server logic ordering)
    unsigned := struct {
        Success       bool   `json:"success"`
        Configured    bool   `json:"configured"`
        Reason        string `json:"reason,omitempty"`
        Nonce         string `json:"nonce,omitempty"`
        Snapshot      struct {
            Hash        string `json:"hash"`
            GeneratedAt string `json:"generated_at"`
        } `json:"snapshot"`
        Audit         *struct {
            HeadHash string `json:"head_hash"`
            Entries  int    `json:"entries"`
        } `json:"audit,omitempty"`
        Anchor        *struct {
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
    }{Success: payload.Success, Configured: payload.Configured, Nonce: payload.Nonce, Snapshot: payload.Snapshot, StrictUnknown: false}
    raw, _ := json.Marshal(unsigned)
    // Domain-separated signing (must match server verify path)
    msg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
    sig := ed25519.Sign(priv, msg)
    payload.Signature = base64.RawStdEncoding.EncodeToString(sig)
    body, _ := json.Marshal(payload)
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/model/limits/attestation/verify", http.NoBody)
    req.Body = io.NopCloser(bytes.NewReader(body))
    s.router.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("status %d body=%s", w.Code, w.Body.String())
    }
    var resp struct { Success bool `json:"success"`; Valid bool `json:"valid"`; Kid string `json:"kid"` }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil { t.Fatalf("unmarshal: %v", err) }
    if !resp.Success || !resp.Valid || resp.Kid != "test-key" { t.Fatalf("unexpected resp (expected valid signature): %+v", resp) }
}

// TestAttestationIntegrity_SignatureInvalid exercises invalid signature path with RFC tagged error.
func TestAttestationIntegrity_SignatureInvalid(t *testing.T) {
    gin.SetMode(gin.TestMode)
    s := NewBetaServer("")
    pub, priv, _ := ed25519.GenerateKey(nil)
    m2, _ := cryptoInt.NewManager(1 * time.Hour)
    if ak := m2.Active(); ak != nil {
        ak.Private = priv
        ak.Public = pub
        ak.ID = "test-key2"
    }
    cryptoInt.GlobalEdDSARegistry = m2
    unsigned := struct {
        Success       bool   `json:"success"`
        Configured    bool   `json:"configured"`
        Reason        string `json:"reason,omitempty"`
        Snapshot      struct {
            Hash        string `json:"hash"`
            GeneratedAt string `json:"generated_at"`
        } `json:"snapshot"`
        Audit         *struct {
            HeadHash string `json:"head_hash"`
            Entries  int    `json:"entries"`
        } `json:"audit,omitempty"`
        Anchor        *struct {
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
    }{Success: true, Configured: true, StrictUnknown: false}
    raw, _ := json.Marshal(unsigned)
    // Domain-separated signing, then tamper a byte to force invalid signature
    msg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
    sig := ed25519.Sign(priv, msg)
    sig[0] ^= 0xFF
    payload := struct {
        Success    bool   `json:"success"`
        Configured bool   `json:"configured"`
        Snapshot   struct {
            Hash        string `json:"hash"`
            GeneratedAt string `json:"generated_at"`
        } `json:"snapshot"`
        SigKid    string `json:"sig_kid"`
        SigMode   string `json:"sig_mode"`
        Signature string `json:"signature"`
    }{Success: true, Configured: true, SigKid: "test-key2", SigMode: sigModeEdDSA}
    payload.Snapshot.Hash = "sha256:beadfeed"
    payload.Snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
    payload.Signature = base64.RawStdEncoding.EncodeToString(sig)
    body, _ := json.Marshal(payload)
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/model/limits/attestation/verify", http.NoBody)
    req.Body = io.NopCloser(bytes.NewReader(body))
    s.router.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String()) }
    // New semantics: 200 with success=true valid=false error signature_invalid (backward compat envelope)
    var resp struct {
        Success bool `json:"success"`
        Valid   bool `json:"valid"`
        Kid     string `json:"kid"`
        Error   string `json:"error"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil { t.Fatalf("unmarshal: %v", err) }
    if !resp.Success || resp.Valid || resp.Kid != "test-key2" || resp.Error != "signature_invalid" {
        t.Fatalf("unexpected invalid signature resp: %+v", resp)
    }
}
