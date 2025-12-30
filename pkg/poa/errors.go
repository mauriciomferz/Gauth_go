package poa

import (
	"fmt"

	"github.com/mauriciomferz/AgentAuth/pkg/rfc/errs"
)

// Error codes mapped to RFC definitions.
// We maintain these constants for backward compatibility but alias them where possible
// or ensure they map cleanly to errs.ErrorCode values.
const (
	ErrCodeValidation   = string(errs.ErrInvalidRequest)
	ErrCodeUnauthorized = string(errs.ErrUnauthorized)
	ErrCodeExpiredToken = string(errs.ErrExpired)
	ErrCodeNotFound     = string(errs.ErrNotFound)
	ErrCodeInternal     = string(errs.ErrInternal)
)

// PoAError represents an error with a code and message.
// It implements the standard error interface.
type PoAError struct {
	Code    string // maps to errs.ErrorCode
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

// ToRFCError converts the internal PoAError to a standardized RFCError.
func (e *PoAError) ToRFCError() errs.RFCError {
	return errs.RFCError{
		Code:    errs.ErrorCode(e.Code),
		Message: e.Message,
	}
}

// AsRFCError implements a convention for typed error conversion.
func (e *PoAError) AsRFCError() errs.RFCError {
	return e.ToRFCError()
}

// NewError creates a new PoAError with the given code and message.
func NewError(code, message string) *PoAError {
	return &PoAError{Code: code, Message: message}
}

// WrapError creates a new PoAError wrapping an existing error.
func WrapError(code, message string, cause error) *PoAError {
	return &PoAError{Code: code, Message: message, Cause: cause}
}
