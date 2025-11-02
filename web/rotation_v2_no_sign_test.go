package web

import (
  "os"
  "path/filepath"
  "testing"
)

// TestRotationV2NoSigning ensures when signing disabled (no SIGN, no AUTO_GEN, no imported keys) we get zero verified weight and threshold not met.
func TestRotationV2NoSigning(t *testing.T) {
  cfgSrc := filepath.Join("..", "config", "multisig_weights.json")
  b, err := os.ReadFile(cfgSrc)
  if err != nil { t.Fatalf("read config: %v", err) }
  tmpDir := t.TempDir()
  cfgPath := filepath.Join(tmpDir, "weights.json")
  if err := os.WriteFile(cfgPath, b, 0o600); err != nil { t.Fatalf("write temp config: %v", err) }
  t.Setenv("GAUTH_ROTATIONS_V2_CONFIG", cfgPath)
  // Explicitly ensure these are unset
  t.Setenv("GAUTH_ROTATIONS_V2_SIGN", "0")
  t.Setenv("GAUTH_ROTATIONS_V2_AUTO_GEN", "0")
  srv := NewBetaServer("")
  art, verified, perAlg, failures, err := srv.buildAndOptionallySignRotationV2()
  if err != nil { t.Fatalf("build error: %v", err) }
  if verified != 0 { t.Fatalf("expected verified 0 got %d", verified) }
  if len(perAlg) != 0 { t.Fatalf("expected empty perAlg got %v", perAlg) }
  if art.ThresholdWeight <= 0 { t.Fatalf("threshold invalid %d", art.ThresholdWeight) }
  if verified >= art.ThresholdWeight { t.Fatalf("unexpected threshold_met when signatures disabled") }
  if len(failures) != 0 { t.Fatalf("no failures expected, got %v", failures) }
}
