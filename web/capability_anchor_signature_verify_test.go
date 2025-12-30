package web

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	internalCrypto "github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestCapabilityAnchorEndpointSignatureVerification exercises full client-side verification:
// 1. Configure server for signed anchor emission.
// 2. Trigger capability reload + anchor emission.
// 3. Fetch /api/v1/beta/capabilities/anchor/material endpoint.
// 4. Extract wrapper, verify signature over inner artifact bytes with active public key.
func TestCapabilityAnchorEndpointSignatureVerification(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "1")
	// Anchor artifact path
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-endpoint-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m") // first load emits regardless
	// Capabilities file to ensure file-backed loader path.
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err2 := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err2 != nil {
		t.Fatalf("write caps file: %v", err2)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	// Disable background polls for deterministic test timing.
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")

	// Create manager with active key
	m, _ := internalCrypto.NewManager(1 * time.Hour)
	ak := m.Active()
	var pub ed25519.PublicKey
	var kid string
	if ak != nil {
		pub = ak.Public
		kid = ak.ID
	}
	srv := NewBetaServer(":0", WithKeyProvider(m))
	t.Cleanup(func() { srv.Shutdown() })
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")

	// Fetch endpoint material
	w := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/material")
	if w.Code != 200 {
		t.Fatalf("endpoint status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool            `json:"success"`
		Configured bool            `json:"configured"`
		Emitted    bool            `json:"emitted"`
		Artifact   json.RawMessage `json:"artifact"`
	}
	if err2 := json.Unmarshal(w.Body.Bytes(), &resp); err2 != nil {
		t.Fatalf("unmarshal resp: %v body=%s", err2, w.Body.String())
	}
	if !resp.Success || !resp.Configured || !resp.Emitted {
		t.Fatalf("flags unexpected success=%v configured=%v emitted=%v", resp.Success, resp.Configured, resp.Emitted)
	}
	// Unmarshal wrapper
	// Endpoint returns either unsigned artifact object or signed wrapper. Attempt to parse wrapper; if fields absent treat as failure for this test.
	var wrapper struct {
		Artifact  json.RawMessage `json:"artifact"`
		Kid       string          `json:"kid"`
		Signature string          `json:"signature"`
		Mode      string          `json:"mode"`
	}
	if err2 := json.Unmarshal(resp.Artifact, &wrapper); err2 != nil {
		t.Fatalf("unmarshal wrapper: %v artifact=%s", err2, string(resp.Artifact))
	}
	if wrapper.Mode != sigModeEdDSA {
		t.Fatalf("expected mode=eddsa got %s", wrapper.Mode)
	}
	if wrapper.Kid == "" {
		t.Fatalf("kid empty")
	}
	if wrapper.Signature == "" {
		t.Fatalf("signature empty")
	}
	sigBytes, err := base64.RawStdEncoding.DecodeString(wrapper.Signature)
	if err != nil {
		t.Fatalf("signature base64 decode: %v", err)
	}
	if pub == nil || kid == "" {
		t.Fatalf("captured active key nil")
	}
	if kid != wrapper.Kid {
		t.Fatalf("kid mismatch captured=%s wrapper=%s", kid, wrapper.Kid)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("public key size invalid")
	}
	if !ed25519.Verify(pub, wrapper.Artifact, sigBytes) {
		t.Fatalf("signature verification failed")
	}
	// Unmarshal inner artifact for structural assertions.
	var inner struct {
		Type          string `json:"type"`
		RegistryHash  string `json:"registry_hash"`
		SchemaVersion int    `json:"schema_version"`
		AnchoredAt    string `json:"anchored_at"`
	}
	if err := json.Unmarshal(wrapper.Artifact, &inner); err != nil {
		t.Fatalf("unmarshal inner artifact: %v", err)
	}
	if inner.Type != "capability_registry_anchor" {
		t.Fatalf("inner type mismatch %s", inner.Type)
	}
	if inner.RegistryHash == "" {
		t.Fatalf("inner registry_hash empty")
	}
	if inner.SchemaVersion != 1 {
		t.Fatalf("inner schema_version expected 1 got %d", inner.SchemaVersion)
	}
	if inner.AnchoredAt == "" {
		t.Fatalf("inner anchored_at empty")
	}
}
