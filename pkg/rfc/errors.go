package rfc

import "fmt"

// ErrorCode enumerates machine-readable RFC-related error identifiers.
type ErrorCode string

const (
	ErrNotFound            ErrorCode = "not_found"
	ErrUnauthorized        ErrorCode = "unauthorized"
	ErrRevoked             ErrorCode = "revoked"
	ErrExpired             ErrorCode = "expired"
	ErrScopeViolation      ErrorCode = "scope_violation"
	ErrRestrictionExceeded ErrorCode = "restriction_exceeded"
	ErrInvalidRequest      ErrorCode = "invalid_request"
	ErrIntegrityFailure    ErrorCode = "integrity_failure"
	ErrInternal            ErrorCode = "internal_error"
	ErrReplay              ErrorCode = "replay"
	ErrDelegationDepthExceeded ErrorCode = "delegation_depth_exceeded"
)

// RFCError wraps an error with a code and message.
type RFCError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e RFCError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// New creates an RFCError.
func New(code ErrorCode, msg string) error { return RFCError{Code: code, Message: msg} }
