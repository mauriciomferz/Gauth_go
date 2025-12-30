package web

import (
	"os"
)

func init() {
	// Disable Redis and background polls by default for all tests in the web package to ensure they are hermetic.
	// Individual tests that specifically need these can override them.
	if os.Getenv("AGENTAUTH_SKIP_REDIS") == "" {
		os.Setenv("AGENTAUTH_SKIP_REDIS", "1")
	}
	if os.Getenv("AGENTAUTH_DISABLE_BG_POLLS") == "" {
		os.Setenv("AGENTAUTH_DISABLE_BG_POLLS", "1")
	}
}
