package testutil

import (
	"os"
)

// UnsetCryptoEnv clears environment variables that can mutate global crypto or signing behavior
// causing order-dependent test failures.
func UnsetCryptoEnv() {
	vars := []string{
		"AGENTAUTH_EDDSA_PERSIST_PATH",
		"AGENTAUTH_EDDSA_ROTATION_LOG",
		"AGENTAUTH_EDDSA_AUTO_ROTATE",
		"AGENTAUTH_EDDSA_ROTATE_INTERVAL",
		"AGENTAUTH_ROTATIONS_SIGN",
		"AGENTAUTH_ROTATIONS_MULTISIG",
		"AGENTAUTH_ROTATIONS_THRESHOLD",
		"AGENTAUTH_ANCHOR_ROTATIONS",
		"AGENTAUTH_CAP_ANCHOR_NOTARIZE",
		"AGENTAUTH_CAP_ANCHOR_SIGN",
		"AGENTAUTH_TOKEN_SIG_MODE",
		"AGENTAUTH_ATTEST_DOMAIN_PREFIX",
	}
	for _, k := range vars {
		_ = os.Unsetenv(k)
	}
}
