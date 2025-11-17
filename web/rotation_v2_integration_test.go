package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRotationV2SigningIntegration exercises buildAndOptionallySignRotationV2 via server instance
// using auto-generated ephemeral keys. It validates that signatures attach and verification weight
// meets threshold and per-algorithm map is populated.
func TestRotationV2SigningIntegration(t *testing.T) {
	// Copy config/multisig_weights.json to a temp file to avoid mutation or external coupling.
	// Resolve config path relative to repository root. When test package cwd is web/, climb one level.
	cfgSrc := filepath.Join("..", "config", "multisig_weights.json")
	if _, statErr := os.Stat(cfgSrc); statErr != nil {
		// Fallback: attempt absolute path from PROJECT_ROOT env if provided.
		if root := os.Getenv("PROJECT_ROOT"); root != "" {
			cfgSrc = filepath.Join(root, "config", "multisig_weights.json")
		}
	}
	b, err := os.ReadFile(cfgSrc)
	if err != nil {
		t.Fatalf("read config source (%s): %v", cfgSrc, err)
	}
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "weights.json")
	if err2 := os.WriteFile(cfgPath, b, 0o600); err2 != nil {
		t.Fatalf("write temp config: %v", err2)
	}

	t.Setenv("GAUTH_ROTATIONS_V2_CONFIG", cfgPath)
	t.Setenv("GAUTH_ROTATIONS_V2_AUTO_GEN", "1")
	t.Setenv("GAUTH_ROTATIONS_V2_SIGN", "1")

	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	art, verified, perAlg, failures, err := srv.buildAndOptionallySignRotationV2()
	if err != nil {
		t.Fatalf("buildAndOptionallySignRotationV2 error: %v", err)
	}
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
	if verified < art.ThresholdWeight {
		t.Fatalf("verified weight %d below threshold %d", verified, art.ThresholdWeight)
	}
	if len(perAlg) == 0 || perAlg["ED25519"] == 0 {
		t.Fatalf("perAlg map missing ED25519 weight: %+v", perAlg)
	}
	// Ensure every configured signer acquired signature.
	missing := []string{}
	for _, s := range art.Signers {
		if s.Signature == "" {
			missing = append(missing, s.ID)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("signatures missing for signers: %v", missing)
	}
	// Marshal to ensure JSON tags intact (no panic).
	if _, mErr := json.Marshal(art); mErr != nil {
		t.Fatalf("marshal artifact failed: %v", mErr)
	}
}
