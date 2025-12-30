package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestCapabilityAnchorStatusEndpoint verifies status endpoint fields and metrics exposure.
func TestCapabilityAnchorStatusEndpoint(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "0")
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-status-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err != nil {
		t.Fatalf("write caps file: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Trigger emission then a skip
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	w := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/status")
	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success      bool   `json:"success"`
		Configured   bool   `json:"configured"`
		LastWrite    string `json:"last_write"`
		Emitted      uint64 `json:"emitted_total"`
		Skipped      uint64 `json:"skipped_total"`
		HashChanged  uint64 `json:"hash_changed_total"`
		RegistryHash string `json:"registry_hash"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured {
		t.Fatalf("unexpected flags configured=%v", resp.Configured)
	}
	if resp.LastWrite == "" {
		t.Fatalf("last_write empty")
	}
	if resp.Emitted < 1 {
		t.Fatalf("emitted_total <1")
	}
	if resp.Skipped < 1 {
		t.Fatalf("skipped_total <1")
	}
	if resp.RegistryHash == "" {
		t.Fatalf("registry_hash empty")
	}
}
