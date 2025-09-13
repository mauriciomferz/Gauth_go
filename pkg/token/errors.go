//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package token

import "errors"

// Common errors returned by token operations
var (
	// ErrTokenNotFound indicates the requested token does not exist
	ErrTokenNotFound = errors.New("token not found")

	// ErrTokenExpired indicates the token has passed its expiration time
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenNotYetValid indicates the token is not yet valid (before nbf)
	ErrTokenNotYetValid = errors.New("token not yet valid")

	// ErrTokenRevoked indicates the token has been explicitly revoked
	ErrTokenRevoked = errors.New("token revoked")

	// ErrTokenBlacklisted indicates the token is in the blacklist
	ErrTokenBlacklisted = errors.New("token is blacklisted")

	// ErrInvalidToken indicates the token fails basic validation
	ErrInvalidToken = errors.New("invalid token")

	// ErrInvalidSignature indicates token signature verification failed
	ErrInvalidSignature = errors.New("invalid token signature")

	// ErrInvalidClaims indicates the token claims are invalid
	ErrInvalidClaims = errors.New("invalid token claims")

	// ErrInsufficientScope indicates the token lacks required scopes
	ErrInsufficientScope = errors.New("insufficient token scope")

	// ErrStorageFailure indicates a storage backend operation failed
	ErrStorageFailure = errors.New("token storage failure")

	// ErrInvalidConfig indicates invalid configuration was provided
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrInvalidIssuer indicates the token issuer is not allowed
	ErrInvalidIssuer = errors.New("invalid token issuer")

	// ErrInvalidAudience indicates the token audience is not allowed
	ErrInvalidAudience = errors.New("invalid token audience")

	// ErrInvalidType indicates the wrong type of token was provided
	ErrInvalidType = errors.New("invalid token type")

	// ErrMissingClaims indicates required claims are missing
	ErrMissingClaims = errors.New("missing required claims")
)

// ValidationErrorCode type for standardized validation error codes
type ValidationErrorCode string

const (
	// ValidationCodeExpired indicates token has expired
	ValidationCodeExpired ValidationErrorCode = "expired"

	// ValidationCodeNotFound indicates token was not found
	ValidationCodeNotFound ValidationErrorCode = "not_found"

	// ValidationCodeInvalid indicates token is invalid
	ValidationCodeInvalid ValidationErrorCode = "invalid"

	// ValidationCodeRevoked indicates token was revoked
	ValidationCodeRevoked ValidationErrorCode = "revoked"

	// ValidationCodeBlacklisted indicates token is blacklisted
	ValidationCodeBlacklisted ValidationErrorCode = "blacklisted"

	// ValidationCodeNotYetValid indicates token is not yet valid
	ValidationCodeNotYetValid ValidationErrorCode = "not_yet_valid"

	// ValidationCodeInvalidAudience indicates invalid audience
	ValidationCodeInvalidAudience ValidationErrorCode = "invalid_audience"

	// ValidationCodeInvalidIssuer indicates invalid issuer
	ValidationCodeInvalidIssuer ValidationErrorCode = "invalid_issuer"

	// ValidationCodeInvalidType indicates wrong token type
	ValidationCodeInvalidType ValidationErrorCode = "invalid_type"

	// ValidationCodeInvalidSignature indicates signature validation failed
	ValidationCodeInvalidSignature ValidationErrorCode = "invalid_signature"

	// ValidationCodeInvalidClaims indicates invalid token claims
	ValidationCodeInvalidClaims ValidationErrorCode = "invalid_claims"

	// ValidationCodeInsufficientScope indicates insufficient scopes
	ValidationCodeInsufficientScope ValidationErrorCode = "insufficient_scope"

	// ValidationCodeMissingClaims indicates missing required claims
	ValidationCodeMissingClaims ValidationErrorCode = "missing_claims"

	// ValidationCodeStorageFailure indicates storage operation failed
	ValidationCodeStorageFailure ValidationErrorCode = "storage_failure"

	// ValidationCodeInvalidConfig indicates invalid configuration
	ValidationCodeInvalidConfig ValidationErrorCode = "invalid_config"
)

// ValidationError represents a token validation failure with a specific code
type ValidationError struct {
	// Code identifies the type of validation failure
	Code ValidationErrorCode

	// Message provides details about the failure
	Message string

	// Err is the underlying error if any
	Err error
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error with the given code and message
func NewValidationError(code ValidationErrorCode, msg string) *ValidationError {
	return &ValidationError{
		Code:    code,
		Message: msg,
	}
}

// NewValidationErrorWithCause creates a validation error with an underlying cause
func NewValidationErrorWithCause(code ValidationErrorCode, msg string, err error) *ValidationError {
	return &ValidationError{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

// Is implements error interface for error wrapping
func (e *ValidationError) Is(target error) bool {
	t, ok := target.(*ValidationError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}
