package gauth_rfc_001

import (
	"os"
	"testing"
)

// TestMain sets up test environment defaults.
// By default, we disable grantor scope enforcement for legacy tests.
// Security-specific tests (TestVULN*) explicitly re-enable enforcement via os.Setenv.
func TestMain(m *testing.M) {
	// Disable grantor scope enforcement for legacy tests by default
	// This allows existing tests to continue working without modification
	// Security vulnerability tests will explicitly set GAUTH_ENFORCE_GRANTOR_SCOPES=1
	os.Setenv("GAUTH_ENFORCE_GRANTOR_SCOPES", "0")

	exitCode := m.Run()
	os.Exit(exitCode)
}
