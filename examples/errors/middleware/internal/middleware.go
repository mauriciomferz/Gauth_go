package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	stderrs "errors"

	"github.com/Gimel-Foundation/gauth/pkg/errors"
	"github.com/google/uuid"
)

// ErrorHandler is middleware that handles errors from downstream handlers
type ErrorHandler struct {
	Next http.Handler
}

// ServeHTTP implements the http.Handler interface
func (e *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	crw := &captureResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	r.Header.Set("X-Request-ID", requestID)
	ctx := r.Context()
	r = r.WithContext(ctx)
	e.Next.ServeHTTP(crw, r)
}

type captureResponseWriter struct {
	http.ResponseWriter
	status int
}

func (c *captureResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	errBody := map[string]interface{}{
		"error":             "server_error",
		"error_description": "An unexpected error occurred",
		"request_id":        r.Header.Get("X-Request-ID"),
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	var authErr *errors.Error
	if stderrs.As(err, &authErr) {
		if authErr.Details != nil && authErr.Details.HTTPStatusCode > 0 {
			status = authErr.Details.HTTPStatusCode
		}
		errBody["error"] = string(authErr.Code)
		errBody["error_description"] = authErr.Message
		if authErr.Details != nil {
			for k, v := range authErr.Details.AdditionalInfo {
				if k != "error" && k != "error_description" && k != "request_id" && k != "timestamp" {
					errBody[k] = v
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errBody)
}
