package apikey

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyAPIKey is the context key for storing API key info
	ContextKeyAPIKey = "api_key"
)

// AuthMiddleware creates a Gin middleware for API key authentication
func AuthMiddleware(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Expected format: "Bearer sk_live_..."
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		secretKey := parts[1]

		// Validate API key
		apiKey, err := manager.ValidateAPIKey(c.Request.Context(), secretKey)
		if err != nil {
			switch err {
			case ErrAPIKeyNotFound, ErrAPIKeyInvalid:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid api key",
				})
			case ErrAPIKeyExpired:
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "api key expired",
				})
			case ErrAPIKeyDisabled:
				c.JSON(http.StatusForbidden, gin.H{
					"error": "api key disabled",
				})
			case ErrQuotaExceeded:
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "quota exceeded",
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
			c.Abort()
			return
		}

		// Check IP whitelist if configured
		if len(apiKey.IPWhitelist) > 0 {
			clientIP := c.ClientIP()
			if !contains(apiKey.IPWhitelist, clientIP) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "request from unauthorized IP address",
				})
				c.Abort()
				return
			}
		}

		// Check allowed endpoints if configured
		if len(apiKey.AllowedEndpoints) > 0 {
			requestPath := c.Request.URL.Path
			if !matchesEndpoint(apiKey.AllowedEndpoints, requestPath) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "endpoint not allowed for this API key",
				})
				c.Abort()
				return
			}
		}

		// Store API key in context
		c.Set(ContextKeyAPIKey, apiKey)

		// Record usage (async to not block request)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = manager.RecordUsage(ctx, &APIKeyUsage{
				KeyID:      apiKey.KeyID,
				Endpoint:   c.Request.URL.Path,
				Method:     c.Request.Method,
				StatusCode: c.Writer.Status(),
				RequestIP:  c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
				Timestamp:  time.Now(),
			})
		}()

		c.Next()
	}
}

// RateLimitMiddleware creates a Gin middleware for API key-specific rate limiting
func RateLimitMiddleware(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from context (set by AuthMiddleware)
		apiKeyInterface, exists := c.Get(ContextKeyAPIKey)
		if !exists {
			c.Next()
			return
		}

		apiKey, ok := apiKeyInterface.(*APIKey)
		if !ok {
			c.Next()
			return
		}

		// Check rate limits
		// This is a simplified implementation - production would use Redis or similar
		// for distributed rate limiting

		// For now, we'll just check the quota
		if apiKey.QuotaRequestsTotal != nil && apiKey.QuotaRequestsUsed >= *apiKey.QuotaRequestsTotal {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":           "quota exceeded",
				"quota_total":     *apiKey.QuotaRequestsTotal,
				"quota_used":      apiKey.QuotaRequestsUsed,
				"quota_remaining": 0,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func matchesEndpoint(allowedPatterns []string, requestPath string) bool {
	for _, pattern := range allowedPatterns {
		// Simple pattern matching - in production, use a proper glob/regex library
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(requestPath, prefix) {
				return true
			}
		} else if pattern == requestPath {
			return true
		}
	}
	return false
}
