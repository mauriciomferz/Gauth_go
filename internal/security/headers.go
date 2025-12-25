// Package security provides comprehensive security middleware for the GAuth application,
// including security headers, CORS configuration, rate limiting, input validation,
// and audit logging capabilities.
package security

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds comprehensive security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	// Load configuration from environment
	isDevelopment := os.Getenv("GAUTH_ENV") == "development"

	// Content Security Policy
	csp := buildContentSecurityPolicy(isDevelopment)

	// Allowed frame ancestors (for X-Frame-Options alternative)
	frameAncestors := os.Getenv("GAUTH_FRAME_ANCESTORS")
	if frameAncestors == "" {
		frameAncestors = "'none'" // Default: no framing
	}

	return func(c *gin.Context) {
		// Content Security Policy
		c.Header("Content-Security-Policy", csp)

		// Strict Transport Security (HSTS) - only in production with HTTPS
		if !isDevelopment && c.Request.TLS != nil {
			maxAge := os.Getenv("GAUTH_HSTS_MAX_AGE")
			if maxAge == "" {
				maxAge = "31536000" // 1 year default
			}
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%s; includeSubDomains; preload", maxAge))
		}

		// X-Frame-Options (defense-in-depth with CSP frame-ancestors)
		frameOptions := os.Getenv("GAUTH_X_FRAME_OPTIONS")
		if frameOptions == "" {
			frameOptions = "DENY" // Default: no framing
		}
		c.Header("X-Frame-Options", frameOptions)

		// X-Content-Type-Options
		c.Header("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection (legacy browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer Policy
		referrerPolicy := os.Getenv("GAUTH_REFERRER_POLICY")
		if referrerPolicy == "" {
			referrerPolicy = "strict-origin-when-cross-origin"
		}
		c.Header("Referrer-Policy", referrerPolicy)

		// Permissions Policy (Feature Policy)
		permissionsPolicy := buildPermissionsPolicy()
		c.Header("Permissions-Policy", permissionsPolicy)

		// Remove server identification headers
		c.Header("Server", "")
		c.Header("X-Powered-By", "")

		// Cache control for sensitive endpoints
		if isSecureEndpoint(c.Request.URL.Path) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// buildContentSecurityPolicy constructs a Content Security Policy header value
func buildContentSecurityPolicy(isDevelopment bool) string {
	// Base CSP directives
	directives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'", // unsafe-inline needed for React inline styles
		"img-src 'self' data: https:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"media-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"upgrade-insecure-requests",
	}

	// Development mode: allow localhost and webpack dev server
	if isDevelopment {
		directives = []string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-eval' 'unsafe-inline'", // webpack HMR needs unsafe-eval
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https:",
			"font-src 'self' data:",
			"connect-src 'self' ws: wss: http://localhost:* http://127.0.0.1:*",
			"media-src 'none'",
			"object-src 'none'",
			"base-uri 'self'",
			"form-action 'self'",
			"frame-ancestors 'self'",
		}
	}

	// Allow custom CSP additions via environment
	customCSP := os.Getenv("GAUTH_CSP_ADDITIONS")
	if customCSP != "" {
		directives = append(directives, strings.Split(customCSP, ";")...)
	}

	return strings.Join(directives, "; ")
}

// buildPermissionsPolicy constructs a Permissions-Policy header value
func buildPermissionsPolicy() string {
	// Restrictive permissions policy
	policies := []string{
		"accelerometer=()",
		"camera=()",
		"geolocation=()",
		"gyroscope=()",
		"magnetometer=()",
		"microphone=()",
		"payment=()",
		"usb=()",
		"interest-cohort=()", // Disable FLoC
	}

	// Allow custom permissions via environment
	customPermissions := os.Getenv("GAUTH_PERMISSIONS_POLICY")
	if customPermissions != "" {
		policies = append(policies, strings.Split(customPermissions, ",")...)
	}

	return strings.Join(policies, ", ")
}

// isSecureEndpoint determines if a path should have strict cache control
func isSecureEndpoint(path string) bool {
	securePatterns := []string{
		"/api/v1/beta/tokens",
		"/api/v1/beta/delegation",
		"/api/v1/beta/pvp",
		"/api/v1/beta/audit",
		"/api/v1/beta/subscriptions",
		"/api/v1/beta/pip",
	}

	for _, pattern := range securePatterns {
		if strings.HasPrefix(path, pattern) {
			return true
		}
	}

	return false
}

// CORSMiddleware configures Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	// Load allowed origins from environment
	allowedOrigins := loadAllowedOrigins()
	isDevelopment := os.Getenv("GAUTH_ENV") == "development"

	// Allowed methods
	allowedMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}

	// Allowed headers
	allowedHeaders := []string{
		"Content-Type",
		"Authorization",
		"X-Requested-With",
		"Accept",
		"Origin",
		"X-Request-ID",
		"X-Correlation-ID",
	}

	// Exposed headers
	exposedHeaders := []string{
		"Content-Length",
		"Content-Type",
		"X-Request-ID",
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && isOriginAllowed(origin, allowedOrigins, isDevelopment) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
			c.Header("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
			c.Header("Access-Control-Max-Age", "86400") // 24 hours

			// Handle preflight requests
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		} else if origin != "" && !isDevelopment {
			// Log rejected origin in production
			c.Set("security_event", "cors_rejected")
			c.Set("rejected_origin", origin)
		}

		c.Next()
	}
}

// loadAllowedOrigins loads and parses allowed CORS origins from environment
func loadAllowedOrigins() []string {
	originsStr := os.Getenv("GAUTH_CORS_ALLOWED_ORIGINS")
	if originsStr == "" {
		// Default to localhost for development
		return []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
		}
	}

	// Parse comma-separated origins
	origins := strings.Split(originsStr, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// isOriginAllowed checks if an origin is in the allowed list or matches pattern
func isOriginAllowed(origin string, allowedOrigins []string, isDevelopment bool) bool {
	// Development mode: allow localhost and 127.0.0.1
	if isDevelopment {
		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			return true
		}
	}

	// Check exact matches
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true // Wildcard (not recommended for production)
		}
		if origin == allowed {
			return true
		}

		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:] // Remove "*."
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// SecureResponseMiddleware adds additional security to outgoing responses
func SecureResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Remove sensitive error details in production
		if os.Getenv("GAUTH_ENV") != "development" {
			if c.Writer.Status() >= 500 {
				// Log actual error but don't expose to client
				c.Set("security_event", "internal_error")
			}
		}

		// Add security context for monitoring
		if c.GetBool("rate_limited") {
			c.Header("X-RateLimit-Blocked", "true")
		}
	}
}
