package web

import (
	"regexp"
	"testing"
)

// TestCapabilityAnchorPrometheusExposition ensures capability anchor metrics endpoint exposes expected lines.
func TestCapabilityAnchorPrometheusExposition(t *testing.T) {
	t.Setenv("GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS", "10")
	// Force a last write by simulating capability load (server initialization seeds static capabilities and may emit artifact if path configured).
	// We set file path env to trigger emission.
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", t.TempDir()+"/anchor.json")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	// Basic expected metric names.
	must := []string{
		"gauth_capability_anchor_age_seconds",
		"gauth_capability_anchor_stale",
		"gauth_capability_anchor_emitted_total",
		"gauth_capability_anchor_skipped_total",
		"gauth_capability_registry_hash_changed_total",
	}
	for _, m := range must {
		if !regexp.MustCompile(m + " ").MatchString(body) {
			t.Fatalf("expected metric line for %s in exposition:\n%s", m, body)
		}
	}
}
