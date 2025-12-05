package testutil

import (
	"os"
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
