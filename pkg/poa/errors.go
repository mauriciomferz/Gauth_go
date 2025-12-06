// Package poa provides error types for module independence.
// These definitions allow pkg/poa to be extracted as a standalone module
// without depending on pkg/errors.
package poa

import (
	"fmt"
)

// Error codes used by the PoA package.
const (
	ErrCodeValidation   = "validation_error"
	ErrCodeUnauthorized = "unauthorized"
	ErrCodeExpiredToken = "expired_token"
	ErrCodeNotFound     = "not_found"
	ErrCodeInternal     = "internal_error"
)

// PoAError represents an error with a code and message.
type PoAError struct {
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *PoAError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *PoAError) Unwrap() error {
	return e.Cause
}

// NewError creates a new PoAError with the given code and message.
func NewError(code, message string) *PoAError {
	return &PoAError{Code: code, Message: message}
}

// WrapError creates a new PoAError wrapping an existing error.
func WrapError(code, message string, cause error) *PoAError {
	return &PoAError{Code: code, Message: message, Cause: cause}
}
