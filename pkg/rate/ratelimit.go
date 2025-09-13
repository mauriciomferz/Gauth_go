package rate

// Package rate provides rate limiting functionality for the GAuth protocol.

import (
	"context"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
)

// RateLimitConfig holds configuration for rate limiters
type RateLimitConfig struct {
	// RequestLimit is the maximum number of requests allowed in the reset interval
	RequestLimit int

	// ResetInterval is the duration after which the request count is reset
	ResetInterval time.Duration

	// ClientSpecific indicates whether to use per-client rate limiting
	ClientSpecific bool

	// Adaptive indicates whether to use adaptive rate limiting
	Adaptive bool

	// MinLimit is the minimum limit for adaptive rate limiting
	MinLimit int

	// MaxLimit is the maximum limit for adaptive rate limiting
	MaxLimit int
}

// RateLimitStats provides statistics about rate limiting
type RateLimitStats struct {
	// CurrentLimit is the current request limit
	CurrentLimit int

	// RemainingRequests is the number of remaining requests
	RemainingRequests int

	// ClientCount is the number of tracked clients (for client-specific limiting)
	ClientCount int

	// TimeUntilReset is the duration until the current window resets
	TimeUntilReset time.Duration
}

// RateLimiter interface defines the common operations for all rate limiter types
type RateLimiter interface {
	// Allow checks if a request is allowed
	Allow(ctx context.Context) bool

	// GetStats returns statistics about the rate limiter
	GetStats() map[string]interface{}
}

// BasicRateLimiter provides a simple rate limiter implementation
type BasicRateLimiter struct {
	impl *ratelimit.RateLimiter
}

// NewRateLimiter creates a new rate limiter based on the provided configuration
func NewRateLimiter(config RateLimitConfig) RateLimiter {
	if config.Adaptive {
		return newAdaptiveRateLimiter(config)
	}

	if config.ClientSpecific {
		return newClientRateLimiter(config)
	}

	return newBasicRateLimiter(config)
}

// newBasicRateLimiter creates a basic rate limiter
func newBasicRateLimiter(config RateLimitConfig) *BasicRateLimiter {
	impl := ratelimit.NewRateLimiter(ratelimit.RateLimiterConfig{
		RequestLimit:  config.RequestLimit,
		ResetInterval: config.ResetInterval,
	})

	return &BasicRateLimiter{
		impl: impl,
	}
}

// Allow checks if a request is allowed
func (b *BasicRateLimiter) Allow(ctx context.Context) bool {
	return b.impl.Allow(ctx)
}

// GetStats returns statistics about the rate limiter
func (b *BasicRateLimiter) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["currentLimit"] = b.impl.RemainingRequests() + b.impl.RemainingRequests()
	stats["remainingRequests"] = b.impl.RemainingRequests()
	stats["timeUntilReset"] = b.impl.TimeUntilReset().String()
	return stats
}

// ClientRateLimiter provides client-specific rate limiting
type ClientRateLimiter struct {
	impl *ratelimit.ClientRateLimiter
}

// newClientRateLimiter creates a client-specific rate limiter
func newClientRateLimiter(config RateLimitConfig) *ClientRateLimiter {
	impl := ratelimit.NewClientRateLimiter(
		config.ResetInterval,
		config.RequestLimit,
	)

	return &ClientRateLimiter{
		impl: impl,
	}
}

// Allow checks if a request is allowed (always using a default client ID)
func (c *ClientRateLimiter) Allow(ctx context.Context) bool {
	// Extract client ID from context, or use default
	clientID := getClientIDFromContext(ctx)
	return c.impl.IsAllowed(clientID)
}

// AllowForClient checks if a request from a specific client is allowed
func (c *ClientRateLimiter) AllowForClient(clientID string) bool {
	return c.impl.IsAllowed(clientID)
}

// GetStats returns statistics about the rate limiter
func (c *ClientRateLimiter) GetStats() map[string]interface{} {
	return c.impl.GetStats()
}

// AdaptiveRateLimiter provides a rate limiter that adapts to system load
type AdaptiveRateLimiter struct {
	impl *ratelimit.AdaptiveRateLimiter
}

// newAdaptiveRateLimiter creates an adaptive rate limiter
func newAdaptiveRateLimiter(config RateLimitConfig) *AdaptiveRateLimiter {
	impl := ratelimit.NewAdaptiveRateLimiter(ratelimit.AdaptiveConfig{
		InitialLimit: config.RequestLimit,
		MinLimit:     config.MinLimit,
		MaxLimit:     config.MaxLimit,
		Window:       config.ResetInterval,
	})

	return &AdaptiveRateLimiter{
		impl: impl,
	}
}

// Allow checks if a request is allowed
func (a *AdaptiveRateLimiter) Allow(ctx context.Context) bool {
	return a.impl.Allow()
}

// GetStats returns statistics about the rate limiter
func (a *AdaptiveRateLimiter) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["currentLimit"] = a.impl.GetCurrentLimit()
	stats["currentUsage"] = a.impl.GetUsage()
	return stats
}

// HTTPRateLimitMiddleware returns HTTP middleware for rate limiting
func HTTPRateLimitMiddleware(config RateLimitConfig) func(http.Handler) http.Handler {
	middleware := ratelimit.NewHTTPRateLimitHandler(ratelimit.HTTPRateLimitConfig{
		Window:      config.ResetInterval,
		MaxRequests: config.RequestLimit,
	})

	return middleware.Middleware
}

// getClientIDFromContext extracts client ID from context, or returns a default
func getClientIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return "default"
	}

	// Use a type key for client ID
	type clientIDKey struct{}

	// Extract client ID from context
	if clientID, ok := ctx.Value(clientIDKey{}).(string); ok {
		return clientID
	}

	return "default"
}

// WithClientID adds client ID to context
func WithClientID(ctx context.Context, clientID string) context.Context {
	type clientIDKey struct{}
	return context.WithValue(ctx, clientIDKey{}, clientID)
}
