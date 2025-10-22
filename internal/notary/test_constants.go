package notary

// Test-only constants used for tamper scenarios to avoid goconst duplication warnings.
const (
	// Tamper hex short invalid root used to force mismatch in CLI tests.
	tamperHexShort = "deadbeef"
)
