package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityAnchorStatusLastWriteUnix verifies the status endpoint includes a unix epoch counter.
func TestCapabilityAnchorStatusLastWriteUnix(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "0")
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-unix-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	// 1m interval ensures second reload is skipped but timestamp retained
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err != nil {
		t.Fatalf("write caps file: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	srv := NewBetaServer(":0")
	// Emit then skip
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	w := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/status")
	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success       bool   `json:"success"`
		Configured    bool   `json:"configured"`
		LastWrite     string `json:"last_write"`
		LastWriteUnix uint64 `json:"last_write_unix"`
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
	if resp.LastWriteUnix == 0 {
		t.Fatalf("last_write_unix zero")
	}
	// Freshness check: unix should be within +/-5s of current time
	now := time.Now().Unix()
	//nolint:gosec // G115: test code, timestamp comparison
	if diff := now - int64(resp.LastWriteUnix); diff < -5 || diff > 5 {
		t.Fatalf("last_write_unix freshness diff=%d now=%d recorded=%d", diff, now, resp.LastWriteUnix)
	}
}
