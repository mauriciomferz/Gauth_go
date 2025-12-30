// Package security - Input validation and sanitization middleware
package security

import (
	"html"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// InputValidator provides input validation and sanitization
type InputValidator struct {
	maxBodySize int64
	patterns    map[string]*regexp.Regexp
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{
		maxBodySize: getEnvInt64("AGENTAUTH_MAX_BODY_SIZE", 1024*1024), // 1MB default
		patterns: map[string]*regexp.Regexp{
			"sql_injection":  regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|script|javascript|<script)`),
			"path_traversal": regexp.MustCompile(`\.\./|\.\.\\`),
			"xss":            regexp.MustCompile(`(?i)<script|javascript:|onerror=|onload=`),
		},
	}
}

// InputValidationMiddleware creates middleware for input validation
func InputValidationMiddleware(validator *InputValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate request body size
		if c.Request.ContentLength > validator.maxBodySize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":    "Request body too large",
				"max_size": validator.maxBodySize,
			})
			c.Abort()
			return
		}

		// Validate query parameters
		if !validator.ValidateQueryParams(c) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid query parameters detected",
			})
			c.Abort()
			return
		}

		// Validate headers
		if !validator.ValidateHeaders(c) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid headers detected",
			})
			c.Abort()
			return
		}

		// Sanitize path
		if validator.ContainsPathTraversal(c.Request.URL.Path) {
			LogSecurityEvent("path_traversal_attempt", "validate_path", "blocked",
				"Path traversal attempt detected",
				map[string]interface{}{
					"path":      c.Request.URL.Path,
					"client_ip": getClientIP(c),
				})

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid path",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateQueryParams validates query parameters for malicious content
func (v *InputValidator) ValidateQueryParams(c *gin.Context) bool {
	for key, values := range c.Request.URL.Query() {
		// Check key
		if v.ContainsSQLInjection(key) || v.ContainsXSS(key) {
			LogSecurityEvent("malicious_query_param", "validate_query", "blocked",
				"Malicious query parameter detected",
				map[string]interface{}{
					"param_key": key,
					"client_ip": getClientIP(c),
				})
			return false
		}

		// Check values
		for _, value := range values {
			if v.ContainsSQLInjection(value) || v.ContainsXSS(value) {
				LogSecurityEvent("malicious_query_value", "validate_query", "blocked",
					"Malicious query value detected",
					map[string]interface{}{
						"param_key":   key,
						"param_value": value,
						"client_ip":   getClientIP(c),
					})
				return false
			}
		}
	}

	return true
}

// ValidateHeaders validates request headers
func (v *InputValidator) ValidateHeaders(c *gin.Context) bool {
	// Check for abnormally long headers
	maxHeaderSize := 8192 // 8KB

	for key, values := range c.Request.Header {
		for _, value := range values {
			if len(value) > maxHeaderSize {
				LogSecurityEvent("oversized_header", "validate_headers", "blocked",
					"Oversized header detected",
					map[string]interface{}{
						"header_key":  key,
						"header_size": len(value),
						"client_ip":   getClientIP(c),
					})
				return false
			}

			// Check for XSS in headers
			if v.ContainsXSS(value) {
				LogSecurityEvent("xss_in_header", "validate_headers", "blocked",
					"XSS attempt in header",
					map[string]interface{}{
						"header_key": key,
						"client_ip":  getClientIP(c),
					})
				return false
			}
		}
	}

	return true
}

// ContainsSQLInjection checks if a string contains SQL injection patterns
func (v *InputValidator) ContainsSQLInjection(s string) bool {
	return v.patterns["sql_injection"].MatchString(s)
}

// ContainsXSS checks if a string contains XSS patterns
func (v *InputValidator) ContainsXSS(s string) bool {
	return v.patterns["xss"].MatchString(s)
}

// ContainsPathTraversal checks if a string contains path traversal patterns
func (v *InputValidator) ContainsPathTraversal(s string) bool {
	return v.patterns["path_traversal"].MatchString(s)
}

// SanitizeString sanitizes a string by removing/encoding dangerous characters
func SanitizeString(s string) string {
	// HTML encode
	s = html.EscapeString(s)

	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Remove control characters
	s = removeControlCharacters(s)

	return s
}

// SanitizeJSON sanitizes a JSON map recursively
func SanitizeJSON(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		// Sanitize key
		cleanKey := SanitizeString(key)

		// Sanitize value based on type
		switch v := value.(type) {
		case string:
			result[cleanKey] = SanitizeString(v)
		case map[string]interface{}:
			result[cleanKey] = SanitizeJSON(v)
		case []interface{}:
			result[cleanKey] = sanitizeArray(v)
		default:
			result[cleanKey] = v
		}
	}

	return result
}

// sanitizeArray sanitizes an array of interface{} values
func sanitizeArray(arr []interface{}) []interface{} {
	result := make([]interface{}, len(arr))

	for i, item := range arr {
		switch v := item.(type) {
		case string:
			result[i] = SanitizeString(v)
		case map[string]interface{}:
			result[i] = SanitizeJSON(v)
		case []interface{}:
			result[i] = sanitizeArray(v)
		default:
			result[i] = v
		}
	}

	return result
}

// removeControlCharacters removes control characters from a string
func removeControlCharacters(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		// Keep newline, tab, and carriage return
		if r == '\n' || r == '\t' || r == '\r' {
			result.WriteRune(r)
			continue
		}

		// Remove other control characters
		if r >= 32 || r == 127 {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ValidateEmail validates an email address format
func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateURL validates a URL format
func ValidateURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(:[0-9]+)?(/.*)?$`)
	return urlRegex.MatchString(url)
}

// ValidateClientID validates a client ID format
func ValidateClientID(clientID string) bool {
	// Client ID should be alphanumeric with hyphens and underscores
	clientIDRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
	return clientIDRegex.MatchString(clientID)
}

// ValidateScope validates a scope format
func ValidateScope(scope string) bool {
	// Scopes should be alphanumeric with dots, hyphens, and underscores
	scopeRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	return scopeRegex.MatchString(scope)
}

// ValidateJSONKeys validates JSON keys are safe
func ValidateJSONKeys(data map[string]interface{}) bool {
	keyRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

	for key := range data {
		if !keyRegex.MatchString(key) {
			return false
		}

		// Recursively check nested objects
		if nested, ok := data[key].(map[string]interface{}); ok {
			if !ValidateJSONKeys(nested) {
				return false
			}
		}
	}

	return true
}

// CSRFProtectionMiddleware provides CSRF protection
func CSRFProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for GET, HEAD, OPTIONS
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Check Origin header
		origin := c.GetHeader("Origin")
		referer := c.GetHeader("Referer")

		if origin == "" && referer == "" {
			// No origin or referer - could be CSRF
			LogSecurityEvent("csrf_no_origin", "csrf_check", "warning",
				"Request with no origin or referer",
				map[string]interface{}{
					"method":    c.Request.Method,
					"path":      c.Request.URL.Path,
					"client_ip": getClientIP(c),
				})
		}

		// In production, verify origin matches allowed origins
		if origin != "" && !isOriginAllowed(origin, loadAllowedOrigins(), false) {
			LogSecurityEvent("csrf_invalid_origin", "csrf_check", "blocked",
				"CSRF attempt with invalid origin",
				map[string]interface{}{
					"origin":    origin,
					"client_ip": getClientIP(c),
				})

			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid origin",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getEnvInt64 retrieves an int64 from environment with default
func getEnvInt64(key string, defaultValue int64) int64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}

	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return defaultValue
	}

	return val
}
