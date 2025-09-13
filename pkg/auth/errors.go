package auth

import "fmt"

// Error types for GAuth authorization
type ErrorCode string

const (
	// Identity verification errors
	ErrInvalidIdentity ErrorCode = "invalid_identity"
	ErrIdentityExpired ErrorCode = "identity_expired"
	ErrIdentityRevoked ErrorCode = "identity_revoked"

	// Authorization errors
	ErrNotAuthorized        ErrorCode = "not_authorized"
	ErrAuthorizationExpired ErrorCode = "authorization_expired"
	ErrAuthorizationRevoked ErrorCode = "authorization_revoked"

	// Attestation errors
	ErrInvalidAttestation ErrorCode = "invalid_attestation"
	ErrAttestationExpired ErrorCode = "attestation_expired"
	ErrMissingAttestation ErrorCode = "missing_attestation"

	// Compliance errors
	ErrRuleViolation   ErrorCode = "rule_violation"
	ErrPolicyViolation ErrorCode = "policy_violation"
	ErrScopeExceeded   ErrorCode = "scope_exceeded"

	// Registration errors
	ErrInvalidRegistry  ErrorCode = "invalid_registry"
	ErrRegistryNotFound ErrorCode = "registry_not_found"
	ErrInvalidDocument  ErrorCode = "invalid_document"
)

// AuthError represents a GAuth-specific error
type AuthError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *AuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAuthError(code ErrorCode, message string, cause error) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsAuthError checks if an error is a specific auth error code
func IsAuthError(err error, code ErrorCode) bool {
	if authErr, ok := err.(*AuthError); ok {
		return authErr.Code == code
	}
	return false
}
