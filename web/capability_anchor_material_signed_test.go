package web

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityAnchorMaterialSigned verifies signature wrapper emission when GAUTH_CAP_ANCHOR_SIGN=1 and EdDSA key manager active.
func TestCapabilityAnchorMaterialSigned(t *testing.T) {
	// Ensure EdDSA mode for active key manager.
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "1")
	// Create anchor file path.
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-signed-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	// Force short interval (>=1m required by server init parse guard, so use 1m and rely on first-load unconditional write).
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	// Provide capabilities file (single capability) to trigger file-backed path.
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err2 := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err2 != nil {
		t.Fatalf("write caps file: %v", err2)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)

	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Perform explicit reload (idempotent) to make sure emission executed.
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")

	// Read anchor file.
	raw, err := os.ReadFile(anchorFile.Name())
	if err != nil {
		t.Fatalf("read anchor file: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("expected non-empty anchor artifact")
	}
	// Unmarshal wrapper and validate fields.
	var wrapper struct {
		Artifact  json.RawMessage `json:"artifact"`
		Kid       string          `json:"kid"`
		Signature string          `json:"signature"`
		Mode      string          `json:"mode"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		t.Fatalf("unmarshal wrapper: %v body=%s", err, string(raw))
	}
	if wrapper.Mode != "eddsa" {
		t.Fatalf("expected mode=eddsa got %s", wrapper.Mode)
	}
	if wrapper.Kid == "" {
		t.Fatalf("kid empty")
	}
	if wrapper.Signature == "" {
		t.Fatalf("signature empty")
	}
	// Signature should decode base64 without error.
	if _, err := base64.RawStdEncoding.DecodeString(wrapper.Signature); err != nil {
		t.Fatalf("signature base64 decode: %v", err)
	}
	// Parse inner artifact and check required fields.
	var artifact struct {
		Type          string `json:"type"`
		RegistryHash  string `json:"registry_hash"`
		SchemaVersion int    `json:"schema_version"`
		AnchoredAt    string `json:"anchored_at"`
	}
	if err := json.Unmarshal(wrapper.Artifact, &artifact); err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}
	if artifact.Type != testCapRegistryAnchor {
		t.Fatalf("unexpected artifact type %s", artifact.Type)
	}
	if artifact.RegistryHash == "" {
		t.Fatalf("registry_hash empty")
	}
	if artifact.SchemaVersion != 1 {
		t.Fatalf("schema_version expected 1 got %d", artifact.SchemaVersion)
	}
	if artifact.AnchoredAt == "" {
		t.Fatalf("anchored_at empty")
	}
}

const testCapRegistryAnchor = "capability_registry_anchor"
