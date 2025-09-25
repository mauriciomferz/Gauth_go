package main

import (
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"net/http"
	"time"

	gauthErrors "github.com/Gimel-Foundation/gauth/pkg/errors"
)

// Define custom context keys
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

func main() {
	// Create router
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/api/token", tokenHandler)
	mux.HandleFunc("/api/resource", resourceHandler)
	mux.HandleFunc("/api/rate-limited", rateLimitedHandler)
	mux.HandleFunc("/api/server-error", serverErrorHandler)
	mux.HandleFunc("/api/context-error", contextErrorHandler)
	mux.HandleFunc("/api/stack-trace", stackTraceHandler)

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Try these endpoints:")
	log.Println("  - GET /api/token - Invalid token error")
	log.Println("  - GET /api/resource - Insufficient scope error")
	log.Println("  - GET /api/rate-limited - Rate limit error")
	log.Println("  - GET /api/server-error - Internal server error")
	log.Println("  - GET /api/context-error - Error with context information")
	log.Println("  - GET /api/stack-trace - Error with stack trace")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate token validation error
	err := gauthErrors.New(gauthErrors.ErrInvalidToken, "The token provided is malformed or invalid")
	err = err.WithSource(gauthErrors.SourceToken)
	err = err.WithRequestInfo(r.Header.Get("X-Request-ID"), "client-456", "user-789")
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusUnauthorized, r.RemoteAddr)
	err = err.AddInfo("token_hint", "Check token format and signature")

	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func resourceHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate insufficient scope error
	err := gauthErrors.New(gauthErrors.ErrInsufficientScope, "The token does not have the required scope")
	err = err.WithSource(gauthErrors.SourceAuthorization)
	err = err.WithRequestInfo(r.Header.Get("X-Request-ID"), "client-456", "user-789")
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusForbidden, r.RemoteAddr)

	// Use WithFields to add multiple fields at once
	err = err.WithFields(map[string]string{
		"required_scope": "admin",
		"provided_scope": "user",
		"resource_id":    "resource-123",
		"action":         "write",
	})

	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func rateLimitedHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate rate limit error
	baseErr := fmt.Errorf("rate limit of 100 requests per minute exceeded")
	err := gauthErrors.New(gauthErrors.ErrRateLimited, "API rate limit exceeded")
	err = err.WithSource(gauthErrors.SourceRateLimiting)
	err = err.WithCause(baseErr)
	err = err.WithRequestInfo(r.Header.Get("X-Request-ID"), "client-123", "")
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusTooManyRequests, r.RemoteAddr)
	err = err.AddInfo("retry_after", "60")
	err = err.AddInfo("limit", "100")
	err = err.AddInfo("remaining", "0")
	err = err.AddInfo("reset", "2023-07-01T15:30:45Z")

	// Demonstrate checking for rate limit errors
	if gauthErrors.IsRateLimitError(err) {
		retryAfter, ok := gauthErrors.GetRetryAfter(err)
		if ok {
			log.Printf("Rate limit exceeded. Retry after %d seconds", retryAfter)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		}
	}

	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func serverErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Example of wrapping a standard error
	baseErr := stderrors.New("database connection failed: timeout")

	// Convert to structured error for more context
	err := gauthErrors.New(gauthErrors.ErrServerError, "Database operation failed")
	err = err.WithSource(gauthErrors.SourceStorage)
	err = err.WithCause(baseErr)
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusInternalServerError, r.RemoteAddr)
	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func contextErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Create a context with values
	ctx := context.WithValue(r.Context(), requestIDKey, "ctx-req-123")
	ctx = context.WithValue(ctx, userIDKey, "ctx-user-456")

	// Simulate an error with context information
	_ = simulateErrorWithContext(ctx)
	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func simulateErrorWithContext(ctx context.Context) error {
	// Create error and extract info from context
	err := gauthErrors.New(gauthErrors.ErrServerError, "Error occurred while processing context")
	err = err.WithSource(gauthErrors.SourceProtocol)

	// Extract context values
	err = err.WithContext(ctx)

	// Add HTTP info (would normally come from request)
	err = err.WithHTTPInfo("/api/context-error", "GET", http.StatusInternalServerError, "")

	return err
}

func stackTraceHandler(w http.ResponseWriter, r *http.Request) {
	// Generate an error with stack trace
	_ = generateErrorWithStack()
	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func generateErrorWithStack() error {
	// Simulate a deeper stack
	return deeperFunction()
}

func deeperFunction() error {
	// Another level deeper
	return evenDeeperFunction()
}

func evenDeeperFunction() error {
	// Create error with stack trace
	err := gauthErrors.New(gauthErrors.ErrServerError, "Error with stack trace")
	err = err.WithSource(gauthErrors.SourceValidation)
	err = err.WithStack() // Capture stack trace
	err = err.WithHTTPInfo("/api/stack-trace", "GET", http.StatusInternalServerError, "")
	return err
}
