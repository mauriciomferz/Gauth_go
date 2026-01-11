// Package security - Rate limiting and DDoS protection middleware
package security

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig defines rate limiting configuration
type RateLimiterConfig struct {
	RequestsPerSecond int           // Requests per second allowed
	BurstSize         int           // Burst capacity
	CleanupInterval   time.Duration // Interval to clean up old limiters
}

// IPRateLimiter manages rate limiters per IP address
type IPRateLimiter struct {
	limiters    map[string]*rate.Limiter
	mu          sync.RWMutex
	config      RateLimiterConfig
	lastCleanup time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter
func NewIPRateLimiter(config RateLimiterConfig) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters:    make(map[string]*rate.Limiter),
		config:      config,
		lastCleanup: time.Now(),
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// GetLimiter returns the rate limiter for a given IP
func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.limiters[ip]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if existingLimiter, exists := l.limiters[ip]; exists {
		return existingLimiter
	}

	limiter = rate.NewLimiter(
		rate.Limit(l.config.RequestsPerSecond),
		l.config.BurstSize,
	)
	l.limiters[ip] = limiter

	return limiter
}

// cleanup periodically removes idle limiters
func (l *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		// Simple cleanup: remove all limiters periodically
		// In production, consider tracking last access time
		if time.Since(l.lastCleanup) > l.config.CleanupInterval*2 {
			l.limiters = make(map[string]*rate.Limiter)
			l.lastCleanup = time.Now()
		}
		l.mu.Unlock()
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c)

		// Get rate limiter for this IP
		ipLimiter := limiter.GetLimiter(ip)

		// Check if request is allowed
		if !ipLimiter.Allow() {
			// Set headers for rate limit info
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerSecond))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))

			// Log rate limit event
			c.Set("security_event", "rate_limit_exceeded")
			c.Set("rate_limited", true)
			c.Set("client_ip", ip)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Too many requests. Please try again later.",
				"retry_after": 1,
			})
			c.Abort()
			return
		}

		// Add rate limit headers to successful requests
		tokens := ipLimiter.Tokens()
		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(int(tokens)))

		c.Next()
	}
}

// EndpointRateLimiter manages rate limits for specific endpoints
type EndpointRateLimiter struct {
	limits map[string]*IPRateLimiter
	mu     sync.RWMutex
}

// NewEndpointRateLimiter creates a new endpoint-specific rate limiter
func NewEndpointRateLimiter() *EndpointRateLimiter {
	return &EndpointRateLimiter{
		limits: make(map[string]*IPRateLimiter),
	}
}

// AddEndpoint adds rate limiting for a specific endpoint
func (e *EndpointRateLimiter) AddEndpoint(pattern string, config RateLimiterConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.limits[pattern] = NewIPRateLimiter(config)
}

// GetLimiter returns the rate limiter for a specific endpoint
func (e *EndpointRateLimiter) GetLimiter(path string) *IPRateLimiter {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for pattern, limiter := range e.limits {
		if matchesPattern(path, pattern) {
			return limiter
		}
	}

	return nil
}

// matchesPattern checks if a path matches a pattern (simple prefix matching)
func matchesPattern(path, pattern string) bool {
	// Simple prefix matching
	// In production, consider using a proper path matcher
	return len(path) >= len(pattern) && path[:len(pattern)] == pattern
}

// EndpointRateLimitMiddleware creates endpoint-specific rate limiting
func EndpointRateLimitMiddleware(limiter *EndpointRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Get endpoint-specific limiter
		endpointLimiter := limiter.GetLimiter(path)
		if endpointLimiter == nil {
			// No specific limit for this endpoint
			c.Next()
			return
		}

		ip := getClientIP(c)
		ipLimiter := endpointLimiter.GetLimiter(ip)

		if !ipLimiter.Allow() {
			c.Set("security_event", "endpoint_rate_limit_exceeded")
			c.Set("rate_limited", true)
			c.Set("client_ip", ip)
			c.Set("endpoint", path)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":    "Rate limit exceeded for this endpoint",
				"message":  "Too many requests to this endpoint. Please try again later.",
				"endpoint": path,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for idx := 0; idx < len(xff); idx++ {
			if xff[idx] == ',' {
				return xff[:idx]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimiterConfig {
	// Load from environment or use defaults
	rps := getEnvInt("AGENTAUTH_RATE_LIMIT_RPS", 100)
	burst := getEnvInt("AGENTAUTH_RATE_LIMIT_BURST", 200)

	return RateLimiterConfig{
		RequestsPerSecond: rps,
		BurstSize:         burst,
		CleanupInterval:   5 * time.Minute,
	}
}

// StrictRateLimitConfig returns strict rate limiting for sensitive endpoints
func StrictRateLimitConfig() RateLimiterConfig {
	rps := getEnvInt("AGENTAUTH_STRICT_RATE_LIMIT_RPS", 10)
	burst := getEnvInt("AGENTAUTH_STRICT_RATE_LIMIT_BURST", 20)

	return RateLimiterConfig{
		RequestsPerSecond: rps,
		BurstSize:         burst,
		CleanupInterval:   5 * time.Minute,
	}
}

// getEnvInt retrieves an integer from environment with a default value
func getEnvInt(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}

	return val
}

// ConfigureEndpointRateLimits sets up rate limiting for specific endpoints
func ConfigureEndpointRateLimits() *EndpointRateLimiter {
	limiter := NewEndpointRateLimiter()

	// Strict limits for authentication and token endpoints
	strictConfig := StrictRateLimitConfig()
	limiter.AddEndpoint("/api/v1/beta/tokens", strictConfig)
	limiter.AddEndpoint("/api/v1/beta/delegation", strictConfig)
	limiter.AddEndpoint("/api/v1/beta/pvp/verify", strictConfig)

	// Moderate limits for read operations
	moderateConfig := RateLimiterConfig{
		RequestsPerSecond: getEnvInt("AGENTAUTH_MODERATE_RATE_LIMIT_RPS", 50),
		BurstSize:         getEnvInt("AGENTAUTH_MODERATE_RATE_LIMIT_BURST", 100),
		CleanupInterval:   5 * time.Minute,
	}
	limiter.AddEndpoint("/api/v1/beta/subscriptions", moderateConfig)
	limiter.AddEndpoint("/api/v1/beta/pip", moderateConfig)
	limiter.AddEndpoint("/api/v1/beta/registry", moderateConfig)

	return limiter
}

// DDoSProtectionMiddleware provides additional DDoS protection
func DDoSProtectionMiddleware() gin.HandlerFunc {
	// Track request counts per IP for aggressive rate checking
	requestCounts := make(map[string]*ddosCounter)
	var mu sync.RWMutex

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, counter := range requestCounts {
				if now.Sub(counter.lastReset) > 1*time.Minute {
					delete(requestCounts, ip)
				}
			}
			mu.Unlock()
		}
	}()

	// DDoS detection threshold
	maxRequestsPerSecond := getEnvInt("AGENTAUTH_DDOS_MAX_RPS", 1000)

	return func(c *gin.Context) {
		ip := getClientIP(c)

		mu.Lock()
		counter, exists := requestCounts[ip]
		if !exists {
			counter = &ddosCounter{
				count:     0,
				lastReset: time.Now(),
			}
			requestCounts[ip] = counter
		}

		// Reset counter if more than 1 second has passed
		if time.Since(counter.lastReset) > 1*time.Second {
			counter.count = 0
			counter.lastReset = time.Now()
		}

		counter.count++
		currentCount := counter.count
		mu.Unlock()

		// Check if exceeds DDoS threshold
		if currentCount > maxRequestsPerSecond {
			c.Set("security_event", "ddos_detected")
			c.Set("client_ip", ip)
			c.Set("request_count", currentCount)

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Service temporarily unavailable",
				"message": "Too many requests detected. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ddosCounter tracks requests per IP for DDoS detection
type ddosCounter struct {
	count     int
	lastReset time.Time
}

// RateLimitMetrics tracks rate limiting metrics
type RateLimitMetrics struct {
	TotalRequests uint64
	RateLimited   uint64
	DDosBlocked   uint64
}

// Global metrics instance
var globalRateLimitMetrics = &RateLimitMetrics{}

// GetRateLimitMetrics returns current rate limit metrics
func GetRateLimitMetrics() RateLimitMetrics {
	return *globalRateLimitMetrics
}

// RateLimitMetricsMiddleware tracks rate limiting metrics
func RateLimitMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Track metrics
		globalRateLimitMetrics.TotalRequests++

		if c.GetBool("rate_limited") {
			globalRateLimitMetrics.RateLimited++
		}

		if event, exists := c.Get("security_event"); exists {
			if eventStr, ok := event.(string); ok && eventStr == "ddos_detected" {
				globalRateLimitMetrics.DDosBlocked++
			}
		}
	}
}
