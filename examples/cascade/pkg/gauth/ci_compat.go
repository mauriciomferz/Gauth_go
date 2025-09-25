// Package gauth compatibility layer for CI/CD environments
// This file ensures all methods are properly exposed regardless of build environment
package gauth

// CICompatibility ensures all required methods are available in CI/CD
type CICompatibility struct{}

// EnsureValidateTokenAvailable provides a helper to verify ValidateToken method availability
func (c *CICompatibility) EnsureValidateTokenAvailable(auth *GAuth) bool {
	// This function exists purely to ensure the compiler recognizes ValidateToken
	_, err := auth.ValidateToken("test")
	return err != nil // We expect an error for test token, but method should exist
}

// EnsureTransactionDetailsAvailable verifies TransactionDetails type is accessible
func (c *CICompatibility) EnsureTransactionDetailsAvailable() bool {
	var _ TransactionDetails // This will fail to compile if TransactionDetails is not available
	return true
}

// ForceMethodResolution ensures all methods are properly resolved at build time
func init() {
	// This init function forces the compiler to resolve all method signatures
	var auth *ServiceAuth
	if auth != nil {
		_, _ = auth.ValidateToken("") // Force method resolution, ignore result
	}
}
