package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	stderrors "errors"

	"github.com/Gimel-Foundation/gauth/pkg/errors"
	"github.com/google/uuid"
)

// ErrorHandler is middleware that handles errors from downstream handlers
type ErrorHandler struct {
	Next http.Handler
}

// ServeHTTP implements the http.Handler interface
func (e *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a custom response writer to capture status
	crw := &captureResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}

	// Store request context for error tracking
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Pass the request ID to downstream handlers
	r.Header.Set("X-Request-ID", requestID)

	// Create a context with request ID
	ctx := r.Context()
	r = r.WithContext(ctx)

	// Call the next handler
	e.Next.ServeHTTP(crw, r)
}

// captureResponseWriter captures the status code written
type captureResponseWriter struct {
	http.ResponseWriter
	status int
}

func (c *captureResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

// ErrorResponse creates an error response for the client
func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	// Default to internal server error
	status := http.StatusInternalServerError
	errBody := map[string]interface{}{
		"error":             "server_error",
		"error_description": "An unexpected error occurred",
		"request_id":        r.Header.Get("X-Request-ID"),
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	// Check if it's our structured error
	var authErr *errors.Error
	if stderrors.As(err, &authErr) {
		// Use our structured error details
		if authErr.Details != nil && authErr.Details.HTTPStatusCode > 0 {
			status = authErr.Details.HTTPStatusCode
		}

		// Build error response
		errBody["error"] = string(authErr.Code)
		errBody["error_description"] = authErr.Message

		// Add any additional info
		if authErr.Details != nil {
			for k, v := range authErr.Details.AdditionalInfo {
				// Don't override standard fields
				if k != "error" && k != "error_description" && k != "request_id" && k != "timestamp" {
					errBody[k] = v
				}
			}
		}
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	w.WriteHeader(status)

	// Write the error response
	json.NewEncoder(w).Encode(errBody)
}
