package testutil

import (
	"os"
	"time"

	internalCrypto "github.com/mauriciomferz/Gauth_go/internal/crypto"
)

// UnsetCryptoEnv clears environment variables that can mutate global crypto or signing behavior
// causing order-dependent test failures.
func UnsetCryptoEnv() {
	vars := []string{
		"GAUTH_EDDSA_PERSIST_PATH",
		"GAUTH_EDDSA_ROTATION_LOG",
		"GAUTH_EDDSA_AUTO_ROTATE",
		"GAUTH_EDDSA_ROTATE_INTERVAL",
		"GAUTH_ROTATIONS_SIGN",
		"GAUTH_ROTATIONS_MULTISIG",
		"GAUTH_ROTATIONS_THRESHOLD",
		"GAUTH_ANCHOR_ROTATIONS",
		"GAUTH_CAP_ANCHOR_NOTARIZE",
		"GAUTH_CAP_ANCHOR_SIGN",
		"GAUTH_TOKEN_SIG_MODE",
		"GAUTH_ATTEST_DOMAIN_PREFIX",
	}
	for _, k := range vars {
		_ = os.Unsetenv(k)
	}
}

// FreshKeyManager replaces the global EdDSA registry with a new manager instance.
// Use before tests needing clean signing state.
func FreshKeyManager(ttl time.Duration) error {
	m, err := internalCrypto.NewManager(ttl)
	if err != nil {
		return err
	}
	internalCrypto.GlobalEdDSARegistry = m
	return nil
}

// DisableKeyRegistry sets the global registry to nil for negative tests.
func DisableKeyRegistry() { internalCrypto.GlobalEdDSARegistry = nil }
