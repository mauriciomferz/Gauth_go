package ratelimit

import (
	"net/http"
	"strconv"
	"time"
)

// HTTPRateLimitHandler provides HTTP middleware for rate limiting
type HTTPRateLimitHandler struct {
	limiter     *ClientRateLimiter
	getClientID func(r *http.Request) string
	onRejected  func(w http.ResponseWriter, r *http.Request)
}

// HTTPRateLimitConfig defines configuration for HTTP rate limit middleware
type HTTPRateLimitConfig struct {
	// Window is the time window for rate limiting
	Window time.Duration

	// MaxRequests is the maximum number of requests allowed in the window
	MaxRequests int

	// GetClientID is a function that extracts a client identifier from the request
	// If nil, remote IP address will be used
	GetClientID func(r *http.Request) string

	// OnRejected is called when a request is rejected due to rate limiting
	// If nil, a default 429 Too Many Requests response will be used
	OnRejected func(w http.ResponseWriter, r *http.Request)
}

// NewHTTPRateLimitHandler creates a new HTTP rate limit middleware
func NewHTTPRateLimitHandler(config HTTPRateLimitConfig) *HTTPRateLimitHandler {
	// Create client rate limiter
	limiter := NewClientRateLimiter(config.Window, config.MaxRequests)

	// Use default client ID function if none provided
	getClientID := config.GetClientID
	if getClientID == nil {
		getClientID = func(r *http.Request) string {
			return r.RemoteAddr
		}
	}

	// Use default rejection handler if none provided
	onRejected := config.OnRejected
	if onRejected == nil {
		onRejected = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			if _, err := w.Write([]byte("Rate limit exceeded. Please try again later.")); err != nil {
				// Log error but don't fail the handler
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}
	}

	return &HTTPRateLimitHandler{
		limiter:     limiter,
		getClientID: getClientID,
		onRejected:  onRejected,
	}
}

// Middleware returns an http.Handler middleware function
func (h *HTTPRateLimitHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client ID
		clientID := h.getClientID(r)

		// Check if request is allowed
		if h.limiter.IsAllowed(clientID) {
			// Request allowed, add rate limit headers
			state := h.limiter.GetClientState(clientID)
			if state != nil {
				remaining := state.MaxRequests - state.Count
				resetTime := state.WindowStart.Add(state.WindowSize).Unix()

				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(state.MaxRequests))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		} else {
			// Request rejected due to rate limiting
			h.onRejected(w, r)
		}
	})
}

// WrapHandlerFunc wraps an http.HandlerFunc with rate limiting
func (h *HTTPRateLimitHandler) WrapHandlerFunc(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.Middleware(handlerFunc).ServeHTTP(w, r)
	}
}
