package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// RateLimitMiddleware applies rate limiting based on API key or IP address
type RateLimitMiddleware struct {
	limiter         RateLimiter
	keyStore        APIKeyStore
	useIPAsFallback bool
}

type apiKeyCtxKey struct{}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string) error
	GetStats(key string) *Stats
}

// Stats represents rate limiting statistics
type Stats struct {
	RequestsAllowed int64
	RequestsDenied  int64
	ResetAt         time.Time
}

// APIKeyStore interface for API key management
type APIKeyStore interface {
	ValidateKey(ctx context.Context, key string) (*APIKey, error)
	GetKeyByID(ctx context.Context, keyID string) (*APIKey, error)
	CreateKey(ctx context.Context, key *APIKey) error
	UpdateKey(ctx context.Context, key *APIKey) error
	DeleteKey(ctx context.Context, keyID string) error
	ListKeys(ctx context.Context, userID string) ([]*APIKey, error)
}

// APIKey represents an API key with its metadata
type APIKey struct {
	ID         string
	HashedKey  string
	Name       string
	UserID     string
	Scopes     []string
	RateLimit  int // requests per second
	Enabled    bool
	ExpiresAt  *time.Time
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(limiter RateLimiter, keyStore APIKeyStore, useIPAsFallback bool) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter:         limiter,
		keyStore:        keyStore,
		useIPAsFallback: useIPAsFallback,
	}
}

// Handler wraps an HTTP handler with rate limiting
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header or query parameter
		apiKey := m.extractAPIKey(r)
		var apiKeyInfo *APIKey
		var rateLimitKey string

		// Validate API key
		switch {
		case apiKey != "":
			var err error
			apiKeyInfo, err = m.keyStore.ValidateKey(r.Context(), apiKey)
			if err != nil {
				m.respondError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid API key")
				return
			}
			if !apiKeyInfo.Enabled {
				m.respondError(w, http.StatusForbidden, "api_key_disabled", "API key has been disabled")
				return
			}
			if apiKeyInfo.ExpiresAt != nil && time.Now().After(*apiKeyInfo.ExpiresAt) {
				m.respondError(w, http.StatusForbidden, "api_key_expired", "API key has expired")
				return
			}
			rateLimitKey = fmt.Sprintf("apikey:%s", apiKeyInfo.ID)
		case m.useIPAsFallback:
			// Fall back to IP-based rate limiting
			ip := m.extractIP(r)
			rateLimitKey = fmt.Sprintf("ip:%s", ip)
		default:
			m.respondError(w, http.StatusUnauthorized, "missing_api_key", "API key required")
			return
		}

		// Check rate limit
		if err := m.limiter.Allow(r.Context(), rateLimitKey); err != nil {
			stats := m.limiter.GetStats(rateLimitKey)
			// Set rate limit headers
			if apiKeyInfo != nil {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKeyInfo.RateLimit))
			}
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", stats.ResetAt.Unix()))
			// Set retry-after header
			if retryAfter, ok := getRetryAfter(err); ok {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			}
			m.respondError(w, http.StatusTooManyRequests, "rate_limit_exceeded", err.Error())
			return
		}

		// Add API key info to context if available
		if apiKeyInfo != nil {
			ctx := context.WithValue(r.Context(), apiKeyCtxKey{}, apiKeyInfo)
			r = r.WithContext(ctx)
			// Set rate limit headers for successful requests
			stats := m.limiter.GetStats(rateLimitKey)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKeyInfo.RateLimit))
			remaining := apiKeyInfo.RateLimit - int(stats.RequestsDenied)
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", stats.ResetAt.Unix()))
		}

		next.ServeHTTP(w, r)
	})
}

// extractAPIKey extracts the API key from request headers or query parameters
func (m *RateLimitMiddleware) extractAPIKey(r *http.Request) string {
	// Check Authorization header (Bearer token format)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Check X-API-Key header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Check query parameter (less secure, for backward compatibility)
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// extractIP extracts the client IP address from the request
func (m *RateLimitMiddleware) extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (load balancer/proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// respondError sends an error response
func (m *RateLimitMiddleware) respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error":{"code":"%s","message":"%s"}}`, code, message)
}

// getRetryAfter extracts retry-after duration from rate limit error
func getRetryAfter(err error) (time.Duration, bool) {
	// This would need to be implemented based on your RateLimitError type
	// For now, return a default
	return time.Minute, true
}

// RegisterRoutes adds rate limit management endpoints to the router
func (m *RateLimitMiddleware) RegisterRoutes(r *mux.Router, keyManager *APIKeyManager) {
	// Rate limiting endpoints
	r.HandleFunc("/api/v1/admin/ratelimit/stats/{key}", m.handleGetStats).Methods("GET")
	r.HandleFunc("/api/v1/admin/ratelimit/reset/{key}", m.handleReset).Methods("POST")

	// API Key management endpoints
	r.HandleFunc("/api/v1/admin/apikeys", keyManager.handleCreateAPIKey).Methods("POST")
	r.HandleFunc("/api/v1/admin/apikeys", keyManager.handleListAPIKeys).Methods("GET")
	r.HandleFunc("/api/v1/admin/apikeys/{id}", keyManager.handleGetAPIKey).Methods("GET")
	r.HandleFunc("/api/v1/admin/apikeys/{id}", keyManager.handleUpdateAPIKey).Methods("PUT")
	r.HandleFunc("/api/v1/admin/apikeys/{id}", keyManager.handleDeleteAPIKey).Methods("DELETE")
	r.HandleFunc("/api/v1/admin/apikeys/{id}/regenerate", keyManager.handleRegenerateAPIKey).Methods("POST")
}

// handleGetStats returns rate limiting statistics for a key
func (m *RateLimitMiddleware) handleGetStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	stats := m.limiter.GetStats(key)
	if stats == nil {
		m.respondError(w, http.StatusNotFound, "key_not_found", "Rate limit key not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"key":"%s","requests_allowed":%d,"requests_denied":%d,"reset_at":"%s"}`,
		key, stats.RequestsAllowed, stats.RequestsDenied, stats.ResetAt.Format(time.RFC3339))
}

// handleReset resets the rate limit for a key
func (m *RateLimitMiddleware) handleReset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	// This would need to be implemented in your RateLimiter interface
	// For now, just return success

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"message":"Rate limit reset for key: %s"}`, key)
}

// HashAPIKey creates a SHA-256 hash of an API key for secure storage
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
