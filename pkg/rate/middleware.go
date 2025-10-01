package rate

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

// HTTPLimiterConfig configures HTTP rate limiting
type HTTPLimiterConfig struct {
	// Limiter is the rate limiter to use
	Limiter Limiter

	// KeyFunc generates the rate limit key from the request
	KeyFunc func(*http.Request) string

	// ExcludeFunc determines if a request should be excluded from rate limiting
	ExcludeFunc func(*http.Request) bool

	// StatusCode is the HTTP status code to return when rate limit is exceeded
	StatusCode int

	// Message is the error message to return when rate limit is exceeded
	Message string

	// Headers determines if rate limit headers should be included in responses
	Headers bool
}

// Middleware creates a new HTTP middleware for rate limiting
func Middleware(cfg HTTPLimiterConfig) func(http.Handler) http.Handler {
	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusTooManyRequests
	}

	if cfg.Message == "" {
		cfg.Message = "rate limit exceeded"
	}

	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultKeyFunc
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request should be excluded
			if cfg.ExcludeFunc != nil && cfg.ExcludeFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Get rate limit key
			key := cfg.KeyFunc(r)

			// Check rate limit
			err := cfg.Limiter.Allow(r.Context(), key)
			if err == ErrRateLimitExceeded {
				if cfg.Headers {
					setRateLimitHeaders(w, cfg.Limiter.GetRemainingRequests(key))
				}
				http.Error(w, cfg.Message, cfg.StatusCode)
				return
			}

			if cfg.Headers {
				setRateLimitHeaders(w, cfg.Limiter.GetRemainingRequests(key))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultKeyFunc generates a rate limit key from the request IP
func DefaultKeyFunc(r *http.Request) string {
	ip := getIP(r)
	return "ip:" + ip
}

// UserKeyFunc generates a rate limit key from the user ID
func UserKeyFunc(userIDFunc func(*http.Request) string) func(*http.Request) string {
	return func(r *http.Request) string {
		userID := userIDFunc(r)
		if userID == "" {
			return DefaultKeyFunc(r)
		}
		return "user:" + userID
	}
}

// EndpointKeyFunc generates a rate limit key from the request path
func EndpointKeyFunc(r *http.Request) string {
	return "endpoint:" + r.Method + ":" + r.URL.Path
}

// CombinedKeyFunc combines multiple key functions
func CombinedKeyFunc(funcs ...func(*http.Request) string) func(*http.Request) string {
	return func(r *http.Request) string {
		var keys []string
		for _, fn := range funcs {
			keys = append(keys, fn(r))
		}
		return strings.Join(keys, ":")
	}
}

// getIP extracts the client IP from the request
func getIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// setRateLimitHeaders sets rate limit headers on the response
func setRateLimitHeaders(w http.ResponseWriter, remaining int64) {
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
}

// ExcludeHealthChecks excludes health check endpoints from rate limiting
func ExcludeHealthChecks(paths ...string) func(*http.Request) bool {
	healthPaths := make(map[string]bool)
	for _, path := range paths {
		healthPaths[path] = true
	}

	if len(paths) == 0 {
		healthPaths["/health"] = true
		healthPaths["/healthz"] = true
		healthPaths["/ready"] = true
		healthPaths["/readyz"] = true
		healthPaths["/live"] = true
		healthPaths["/livez"] = true
	}

	return func(r *http.Request) bool {
		return healthPaths[r.URL.Path]
	}
}

// ExcludeMethod excludes specific HTTP methods from rate limiting
func ExcludeMethod(methods ...string) func(*http.Request) bool {
	excluded := make(map[string]bool)
	for _, method := range methods {
		excluded[strings.ToUpper(method)] = true
	}

	return func(r *http.Request) bool {
		return excluded[r.Method]
	}
}

// ExcludePathPrefix excludes paths with specific prefixes from rate limiting
func ExcludePathPrefix(prefixes ...string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		path := r.URL.Path
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		return false
	}
}
