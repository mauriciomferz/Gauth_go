package web

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityAnchorEndpointSignatureTamper ensures modifying artifact bytes invalidates signature verification.
func TestCapabilityAnchorEndpointSignatureTamper(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "1")
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-tamper-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err2 := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err2 != nil {
		t.Fatalf("write caps file: %v", err2)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	srv := NewBetaServer(":0")
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	w := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/material")
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool            `json:"success"`
		Configured bool            `json:"configured"`
		Emitted    bool            `json:"emitted"`
		Artifact   json.RawMessage `json:"artifact"`
	}
	if err2 := json.Unmarshal(w.Body.Bytes(), &resp); err2 != nil {
		t.Fatalf("unmarshal resp: %v", err2)
	}
	if !resp.Success || !resp.Configured || !resp.Emitted {
		t.Fatalf("unexpected flags")
	}
	var wrapper struct {
		Artifact  json.RawMessage `json:"artifact"`
		Kid       string          `json:"kid"`
		Signature string          `json:"signature"`
		Mode      string          `json:"mode"`
	}
	if err2 := json.Unmarshal(resp.Artifact, &wrapper); err2 != nil {
		t.Fatalf("unmarshal wrapper: %v", err2)
	}
	if wrapper.Mode != sigModeEdDSA {
		t.Fatalf("expected mode eddsa")
	}
	sigBytes, err := base64.RawStdEncoding.DecodeString(wrapper.Signature)
	if err != nil {
		t.Fatalf("signature decode: %v", err)
	}
	// Decode public key from endpoint for consistency
	pkResp := performRequest(srv.router, "GET", "/api/v1/beta/keys/eddsa")
	if pkResp.Code != 200 {
		t.Fatalf("pk status=%d", pkResp.Code)
	}
	var pk struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Kid        string `json:"kid"`
		PublicKey  string `json:"public_key"`
	}
	if err := json.Unmarshal(pkResp.Body.Bytes(), &pk); err != nil {
		t.Fatalf("unmarshal pk: %v", err)
	}
	if !pk.Success || !pk.Configured || pk.PublicKey == "" {
		t.Fatalf("public key response invalid")
	}
	pubBytes, err := base64.RawStdEncoding.DecodeString(pk.PublicKey)
	if err != nil {
		t.Fatalf("pub decode: %v", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		t.Fatalf("pub size mismatch")
	}
	pubKey := ed25519.PublicKey(pubBytes)
	if !ed25519.Verify(pubKey, wrapper.Artifact, sigBytes) {
		t.Fatalf("baseline verification failed")
	}
	// Tamper: modify artifact bytes (flip a character) and expect verification failure.
	tampered := make([]byte, len(wrapper.Artifact))
	copy(tampered, wrapper.Artifact)
	// Find a position to mutate (skip first '{').
	for i := 1; i < len(tampered); i++ {
		if tampered[i] != '}' {
			tampered[i] ^= 0x01
			break
		}
	}
	if ed25519.Verify(pubKey, tampered, sigBytes) {
		t.Fatalf("verification unexpectedly succeeded after tamper")
	}
}
