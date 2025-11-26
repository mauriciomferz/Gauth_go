package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mauriciomferz/Gauth_go/web/testutil"
)

// TestCapabilityAnchorMaterial verifies that enabling anchor file emission writes an artifact
// and the /api/v1/beta/capabilities/anchor/material endpoint returns structured content.
func TestCapabilityAnchorMaterial(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "cap-anchor-*.json")
	if err != nil {
		t.Fatalf("temp file create: %v", err)
	}
	tmp.Close()
	// Use t.Setenv to ensure environment isolation (automatically restored after test) to
	// prevent leakage influencing subsequent tests in this package.
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", tmp.Name())
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1ms") // force immediate write
	// Sign if EdDSA active (optional) - do not enforce present
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "0")
	// Provide a capabilities file to ensure loader path executes anchor emission logic (file-backed scenario).
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err2 := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err2 != nil {
		t.Fatalf("write capabilities file: %v", err2)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Anchor artifact might emit only after file load; perform explicit reload to ensure loader executed (idempotent)
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read anchor file: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("expected anchor artifact bytes >0 after reload")
	}
	// Call endpoint
	w := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/material")
	if w.Code != 200 {
		t.Fatalf("unexpected status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success      bool   `json:"success"`
		Configured   bool   `json:"configured"`
		Emitted      bool   `json:"emitted"`
		RegistryHash string `json:"registry_hash"`
		Artifact     any    `json:"artifact"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v body=%s", err, w.Body.String())
	}
	if !resp.Success || !resp.Configured || !resp.Emitted {
		t.Fatalf("unexpected flags success=%v configured=%v emitted=%v", resp.Success, resp.Configured, resp.Emitted)
	}
	if resp.RegistryHash == "" {
		t.Fatalf("registry_hash empty")
	}
	// Basic structural assertion: artifact should decode as map
	if _, ok := resp.Artifact.(map[string]any); !ok {
		t.Fatalf("artifact not object: %#v", resp.Artifact)
	}
}
