package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	cryptopkg "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// TestCapabilityAnchorAlgorithmMetrics verifies per-algorithm capability anchor emission counters
// increment on initial capability load + anchor artifact emission.
func TestCapabilityAnchorAlgorithmMetrics(t *testing.T) {
	tmp := t.TempDir()
	capPath := filepath.Join(tmp, "caps.json")
	// Minimal valid capability file
	capJSON := `{"schema_version":1,"capabilities":[{"id":"cap.test","version":"1.0","stable":true}],"action_mappings":{}}`
	if err := os.WriteFile(capPath, []byte(capJSON), 0o600); err != nil {
		t.Fatalf("write cap file: %v", err)
	}
	anchorPath := filepath.Join(tmp, "anchor_artifact.json")
	// Environment configuration required for anchor emission & signing
	t.Setenv("GAUTH_CAPABILITIES_PATH", capPath)
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorPath)
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m") // minimum accepted
	t.Setenv("GAUTH_EDDSA_TTL_HOURS", "24")           // enable key manager
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "1")            // enable signing path
	// Ensure cleanup
	t.Cleanup(func() {
		os.Unsetenv("GAUTH_CAPABILITIES_PATH")
		os.Unsetenv("GAUTH_CAP_ANCHOR_FILE_PATH")
		os.Unsetenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL")
		os.Unsetenv("GAUTH_EDDSA_TTL_HOURS")
		os.Unsetenv("GAUTH_CAP_ANCHOR_SIGN")
	})
	m := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics("0", m)
	t.Cleanup(func() { srv.Shutdown() })
	// Perform a second explicit capability load now that anchor file path has been configured
	// so that emission block inside loadCapabilitiesFromFile observes non-empty capAnchorFilePath.
	if err := srv.capabilitiesHandler.LoadFromFile(capPath); err != nil {
		t.Fatalf("second capability load failed: %v", err)
	}
	time.Sleep(25 * time.Millisecond) // brief yield
	mem := m
	if mem.CapabilityAnchorEmitted() == 0 {
		t.Fatalf("expected at least one capability anchor emission")
	}
	snap := mem.SnapshotEx()
	algCounts := snap.CapabilityAnchorAlgorithmCounts
	if len(algCounts) == 0 {
		t.Fatalf("expected non-empty algorithm emission counts")
	}
	// Registered algorithms (from crypto/signature registry) should be attributed.
	// We lower-case the names in emission; ensure expected algorithms present.
	expectedPresent := map[string]bool{"Ed25519": false, "ecdsa-p256": false}
	for _, info := range cryptopkg.ListAlgorithms() { // sanity registry iteration
		if _, ok := expectedPresent[info.Name]; ok {
			if algCounts[info.Name] == 0 {
				t.Fatalf("expected non-zero count for algorithm %s", info.Name)
			}
			expectedPresent[info.Name] = true
		}
	}
	for k, v := range expectedPresent {
		if !v {
			t.Fatalf("expected algorithm facet %s to be present in counts; counts=%v", k, algCounts)
		}
	}
}
