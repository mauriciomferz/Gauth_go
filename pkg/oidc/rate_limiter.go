package oidc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter defines the interface for rate limiting implementations.
type RateLimiter interface {
	// Allow checks if a request should be allowed based on the key.
	// Returns true if allowed, false if rate limit exceeded.
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN checks if N requests should be allowed.
	AllowN(ctx context.Context, key string, n int) (bool, error)

	// Reset resets the rate limit for a specific key.
	Reset(ctx context.Context, key string) error

	// GetLimit returns the current limit configuration.
	GetLimit() (requests int, window time.Duration)

	// Close closes any resources used by the rate limiter.
	Close() error
}

// TokenBucketLimiter implements rate limiting using the token bucket algorithm.
// This is an in-memory implementation suitable for single-instance deployments.
type TokenBucketLimiter struct {
	capacity int           // Maximum number of tokens in bucket
	refill   int           // Tokens to add per refill period
	interval time.Duration // Refill interval
	buckets  map[string]*bucket
	mu       sync.RWMutex
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter.
// capacity: maximum requests allowed in burst
// refill: number of tokens to add per interval
// interval: how often to add tokens
func NewTokenBucketLimiter(capacity, refill int, interval time.Duration) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		capacity: capacity,
		refill:   refill,
		interval: interval,
		buckets:  make(map[string]*bucket),
	}

	// Start cleanup goroutine to remove old buckets
	go limiter.cleanupLoop()

	return limiter
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

func (l *TokenBucketLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	l.mu.RLock()
	b, exists := l.buckets[key]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		// Double-check after acquiring write lock
		b, exists = l.buckets[key]
		if !exists {
			b = &bucket{
				tokens:     float64(l.capacity),
				lastRefill: time.Now(),
			}
			l.buckets[key] = b
		}
		l.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := float64(l.refill) * (float64(elapsed) / float64(l.interval))
	b.tokens = min(float64(l.capacity), b.tokens+tokensToAdd)
	b.lastRefill = now

	// Check if we have enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true, nil
	}

	return false, nil
}

func (l *TokenBucketLimiter) Reset(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
	return nil
}

func (l *TokenBucketLimiter) GetLimit() (int, time.Duration) {
	return l.refill, l.interval
}

func (l *TokenBucketLimiter) Close() error {
	// No resources to clean up for in-memory implementation
	return nil
}

func (l *TokenBucketLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, b := range l.buckets {
			b.mu.Lock()
			// Remove buckets that haven't been used in the last hour
			if now.Sub(b.lastRefill) > time.Hour {
				delete(l.buckets, key)
			}
			b.mu.Unlock()
		}
		l.mu.Unlock()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// SlidingWindowLimiter implements rate limiting using a sliding window counter.
type SlidingWindowLimiter struct {
	limit    int           // Maximum requests per window
	window   time.Duration // Time window
	counters map[string]*windowCounter
	mu       sync.RWMutex
}

type windowCounter struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter.
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	limiter := &SlidingWindowLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string]*windowCounter),
	}

	go limiter.cleanupLoop()

	return limiter
}

func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

func (l *SlidingWindowLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	l.mu.RLock()
	counter, exists := l.counters[key]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		counter, exists = l.counters[key]
		if !exists {
			counter = &windowCounter{
				requests: make([]time.Time, 0, l.limit),
			}
			l.counters[key] = counter
		}
		l.mu.Unlock()
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.window)

	// Remove requests outside the current window
	validRequests := make([]time.Time, 0, len(counter.requests))
	for _, t := range counter.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	counter.requests = validRequests

	// Check if we're within the limit
	if len(counter.requests)+n <= l.limit {
		for i := 0; i < n; i++ {
			counter.requests = append(counter.requests, now)
		}
		return true, nil
	}

	return false, nil
}

func (l *SlidingWindowLimiter) Reset(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.counters, key)
	return nil
}

func (l *SlidingWindowLimiter) GetLimit() (int, time.Duration) {
	return l.limit, l.window
}

func (l *SlidingWindowLimiter) Close() error {
	return nil
}

func (l *SlidingWindowLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, counter := range l.counters {
			counter.mu.Lock()
			// Remove counters with no recent requests
			if len(counter.requests) == 0 || now.Sub(counter.requests[len(counter.requests)-1]) > l.window*2 {
				delete(l.counters, key)
			}
			counter.mu.Unlock()
		}
		l.mu.Unlock()
	}
}

// RateLimitConfig defines rate limiting configuration for different endpoint types.
type RateLimitConfig struct {
	// Enabled enables rate limiting globally
	Enabled bool

	// Per-endpoint limits (requests per window)
	TokenEndpoint         EndpointLimit
	RefreshEndpoint       EndpointLimit
	IntrospectionEndpoint EndpointLimit
	RevocationEndpoint    EndpointLimit
	DeviceAuthEndpoint    EndpointLimit
	PAREndpoint           EndpointLimit
	JWKSEndpoint          EndpointLimit
	UserinfoEndpoint      EndpointLimit

	// Global IP-based limit (applies to all endpoints)
	GlobalIPLimit *EndpointLimit

	// Client-based limit (applies per client_id)
	GlobalClientLimit *EndpointLimit
}

// EndpointLimit defines rate limit for a specific endpoint.
type EndpointLimit struct {
	Enabled       bool
	RequestsPerIP int           // Requests per IP per window
	Window        time.Duration // Time window for rate limit
	BurstSize     int           // Burst capacity (for token bucket)
}

// DefaultRateLimitConfig returns a production-ready rate limit configuration.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: true,
		TokenEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 10,
			Window:        time.Minute,
			BurstSize:     20,
		},
		RefreshEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 20,
			Window:        time.Minute,
			BurstSize:     30,
		},
		IntrospectionEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 30,
			Window:        time.Minute,
			BurstSize:     50,
		},
		RevocationEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 10,
			Window:        time.Minute,
			BurstSize:     20,
		},
		DeviceAuthEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 5,
			Window:        time.Minute,
			BurstSize:     10,
		},
		PAREndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 10,
			Window:        time.Minute,
			BurstSize:     20,
		},
		JWKSEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 100,
			Window:        time.Minute,
			BurstSize:     200,
		},
		UserinfoEndpoint: EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 30,
			Window:        time.Minute,
			BurstSize:     50,
		},
		GlobalIPLimit: &EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 100,
			Window:        time.Minute,
			BurstSize:     150,
		},
		GlobalClientLimit: &EndpointLimit{
			Enabled:       true,
			RequestsPerIP: 200,
			Window:        time.Minute,
			BurstSize:     300,
		},
	}
}

// RateLimitService provides rate limiting for OIDC endpoints.
type RateLimitService struct {
	config   *RateLimitConfig
	limiters map[string]RateLimiter
	mu       sync.RWMutex
}

// NewRateLimitService creates a new rate limiting service.
func NewRateLimitService(config *RateLimitConfig) *RateLimitService {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	service := &RateLimitService{
		config:   config,
		limiters: make(map[string]RateLimiter),
	}

	// Create limiters for each endpoint
	if config.TokenEndpoint.Enabled {
		service.limiters["token"] = NewTokenBucketLimiter(
			config.TokenEndpoint.BurstSize,
			config.TokenEndpoint.RequestsPerIP,
			config.TokenEndpoint.Window,
		)
	}
	if config.RefreshEndpoint.Enabled {
		service.limiters["refresh"] = NewTokenBucketLimiter(
			config.RefreshEndpoint.BurstSize,
			config.RefreshEndpoint.RequestsPerIP,
			config.RefreshEndpoint.Window,
		)
	}
	if config.IntrospectionEndpoint.Enabled {
		service.limiters["introspection"] = NewTokenBucketLimiter(
			config.IntrospectionEndpoint.BurstSize,
			config.IntrospectionEndpoint.RequestsPerIP,
			config.IntrospectionEndpoint.Window,
		)
	}
	if config.RevocationEndpoint.Enabled {
		service.limiters["revocation"] = NewTokenBucketLimiter(
			config.RevocationEndpoint.BurstSize,
			config.RevocationEndpoint.RequestsPerIP,
			config.RevocationEndpoint.Window,
		)
	}
	if config.DeviceAuthEndpoint.Enabled {
		service.limiters["device_auth"] = NewTokenBucketLimiter(
			config.DeviceAuthEndpoint.BurstSize,
			config.DeviceAuthEndpoint.RequestsPerIP,
			config.DeviceAuthEndpoint.Window,
		)
	}
	if config.PAREndpoint.Enabled {
		service.limiters["par"] = NewTokenBucketLimiter(
			config.PAREndpoint.BurstSize,
			config.PAREndpoint.RequestsPerIP,
			config.PAREndpoint.Window,
		)
	}
	if config.JWKSEndpoint.Enabled {
		service.limiters["jwks"] = NewTokenBucketLimiter(
			config.JWKSEndpoint.BurstSize,
			config.JWKSEndpoint.RequestsPerIP,
			config.JWKSEndpoint.Window,
		)
	}
	if config.UserinfoEndpoint.Enabled {
		service.limiters["userinfo"] = NewTokenBucketLimiter(
			config.UserinfoEndpoint.BurstSize,
			config.UserinfoEndpoint.RequestsPerIP,
			config.UserinfoEndpoint.Window,
		)
	}

	// Global limiters
	if config.GlobalIPLimit != nil && config.GlobalIPLimit.Enabled {
		service.limiters["global_ip"] = NewTokenBucketLimiter(
			config.GlobalIPLimit.BurstSize,
			config.GlobalIPLimit.RequestsPerIP,
			config.GlobalIPLimit.Window,
		)
	}
	if config.GlobalClientLimit != nil && config.GlobalClientLimit.Enabled {
		service.limiters["global_client"] = NewTokenBucketLimiter(
			config.GlobalClientLimit.BurstSize,
			config.GlobalClientLimit.RequestsPerIP,
			config.GlobalClientLimit.Window,
		)
	}

	return service
}

// CheckLimit checks if a request should be allowed for a specific endpoint.
func (s *RateLimitService) CheckLimit(ctx context.Context, endpoint, ip, clientID string) error {
	if !s.config.Enabled {
		return nil
	}

	// Check global IP limit first
	if s.config.GlobalIPLimit != nil && s.config.GlobalIPLimit.Enabled {
		limiter := s.limiters["global_ip"]
		allowed, err := limiter.Allow(ctx, fmt.Sprintf("ip:%s", ip))
		if err != nil {
			return fmt.Errorf("rate limit check failed: %w", err)
		}
		if !allowed {
			return s.rateLimitError("global IP limit exceeded", s.config.GlobalIPLimit.Window)
		}
	}

	// Check global client limit
	if clientID != "" && s.config.GlobalClientLimit != nil && s.config.GlobalClientLimit.Enabled {
		limiter := s.limiters["global_client"]
		allowed, err := limiter.Allow(ctx, fmt.Sprintf("client:%s", clientID))
		if err != nil {
			return fmt.Errorf("rate limit check failed: %w", err)
		}
		if !allowed {
			return s.rateLimitError("client limit exceeded", s.config.GlobalClientLimit.Window)
		}
	}

	// Check endpoint-specific limit
	limiter, exists := s.limiters[endpoint]
	if !exists {
		return nil // No limit configured for this endpoint
	}

	key := fmt.Sprintf("%s:ip:%s", endpoint, ip)
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	if !allowed {
		requests, window := limiter.GetLimit()
		return &OIDCError{
			ErrorCode:        "rate_limit_exceeded",
			ErrorDescription: fmt.Sprintf("Rate limit exceeded: %d requests per %s", requests, window),
		}
	}

	return nil
}

// rateLimitError creates a rate limit error with retry-after information.
func (s *RateLimitService) rateLimitError(message string, window time.Duration) error {
	return &OIDCError{
		ErrorCode:        "rate_limit_exceeded",
		ErrorDescription: fmt.Sprintf("%s. Retry after %s", message, window),
	}
}

// Reset resets rate limits for a specific key.
func (s *RateLimitService) Reset(ctx context.Context, endpoint, key string) error {
	limiter, exists := s.limiters[endpoint]
	if !exists {
		return fmt.Errorf("no limiter found for endpoint: %s", endpoint)
	}
	return limiter.Reset(ctx, key)
}

// Close closes all rate limiters.
func (s *RateLimitService) Close() error {
	for _, limiter := range s.limiters {
		if err := limiter.Close(); err != nil {
			return err
		}
	}
	return nil
}

// RateLimitMiddleware creates an HTTP middleware for rate limiting.
func RateLimitMiddleware(service *RateLimitService, endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address
			ip := getClientIP(r)

			// Extract client ID from request (if available)
			clientID := r.FormValue("client_id")
			if clientID == "" {
				// Try to get from basic auth
				username, _, _ := r.BasicAuth()
				clientID = username
			}

			// Check rate limit
			if err := service.CheckLimit(r.Context(), endpoint, ip, clientID); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", service.config.TokenEndpoint.RequestsPerIP))
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(service.config.TokenEndpoint.Window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)

				if oidcErr, ok := err.(*OIDCError); ok {
					w.Write([]byte(fmt.Sprintf(`{"error":"%s","error_description":"%s"}`,
						oidcErr.ErrorCode, oidcErr.ErrorDescription)))
				} else {
					w.Write([]byte(`{"error":"rate_limit_exceeded","error_description":"Too many requests"}`))
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For, X-Real-IP headers, and falls back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if ip := parseForwardedIP(xff); ip != "" {
			return ip
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// parseForwardedIP extracts the first valid IP from X-Forwarded-For header.
func parseForwardedIP(xff string) string {
	// X-Forwarded-For can be: "client, proxy1, proxy2"
	for i := 0; i < len(xff); i++ {
		if xff[i] == ',' {
			if ip := net.ParseIP(xff[:i]); ip != nil {
				return ip.String()
			}
			break
		}
	}
	if ip := net.ParseIP(xff); ip != nil {
		return ip.String()
	}
	return ""
}
