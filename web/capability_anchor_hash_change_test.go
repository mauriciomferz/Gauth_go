package web

import (
	"os"
	"path/filepath"
	"testing"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/web/testutil"
)

// TestCapabilityAnchorHashChange triggers a semantic hash change by modifying capability set
// and asserts capability_registry_hash_changed_total increments.
func TestCapabilityAnchorHashChange(t *testing.T) {
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "0")
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-hash-change-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	// Base capabilities file (single capability)
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err != nil {
		t.Fatalf("write caps file: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	mem, ok := srv.metrics.(*imetrics.Memory)
	if !ok {
		t.Fatalf("metrics not memory")
	}
	initial := mem.CapabilityRegistryHashChanged()
	// Modify capabilities to force hash change (add new capability + mapping)
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferAuditV1), 0o600); err != nil {
		t.Fatalf("rewrite caps file: %v", err)
	}
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	after := mem.CapabilityRegistryHashChanged()
	if after <= initial {
		t.Fatalf("expected hash_changed counter to increment initial=%d after=%d", initial, after)
	}
}
