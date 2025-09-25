package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	gauthErrors "github.com/Gimel-Foundation/gauth/pkg/errors"
)

func main() {
	// Create router
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/api/token", tokenHandler)
	mux.HandleFunc("/api/resource", resourceHandler)
	mux.HandleFunc("/api/rate-limited", rateLimitedHandler)
	mux.HandleFunc("/api/server-error", serverErrorHandler)

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Try these endpoints:")
	log.Println("  - GET /api/token - Invalid token error")
	log.Println("  - GET /api/resource - Insufficient scope error")
	log.Println("  - GET /api/rate-limited - Rate limit error")
	log.Println("  - GET /api/server-error - Internal server error")

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
	err = err.AddInfo("required_scope", "admin")
	err = err.AddInfo("provided_scope", "user")

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

	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func serverErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Example of wrapping a standard error
	_ = errors.New("database connection failed: timeout")

	// You could use it directly
	// middleware.ErrorResponse(w, r, baseErr) // Not available: middleware package missing

	// Or convert to structured error for more context
	// err := gauthErrors.New(gauthErrors.ErrServerError, "Database operation failed")
	// err = err.WithSource(gauthErrors.SourceStorage)
	// err = err.WithCause(baseErr)
	// err = err.WithHTTPInfo(r.URL.Path, r.Method, http.StatusInternalServerError, r.RemoteAddr)
	// middleware.ErrorResponse(w, r, err)
}
