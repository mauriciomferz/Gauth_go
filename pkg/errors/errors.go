// Package errors provides structured error handling for GAuth
package errors

import (
	"fmt"
	"time"
)

// ErrorCode represents a specific error type
type ErrorCode string

// Error implements the error interface for ErrorCode
func (e ErrorCode) Error() string {
	return string(e)
}

// Predefined error codes
const (
	ErrTokenExpired           ErrorCode = "token_expired"
	ErrInvalidToken           ErrorCode = "invalid_token"
	ErrInsufficientScope      ErrorCode = "insufficient_scope"
	ErrRateLimited            ErrorCode = "rate_limited"
	ErrInvalidRequest         ErrorCode = "invalid_request"
	ErrInvalidClient          ErrorCode = "invalid_client"
	ErrInvalidGrant           ErrorCode = "invalid_grant"
	ErrUnauthorizedClient     ErrorCode = "unauthorized_client"
	ErrInvalidScope           ErrorCode = "invalid_scope"
	ErrServerError            ErrorCode = "server_error"
	ErrTemporarilyUnavailable ErrorCode = "temporarily_unavailable"

	// Token store related errors
	ErrMissingEncryptionKey ErrorCode = "missing_encryption_key"
	ErrMissingUserID        ErrorCode = "missing_user_id"
	ErrMissingClientID      ErrorCode = "missing_client_id"
	ErrMissingExpiry        ErrorCode = "missing_expiry"
)

// ErrorSource indicates where the error originated
type ErrorSource string

// Predefined error sources
const (
	SourceAuthentication ErrorSource = "authentication"
	SourceAuthorization  ErrorSource = "authorization"
	SourceToken          ErrorSource = "token"
	SourceStorage        ErrorSource = "storage"
	SourceRateLimiting   ErrorSource = "rate_limiting"
	SourceCircuitBreaker ErrorSource = "circuit_breaker"
	SourceValidation     ErrorSource = "validation"
	SourceProtocol       ErrorSource = "protocol"
	SourceResourceServer ErrorSource = "resource_server"
)

// ErrorDetails contains structured information about an error
type ErrorDetails struct {
	// Timestamp when the error occurred
	Timestamp time.Time

	// RequestID associated with the error
	RequestID string

	// ClientID involved in the error
	ClientID string

	// UserID affected by the error
	UserID string

	// ResourceID being accessed when error occurred
	ResourceID string

	// IPAddress of the client
	IPAddress string

	// Path being accessed
	Path string

	// Method being used (HTTP method, etc)
	Method string

	// HTTPStatusCode if applicable
	HTTPStatusCode int

	// AdditionalInfo contains any extra information
	AdditionalInfo map[string]string
}

// Error is a structured error type for GAuth
type Error struct {
	// Code identifies the error type
	Code ErrorCode

	// Message is a human-readable error message
	Message string

	// Source indicates where the error originated
	Source ErrorSource

	// Details contains additional error information
	Details *ErrorDetails

	// Cause is the underlying error
	Cause error
}

// New creates a new structured error
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]string),
		},
	}
}

// WithSource adds a source to the error
func (e *Error) WithSource(source ErrorSource) *Error {
	e.Source = source
	return e
}

// WithCause adds a cause to the error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details *ErrorDetails) *Error {
	if details != nil {
		// Preserve timestamp if not set in the new details
		if details.Timestamp.IsZero() && e.Details != nil {
			details.Timestamp = e.Details.Timestamp
		}
		e.Details = details
	}
	return e
}

// WithRequestInfo adds request info to the error details
func (e *Error) WithRequestInfo(requestID, clientID, userID string) *Error {
	if e.Details == nil {
		e.Details = &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]string),
		}
	}
	e.Details.RequestID = requestID
	e.Details.ClientID = clientID
	e.Details.UserID = userID
	return e
}

// WithHTTPInfo adds HTTP-specific info to the error
func (e *Error) WithHTTPInfo(path, method string, statusCode int, ipAddress string) *Error {
	if e.Details == nil {
		e.Details = &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]string),
		}
	}
	e.Details.Path = path
	e.Details.Method = method
	e.Details.HTTPStatusCode = statusCode
	e.Details.IPAddress = ipAddress
	return e
}

// AddInfo adds additional info to the error
func (e *Error) AddInfo(key, value string) *Error {
	if e.Details == nil {
		e.Details = &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]string),
		}
	}
	e.Details.AdditionalInfo[key] = value
	return e
}

// Error returns the error message
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
