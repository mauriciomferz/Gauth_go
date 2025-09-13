package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	stderrors "errors"
)

// WithStack adds stack trace information to the error
func (e *Error) WithStack() *Error {
	if e.Details == nil {
		e.Details = &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]string),
		}
	}

	// Capture stack trace (skip this function and caller)
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	// Format stack trace
	var stackBuilder strings.Builder
	frameCount := 0

	for {
		frame, more := frames.Next()
		if !more || frameCount >= 10 { // Limit to 10 frames
			break
		}

		// Skip runtime functions
		if strings.Contains(frame.Function, "runtime.") {
			continue
		}

		// Add frame to stack trace
		fmt.Fprintf(&stackBuilder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		frameCount++
	}

	// Add stack trace to error details
	e.Details.AdditionalInfo["stack_trace"] = stackBuilder.String()
	return e
}

// WithField adds a custom field to the error
func (e *Error) WithField(key, value string) *Error {
	return e.AddInfo(key, value)
}

// WithFields adds multiple custom fields to the error
func (e *Error) WithFields(fields map[string]string) *Error {
	if e.Details == nil {
		e.Details = &ErrorDetails{
			Timestamp:      time.Now(),
			AdditionalInfo: make(map[string]string),
		}
	}

	for k, v := range fields {
		e.Details.AdditionalInfo[k] = v
	}

	return e
}

// WithContext extracts relevant information from a context
func (e *Error) WithContext(ctx context.Context) *Error {
	// Example of extracting information from context
	// In a real implementation, you'd extract your application-specific values

	// Example: Extract request ID from context
	if requestID, ok := ctx.Value("request_id").(string); ok && requestID != "" {
		if e.Details == nil {
			e.Details = &ErrorDetails{
				Timestamp:      time.Now(),
				AdditionalInfo: make(map[string]string),
			}
		}
		e.Details.RequestID = requestID
	}

	// Example: Extract user ID from context
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		if e.Details == nil {
			e.Details = &ErrorDetails{
				Timestamp:      time.Now(),
				AdditionalInfo: make(map[string]string),
			}
		}
		e.Details.UserID = userID
	}

	return e
}

// IsAuthError checks if the error is an authentication error
func IsAuthError(err error) bool {
	var authErr *Error
	if stderrors.As(err, &authErr) {
		return authErr.Code == ErrInvalidToken ||
			authErr.Code == ErrTokenExpired ||
			authErr.Code == ErrInvalidClient
	}
	return false
}

// IsRateLimitError checks if the error is a rate limit error
func IsRateLimitError(err error) bool {
	var authErr *Error
	if stderrors.As(err, &authErr) {
		return authErr.Code == ErrRateLimited
	}
	return false
}

// GetRetryAfter extracts the retry-after value if present
func GetRetryAfter(err error) (int, bool) {
	var authErr *Error
	if stderrors.As(err, &authErr) && authErr.Details != nil {
		if val, ok := authErr.Details.AdditionalInfo["retry_after"]; ok {
			if retryAfter, err := strconv.Atoi(val); err == nil {
				return retryAfter, true
			}
		}
	}
	return 0, false
}

// NewHTTPError creates an error from an HTTP response
func NewHTTPError(resp *http.Response, body []byte) *Error {
	var code ErrorCode
	var message string

	// Map HTTP status code to error code
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		code = ErrInvalidToken
		message = "Authentication failed"
	case http.StatusForbidden:
		code = ErrInsufficientScope
		message = "Insufficient permissions"
	case http.StatusTooManyRequests:
		code = ErrRateLimited
		message = "Rate limit exceeded"
	case http.StatusBadRequest:
		code = ErrInvalidRequest
		message = "Invalid request"
	default:
		code = ErrServerError
		message = "Server error"
	}

	// Try to parse response body as JSON
	var jsonResp map[string]interface{}
	if err := json.Unmarshal(body, &jsonResp); err == nil {
		// Override message if available in response
		if errMsg, ok := jsonResp["error_description"].(string); ok && errMsg != "" {
			message = errMsg
		} else if errMsg, ok := jsonResp["message"].(string); ok && errMsg != "" {
			message = errMsg
		}
	}

	// Create error with HTTP details
	err := New(code, message)
	err = err.WithHTTPInfo(
		resp.Request.URL.Path,
		resp.Request.Method,
		resp.StatusCode,
		"",
	)

	// Add response headers as additional info
	for k, v := range resp.Header {
		if len(v) > 0 {
			err = err.AddInfo(strings.ToLower(k), strings.Join(v, ", "))
		}
	}

	return err
}
