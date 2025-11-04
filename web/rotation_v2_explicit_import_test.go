package web

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRotationV2ExplicitKeysImport verifies explicit private key import path produces signatures and verification.
func TestRotationV2ExplicitKeysImport(t *testing.T) {
	cfgSrc := filepath.Join("..", "config", "multisig_weights.json")
	b, err := os.ReadFile(cfgSrc)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "weights.json")
	if err := os.WriteFile(cfgPath, b, 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	// Generate deterministic private keys for each signer id.
	type signer struct{ id string }
	signers := []signer{{"hsm-a"}, {"soft-b"}, {"notary-c"}}
	kvPairs := make([]string, 0, len(signers))
	for _, s := range signers {
		// Use ed25519.GenerateKey for randomness (acceptable for test).
		_, priv, _ := ed25519.GenerateKey(nil)
		kvPairs = append(kvPairs, s.id+":"+base64.RawURLEncoding.EncodeToString(priv))
	}
	t.Setenv("GAUTH_ROTATIONS_V2_CONFIG", cfgPath)
	t.Setenv("GAUTH_ROTATIONS_V2_SIGN", "1")
	t.Setenv("GAUTH_ROTATIONS_V2_ED25519_KEYS", joinComma(kvPairs))

	srv := NewBetaServer("")
	art, verified, perAlg, failures, err := srv.buildAndOptionallySignRotationV2()
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
	total := 0
	for _, s := range art.Signers {
		total += s.Weight
	}
	if verified != total {
		t.Fatalf("expected verified=%d got %d", total, verified)
	}
	if perAlg["ED25519"] != total {
		t.Fatalf("perAlg mismatch: %v", perAlg)
	}
	for _, s := range art.Signers {
		if s.Signature == "" {
			t.Fatalf("missing signature for %s", s.ID)
		}
	}
	if _, mErr := json.Marshal(art); mErr != nil {
		t.Fatalf("marshal artifact failed: %v", mErr)
	}
}

// joinComma local helper to avoid importing strings just for Join.
func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
