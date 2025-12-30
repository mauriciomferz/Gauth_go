package main

import (
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"net/http"
	"time"

	agentauthErrors "github.com/mauriciomferz/AgentAuth/pkg/errors"
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
	err := agentauthErrors.New(agentauthErrors.ErrInvalidToken, "The token provided is malformed or invalid")
	err = err.WithSource(agentauthErrors.SourceToken)
	err = err.WithRequestInfo(r.Header.Get("X-Request-ID"), "client-456", "user-789")
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusUnauthorized, r.RemoteAddr)
	err = err.AddInfo("token_hint", "Check token format and signature")

	log.Println(err)
	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func resourceHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate insufficient scope error
	err := agentauthErrors.New(agentauthErrors.ErrInsufficientScope, "The token does not have the required scope")
	err = err.WithSource(agentauthErrors.SourceAuthorization)
	err = err.WithRequestInfo(r.Header.Get("X-Request-ID"), "client-456", "user-789")
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusForbidden, r.RemoteAddr)

	// Use WithFields to add multiple fields at once
	err = err.WithFields(map[string]string{
		"required_scope": "admin",
		"provided_scope": "user",
		"resource_id":    "resource-123",
		"action":         "write",
	})

	log.Println(err)
	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func rateLimitedHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate rate limit error
	baseErr := fmt.Errorf("rate limit of 100 requests per minute exceeded")
	err := agentauthErrors.New(agentauthErrors.ErrRateLimited, "API rate limit exceeded")
	err = err.WithSource(agentauthErrors.SourceRateLimiting)
	err = err.WithCause(baseErr)
	err = err.WithRequestInfo(r.Header.Get("X-Request-ID"), "client-123", "")
	err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusTooManyRequests, r.RemoteAddr)
	err = err.AddInfo("retry_after", "60")
	err = err.AddInfo("limit", "100")
	err = err.AddInfo("remaining", "0")
	err = err.AddInfo("reset", "2023-07-01T15:30:45Z")

	// Demonstrate checking for rate limit errors
	if agentauthErrors.IsRateLimitError(err) {
		retryAfter := agentauthErrors.GetRetryAfter(err)
		log.Printf("Rate limit exceeded. Retry after %v", retryAfter)
		w.Header().Set("Retry-After", fmt.Sprintf("%v", retryAfter))
	}

	log.Println(err)
	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func serverErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Example of wrapping a standard error
	baseErr := stderrors.New("database connection failed: timeout")

	// Convert to structured error for more context
	err := agentauthErrors.New(agentauthErrors.ErrServerError, "Database operation failed")
	err = err.WithSource(agentauthErrors.SourceStorage)
	err = err.WithCause(baseErr)
	log.Println(err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusInternalServerError, r.RemoteAddr))
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
	err := agentauthErrors.New(agentauthErrors.ErrServerError, "Error occurred while processing context")
	// err = err.WithSource(agentauthErrors.SourceProtocol) // SourceProtocol not defined

	// Extract context values
	// err = err.WithContext(ctx) // Method not implemented

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
	err := agentauthErrors.New(agentauthErrors.ErrServerError, "Error with stack trace")
	// err = err.WithSource(agentauthErrors.SourceValidation) // SourceValidation not defined
	// err = err.WithStack() // Method not implemented
	err = err.WithHTTPInfo("/api/stack-trace", "GET", http.StatusInternalServerError, "")
	return err
}
