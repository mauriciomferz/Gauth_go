// Package errors provides custom error types for GAuth
package errors

import (
	"fmt"
	"time"
)

// ErrorCode represents the type of error that occurred
type ErrorCode int

const (
	// Authentication related errors
	ErrInvalidToken ErrorCode = iota + 100
	ErrTokenExpired
	ErrInvalidGrant

	// Authorization related errors
	ErrUnauthorized
	ErrInsufficientScope
	ErrRateLimitExceeded

	// Resource related errors
	ErrResourceNotFound
	ErrResourceUnavailable

	// Configuration related errors
	ErrInvalidConfig
	ErrMissingConfig

	// Internal errors
	ErrInternalError
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

	// AdditionalInfo contains any extra information
	AdditionalInfo map[string]interface{}
}

// Error represents a GAuth error with context
type Error struct {
	Code    ErrorCode
	Message string
	Details *ErrorDetails
}

func (e *Error) Error() string {
	if e.Details != nil && len(e.Details.AdditionalInfo) > 0 {
		return fmt.Sprintf("%s (code: %d, details: %v)", e.Message, e.Code, e.Details.AdditionalInfo)
	}
	return fmt.Sprintf("%s (code: %d)", e.Message, e.Code)
}

// New creates a new error with the given code and message
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]interface{}),
		},
	}
}

// WithDetails adds context details to the error
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

// WithAdditionalInfo adds additional information to the error details
func (e *Error) WithAdditionalInfo(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]interface{}),
		}
	}

	e.Details.AdditionalInfo[key] = value
	return e
}

// IsAuthError returns true if the error is authentication related
func IsAuthError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code >= 100 && e.Code < 200
	}
	return false
}

// IsResourceError returns true if the error is resource related
func IsResourceError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code >= 300 && e.Code < 400
	}
	return false
}
