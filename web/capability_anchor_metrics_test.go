package web

import (
	"os"
	"path/filepath"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityAnchorMetrics verifies metrics counters increment for emitted vs skipped scenarios.
func TestCapabilityAnchorMetrics(t *testing.T) {
	// Prepare environment for signed emission to exercise full path (signature optional for metrics semantics).
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "0")
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-metrics-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	// Interval 1m ensures first load emits then second reload is skipped (throttled).
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err != nil {
		t.Fatalf("write caps file: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	// Disable background polls for determinism.
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	srv := NewBetaServer(":0")
	// First reload triggers emission (metrics emitted++)
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	// Second reload should be skipped due to interval throttle (metrics skipped++)
	_ = performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	// Access internal memory metrics via type assertion.
	mem, ok := srv.metrics.(*imetrics.Memory)
	if !ok {
		t.Fatalf("metrics collector not memory type")
	}
	// Reflect emitted/skipped counters using unexported fields through accessor pattern: not available yet, so approximate via anchorAttempts vs emitted/skipped fields if exported.
	// Since memory.go does not yet expose accessors for new counters, we rely on unsafe reflection only if necessary. Prefer adding simple exported accessors if test fails.
	// For minimal intrusion, assert at least one emitted and one skipped by reading file timestamps & size heuristics.
	// Assertions: Expect at least 1 emission and at least 1 skip.
	if mem.CapabilityAnchorEmitted() < 1 {
		t.Fatalf("expected >=1 capability anchor emitted counter got %d", mem.CapabilityAnchorEmitted())
	}
	if mem.CapabilityAnchorSkipped() < 1 {
		t.Fatalf("expected >=1 capability anchor skipped counter got %d", mem.CapabilityAnchorSkipped())
	}
	// Registry hash changed counter increments only on semantic change after initial load; not exercised in this test.
}
