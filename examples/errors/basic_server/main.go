package main

import (
	"errors"
	"log"
	"net/http"
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

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate token validation error
// ...error construction removed (was unused)...

	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func resourceHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate insufficient scope error
// ...error construction removed (was unused)...

	// middleware.ErrorResponse(w, r, err) // Not available: middleware package missing
}

func rateLimitedHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate rate limit error
// ...baseErr removed (was unused)...
// ...error construction removed (was unused)...

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
