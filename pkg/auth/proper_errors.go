// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
)

// ProperError provides structured error handling with context and security considerations
// This replaces the generic error handling throughout the codebase
type ProperError struct {
	Type      ErrorType         `json:"type"`
	Message   string            `json:"message"`
	Code      string            `json:"code"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	RequestID string            `json:"request_id,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	
	// Internal fields (not exposed in JSON for security)
	InternalMessage string `json:"-"`
	StackTrace      string `json:"-"`
	Cause           error  `json:"-"`
}

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeAuthentication  ErrorType = "authentication"
	ErrorTypeAuthorization   ErrorType = "authorization"
	ErrorTypeValidation      ErrorType = "validation"
	ErrorTypeConfiguration   ErrorType = "configuration"
	ErrorTypeInternal        ErrorType = "internal"
	ErrorTypeExternal        ErrorType = "external"
	ErrorTypeRateLimit       ErrorType = "rate_limit"
	ErrorTypeCryptographic   ErrorType = "cryptographic"
	ErrorTypeCompliance      ErrorType = "compliance"
)

// Error implements the error interface
func (pe *ProperError) Error() string {
	return pe.Message
}

// Unwrap implements error unwrapping for Go 1.13+
func (pe *ProperError) Unwrap() error {
	return pe.Cause
}

// Is implements error checking for Go 1.13+
func (pe *ProperError) Is(target error) bool {
	var targetError *ProperError
	if errors.As(target, &targetError) {
		return pe.Type == targetError.Type && pe.Code == targetError.Code
	}
	return false
}

// NewProperError creates a new structured error
func NewProperError(errorType ErrorType, code, message string) *ProperError {
	// Get stack trace for debugging (only first 3 frames to avoid noise)
	stackTrace := getStackTrace(3)
	
	return &ProperError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Details:   make(map[string]string),
		StackTrace: stackTrace,
	}
}

// WithCause adds a cause to the error
func (pe *ProperError) WithCause(cause error) *ProperError {
	pe.Cause = cause
	pe.InternalMessage = fmt.Sprintf("%s: %v", pe.Message, cause)
	return pe
}

// WithDetail adds additional detail to the error
func (pe *ProperError) WithDetail(key, value string) *ProperError {
	if pe.Details == nil {
		pe.Details = make(map[string]string)
	}
	pe.Details[key] = value
	return pe
}

// WithRequestID adds request ID for tracing
func (pe *ProperError) WithRequestID(requestID string) *ProperError {
	pe.RequestID = requestID
	return pe
}

// WithUserID adds user ID (sanitized for logging)
func (pe *ProperError) WithUserID(userID string) *ProperError {
	// Only store first 8 characters of user ID for privacy
	if len(userID) > 8 {
		pe.UserID = userID[:8] + "..."
	} else {
		pe.UserID = userID
	}
	return pe
}

// IsSensitive determines if error contains sensitive information that shouldn't be logged
func (pe *ProperError) IsSensitive() bool {
	sensitiveTypes := []ErrorType{
		ErrorTypeAuthentication,
		ErrorTypeCryptographic,
		ErrorTypeCompliance,
	}
	
	for _, sensitiveType := range sensitiveTypes {
		if pe.Type == sensitiveType {
			return true
		}
	}
	return false
}

// PublicMessage returns a user-safe version of the error message
func (pe *ProperError) PublicMessage() string {
	switch pe.Type {
	case ErrorTypeAuthentication:
		return "Authentication failed"
	case ErrorTypeAuthorization:
		return "Access denied"
	case ErrorTypeCryptographic:
		return "Security operation failed"
	case ErrorTypeInternal:
		return "Internal server error"
	case ErrorTypeCompliance:
		return "Compliance validation failed"
	default:
		return pe.Message
	}
}

// getStackTrace captures stack trace for debugging
func getStackTrace(maxFrames int) string {
	var stackTrace string
	for i := 2; i < maxFrames+2; i++ { // Skip getStackTrace and NewProperError
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		stackTrace += fmt.Sprintf("%s:%d\n", file, line)
	}
	return stackTrace
}

// Common error constructors for frequently used errors

// ErrInvalidToken creates a token validation error
func ErrInvalidToken(reason string) *ProperError {
	return NewProperError(ErrorTypeAuthentication, "INVALID_TOKEN", "Invalid token").
		WithDetail("reason", reason)
}

// ErrTokenExpired creates a token expiration error
func ErrTokenExpired() *ProperError {
	return NewProperError(ErrorTypeAuthentication, "TOKEN_EXPIRED", "Token has expired")
}

// ErrInsufficientPermissions creates an authorization error
func ErrInsufficientPermissions(requiredScope string) *ProperError {
	return NewProperError(ErrorTypeAuthorization, "INSUFFICIENT_PERMISSIONS", "Insufficient permissions").
		WithDetail("required_scope", requiredScope)
}

// ErrRateLimitExceeded creates a rate limiting error
func ErrRateLimitExceeded(limit int, window time.Duration) *ProperError {
	return NewProperError(ErrorTypeRateLimit, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded").
		WithDetail("limit", fmt.Sprintf("%d", limit)).
		WithDetail("window", window.String())
}

// ErrCryptographicOperation creates a cryptographic error (sanitized for security)
func ErrCryptographicOperation(operation string) *ProperError {
	return NewProperError(ErrorTypeCryptographic, "CRYPTO_ERROR", "Cryptographic operation failed").
		WithDetail("operation", operation)
}

// ErrValidationFailed creates a validation error
func ErrValidationFailed(field, reason string) *ProperError {
	return NewProperError(ErrorTypeValidation, "VALIDATION_FAILED", "Validation failed").
		WithDetail("field", field).
		WithDetail("reason", reason)
}

// ErrorHandler provides centralized error handling with logging and metrics
type ErrorHandler struct {
	logger      Logger
	metrics     MetricsCollector
	environment string
}

// Logger interface for structured logging
type Logger interface {
	Error(ctx context.Context, message string, fields map[string]interface{})
	Warn(ctx context.Context, message string, fields map[string]interface{})
	Info(ctx context.Context, message string, fields map[string]interface{})
}

// MetricsCollector interface for error metrics
type MetricsCollector interface {
	IncrementErrorCounter(errorType string, errorCode string)
	RecordErrorLatency(operation string, duration time.Duration)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger Logger, metrics MetricsCollector, environment string) *ErrorHandler {
	return &ErrorHandler{
		logger:      logger,
		metrics:     metrics,
		environment: environment,
	}
}

// HandleError processes errors with appropriate logging and metrics
func (eh *ErrorHandler) HandleError(ctx context.Context, err error) *ProperError {
	var properErr *ProperError
	
	// Convert to ProperError if it isn't already
	if !errors.As(err, &properErr) {
		properErr = NewProperError(ErrorTypeInternal, "UNKNOWN_ERROR", err.Error()).
			WithCause(err)
	}
	
	// Record metrics
	if eh.metrics != nil {
		eh.metrics.IncrementErrorCounter(string(properErr.Type), properErr.Code)
	}
	
	// Log error with appropriate level
	eh.logError(ctx, properErr)
	
	return properErr
}

// logError logs the error with appropriate level and sanitization
func (eh *ErrorHandler) logError(ctx context.Context, err *ProperError) {
	if eh.logger == nil {
		return
	}
	
	// Prepare log fields
	fields := map[string]interface{}{
		"error_type":  err.Type,
		"error_code":  err.Code,
		"timestamp":   err.Timestamp,
		"request_id":  err.RequestID,
		"user_id":     err.UserID,
	}
	
	// Add details if not sensitive
	if !err.IsSensitive() {
		fields["details"] = err.Details
	}
	
	// Add stack trace in development
	if eh.environment == "development" {
		fields["stack_trace"] = err.StackTrace
		fields["internal_message"] = err.InternalMessage
	}
	
	// Choose log level based on error type
	switch err.Type {
	case ErrorTypeInternal, ErrorTypeCryptographic:
		eh.logger.Error(ctx, err.Message, fields)
	case ErrorTypeConfiguration, ErrorTypeExternal:
		eh.logger.Warn(ctx, err.Message, fields)
	default:
		eh.logger.Info(ctx, err.Message, fields)
	}
}

// RecoverFromPanic handles panics and converts them to proper errors
func (eh *ErrorHandler) RecoverFromPanic(ctx context.Context) *ProperError {
	if r := recover(); r != nil {
		var err *ProperError
		
		switch v := r.(type) {
		case error:
			err = NewProperError(ErrorTypeInternal, "PANIC_RECOVERED", "Panic recovered").
				WithCause(v)
		case string:
			err = NewProperError(ErrorTypeInternal, "PANIC_RECOVERED", v)
		default:
			err = NewProperError(ErrorTypeInternal, "PANIC_RECOVERED", "Unknown panic occurred")
		}
		
		// Log critical error
		eh.logError(ctx, err)
		
		return err
	}
	return nil
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid  bool           `json:"valid"`
	Errors []*ProperError `json:"errors,omitempty"`
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(err *ProperError) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, err)
}

// HasErrors checks if there are any validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// FirstError returns the first validation error if any
func (vr *ValidationResult) FirstError() *ProperError {
	if len(vr.Errors) > 0 {
		return vr.Errors[0]
	}
	return nil
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:  true,
		Errors: make([]*ProperError, 0),
	}
}

// ErrorMiddleware provides HTTP middleware for error handling
type ErrorMiddleware struct {
	handler *ErrorHandler
}

// NewErrorMiddleware creates new error middleware
func NewErrorMiddleware(handler *ErrorHandler) *ErrorMiddleware {
	return &ErrorMiddleware{handler: handler}
}

// Context helpers for error handling

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, "request_id", requestID)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}