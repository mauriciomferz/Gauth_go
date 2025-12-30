package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

func TestRevocationAnchoringPersistence(t *testing.T) {
	// 1. Setup temporary persistence file
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "anchor.json")

	// 2. Configure server with anchor persistence enabled
	t.Setenv("AGENTAUTH_ANCHOR_PERSIST_PATH", persistPath)
	t.Setenv("AGENTAUTH_REVOCATION_ENABLED", "1")
	t.Setenv("AGENTAUTH_DISABLE_BG_POLLS", "1")

	// Start server (initializes anchor client and hooks)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// Wait for async initialization if any (though NewBetaServer is synchronous for anchor init)

	// Ensure anchor client initialized
	initialCount := srv.anchorClient.TotalAnchors()
	t.Logf("Initial anchors: %d", initialCount)

	// 3. Trigger a revocation hook via direct append
	// (Simulates a successful revocation event handled by the system)
	ev := delegation.RevocationEvent{
		ID:           "rev-test-anchor-1",
		DelegationID: "del-123",
		Reason:       "compromise",
		RevokedAt:    time.Now().UTC(),
	}

	if _, err := srv.revocationChain.Append(ev); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// 5. Verify persistence file contains our hash
	// We iterate all anchors because startup capability anchor might race with us.
	content, err := os.ReadFile(persistPath)
	if err != nil {
		t.Fatalf("failed to read persist file: %v", err)
	}

	var diskData struct {
		Anchors []struct {
			Hash       string    `json:"hash"`
			AnchoredAt time.Time `json:"anchored_at"`
		} `json:"anchors"`
	}
	if err := json.Unmarshal(content, &diskData); err != nil {
		t.Fatalf("failed to parse persist file: %v", err)
	}

	targetHash := srv.revocationChain.AggregateHash()
	found := false
	for _, a := range diskData.Anchors {
		if a.Hash == targetHash {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("revocation aggregate hash %s not found in persistence file (found %d anchors)", targetHash, len(diskData.Anchors))
	} else {
		t.Logf("Successfully verified anchor %s in persistence file", targetHash)
	}
}
