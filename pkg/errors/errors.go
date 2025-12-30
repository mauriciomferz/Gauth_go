package errors

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	// Source constants for compatibility with examples
	SourceToken         = "token"
	SourceAuthorization = "authorization"
	SourceRateLimiting  = "rate_limiting"
	SourceStorage       = "storage" // legacy example expects this
	// Authentication errors
	ErrCodeUnauthenticated ErrorCode = "UNAUTHENTICATED"
	ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidToken    ErrorCode = "INVALID_TOKEN"
	ErrCodeExpiredToken    ErrorCode = "EXPIRED_TOKEN"
	// Legacy alias expectations
	ErrServerError  ErrorCode = "INTERNAL_ERROR" // alias used in examples
	ErrTokenExpired ErrorCode = "EXPIRED_TOKEN"  // alias used in examples

	// Validation errors
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrCodeMissingField   ErrorCode = "MISSING_FIELD"

	// System errors
	ErrCodeInternal  ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotFound  ErrorCode = "NOT_FOUND"
	ErrCodeConflict  ErrorCode = "CONFLICT"
	ErrCodeRateLimit ErrorCode = "RATE_LIMIT"
	ErrCodeTimeout   ErrorCode = "TIMEOUT"

	// Legacy/example specific codes
	ErrCodeInsufficientScope ErrorCode = "INSUFFICIENT_SCOPE"

	// Network errors
	ErrCodeNetworkError ErrorCode = "NETWORK_ERROR"
	ErrCodeServiceDown  ErrorCode = "SERVICE_DOWN"
)

// ErrorDetails represents structured error details
type ErrorDetails struct {
	Timestamp      time.Time              `json:"timestamp"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
	ClientID       string                 `json:"client_id,omitempty"`
	HTTPMethod     string                 `json:"http_method,omitempty"`
	HTTPPath       string                 `json:"http_path,omitempty"`
	HTTPStatusCode int                    `json:"http_status_code,omitempty"` // legacy compatibility
}

// AgentAuthError represents a structured error
type AgentAuthError struct {
	Code      ErrorCode     `json:"code"`
	Message   string        `json:"message"`
	Details   *ErrorDetails `json:"details,omitempty"`
	Source    string        `json:"source,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	File      string        `json:"file,omitempty"`
	Line      int           `json:"line,omitempty"`
}

// Error is a compatibility type alias expected by legacy examples.
type Error = AgentAuthError

// Error implements the error interface
func (e *AgentAuthError) Error() string {
	if e.File != "" && e.Line != 0 {
		return fmt.Sprintf("[%s] %s (%s:%d)", e.Code, e.Message, e.File, e.Line)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithDetails adds details to the error
func (e *AgentAuthError) WithDetails(details interface{}) *AgentAuthError {
	if e.Details == nil {
		e.Details = &ErrorDetails{Timestamp: e.Timestamp}
	}
	// Store in AdditionalInfo
	if e.Details.AdditionalInfo == nil {
		e.Details.AdditionalInfo = make(map[string]interface{})
	}
	e.Details.AdditionalInfo["details"] = details
	return e
}

// WithCause adds a cause to the error
func (e *AgentAuthError) WithCause(cause error) *AgentAuthError {
	if e.Details == nil {
		e.Details = &ErrorDetails{Timestamp: e.Timestamp}
	}
	if e.Details.AdditionalInfo == nil {
		e.Details.AdditionalInfo = make(map[string]interface{})
	}
	e.Details.AdditionalInfo["cause"] = cause.Error()
	return e
}

// WithFields adds fields to the error (accepts both map[string]interface{} and map[string]string)
func (e *AgentAuthError) WithFields(fields interface{}) *AgentAuthError {
	if e.Details == nil {
		e.Details = &ErrorDetails{Timestamp: e.Timestamp}
	}
	if e.Details.AdditionalInfo == nil {
		e.Details.AdditionalInfo = make(map[string]interface{})
	}

	// Handle map[string]interface{}
	if interfaceFields, ok := fields.(map[string]interface{}); ok {
		for k, v := range interfaceFields {
			e.Details.AdditionalInfo[k] = v
		}
	} else if stringFields, ok := fields.(map[string]string); ok {
		// Handle map[string]string for compatibility
		for k, v := range stringFields {
			e.Details.AdditionalInfo[k] = v
		}
	}
	return e
}

// --- Compatibility chain methods (no-op enrichers used by examples) ---
func (e *AgentAuthError) WithSource(source string) *AgentAuthError {
	e.Source = source
	return e
}

func (e *AgentAuthError) WithRequestInfo(parts ...interface{}) *AgentAuthError {
	if e.Details == nil {
		e.Details = &ErrorDetails{Timestamp: e.Timestamp}
	}
	if e.Details.AdditionalInfo == nil {
		e.Details.AdditionalInfo = map[string]interface{}{}
	}
	e.Details.AdditionalInfo["request_info"] = parts
	return e
}

func (e *AgentAuthError) WithHTTPInfo(parts ...interface{}) *AgentAuthError {
	if e.Details == nil {
		e.Details = &ErrorDetails{Timestamp: e.Timestamp}
	}
	if e.Details.AdditionalInfo == nil {
		e.Details.AdditionalInfo = map[string]interface{}{}
	}
	e.Details.AdditionalInfo["http_info"] = parts
	// Best-effort: detect numeric status code in parts
	for _, p := range parts {
		if code, ok := p.(int); ok && code >= 100 && code < 600 {
			e.Details.HTTPStatusCode = code
		}
	}
	return e
}

func (e *AgentAuthError) AddInfo(k string, v interface{}) *AgentAuthError {
	if e.Details == nil {
		e.Details = &ErrorDetails{Timestamp: e.Timestamp}
	}
	if e.Details.AdditionalInfo == nil {
		e.Details.AdditionalInfo = map[string]interface{}{}
	}
	e.Details.AdditionalInfo[k] = v
	return e
}

// New creates a new AgentAuthError
func New(codeOrError interface{}, message string) *AgentAuthError {
	_, file, line, _ := runtime.Caller(1)

	// Handle case where first argument is a *AgentAuthError (for compatibility with examples)
	if gErr, ok := codeOrError.(*AgentAuthError); ok {
		newErr := &AgentAuthError{
			Code:      gErr.Code,
			Message:   message,
			Timestamp: time.Now(),
			File:      file,
			Line:      line,
		}
		newErr.Details = &ErrorDetails{Timestamp: newErr.Timestamp}
		return newErr
	}

	// Handle normal case where first argument is an ErrorCode
	if code, ok := codeOrError.(ErrorCode); ok {
		newErr := &AgentAuthError{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
			File:      file,
			Line:      line,
		}
		newErr.Details = &ErrorDetails{Timestamp: newErr.Timestamp}
		return newErr
	}

	// Fallback to internal error
	newErr := &AgentAuthError{
		Code:      ErrCodeInternal,
		Message:   message,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
	newErr.Details = &ErrorDetails{Timestamp: newErr.Timestamp}
	return newErr
}

// Newf creates a new AgentAuthError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *AgentAuthError {
	return New(code, fmt.Sprintf(format, args...))
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if gErr, ok := err.(*AgentAuthError); ok {
		return gErr.Code == ErrCodeRateLimit
	}
	return false
}

// GetRetryAfter extracts retry-after duration from error details
func GetRetryAfter(err error) time.Duration {
	if gErr, ok := err.(*AgentAuthError); ok {
		if gErr.Details != nil && gErr.Details.AdditionalInfo != nil {
			if retryAfter, ok := gErr.Details.AdditionalInfo["retry_after"]; ok {
				if duration, ok := retryAfter.(time.Duration); ok {
					return duration
				}
			}
		}
	}
	return 0
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *AgentAuthError {
	if gErr, ok := err.(*AgentAuthError); ok {
		// If it's already a AgentAuthError, preserve the original but add context
		newErr := &AgentAuthError{
			Code:      code,
			Message:   fmt.Sprintf("%s: %s", message, gErr.Message),
			Timestamp: time.Now(),
		}
		newErr.Details = &ErrorDetails{
			Timestamp: newErr.Timestamp,
			AdditionalInfo: map[string]interface{}{
				"wrapped_error": gErr,
			},
		}
		return newErr
	}

	_, file, line, _ := runtime.Caller(1)
	return &AgentAuthError{
		Code:      code,
		Message:   fmt.Sprintf("%s: %v", message, err),
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
}

// IsCode checks if the error has a specific code
func IsCode(err error, code ErrorCode) bool {
	if gErr, ok := err.(*AgentAuthError); ok {
		return gErr.Code == code
	}
	return false
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	if gErr, ok := err.(*AgentAuthError); ok {
		return gErr.Code
	}
	return ErrCodeInternal
}

// Additional predefined errors for common cases
var (
	ErrUnauthenticated = New(ErrCodeUnauthenticated, "authentication required")
	ErrUnauthorized    = New(ErrCodeUnauthorized, "insufficient permissions")
	ErrNotFound        = New(ErrCodeNotFound, "resource not found")
	ErrInternal        = New(ErrCodeInternal, "internal server error")
	ErrRateLimit       = New(ErrCodeRateLimit, "rate limit exceeded")
	ErrTimeout         = New(ErrCodeTimeout, "operation timed out")
	// Legacy/example exported errors for compatibility
	ErrInvalidToken      = New(ErrCodeInvalidToken, "invalid token")
	ErrInsufficientScope = New(ErrCodeInsufficientScope, "insufficient scope")
	ErrRateLimited       = ErrRateLimit
)

// ValidationError represents a validation error with field-specific information
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}

// Error implements the error interface
func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", v.Field, v.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// MultiError holds multiple errors
type MultiError struct {
	Errors []error `json:"errors"`
}

// Error implements the error interface
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors: %d total", len(m.Errors))
}

// Add adds an error to the multi-error
func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

// NewMultiError creates a new multi-error
func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]error, 0),
	}
}

// Common error constructors

// Middleware creates an error handling middleware function
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					var gErr *AgentAuthError
					if e, ok := err.(*AgentAuthError); ok {
						gErr = New(ErrCodeInternal, fmt.Sprintf("Internal server error: %v", e))
					} else {
						gErr = New(ErrCodeInternal, fmt.Sprintf("Internal server error: %v", err))
					}
					http.Error(w, gErr.Error(), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
