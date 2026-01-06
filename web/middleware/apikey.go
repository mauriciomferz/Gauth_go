package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/apikey"
	"github.com/mauriciomferz/AgentAuth/pkg/ratelimit"
)

// APIKeyMiddleware handles API key authentication
type APIKeyMiddleware struct {
	manager *apikey.Manager
	limiter ratelimit.DynamicLimiter
}

// NewAPIKeyMiddleware creates a new API key middleware
func NewAPIKeyMiddleware(m *apikey.Manager, l ratelimit.DynamicLimiter) *APIKeyMiddleware {
	return &APIKeyMiddleware{manager: m, limiter: l}
}

// Authenticate returns a Gin middleware that validates API keys
func (m *APIKeyMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 1. Extract API Key
		key := extractAPIKey(c)
		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// 2. Validate API Key
		// We use a separate context for database operations to avoid cancellation issues if needed,
		// but using request context is standard.
		apiKeyRecord, err := m.manager.ValidateAPIKey(c.Request.Context(), key)
		if err != nil {
			status := http.StatusUnauthorized
			if err == apikey.ErrQuotaExceeded {
				status = http.StatusTooManyRequests
			}
			c.JSON(status, gin.H{"error": err.Error()})
			c.Abort()

			// Record failed attempt
			// We use a background context here for the same reason as success recording
			go func() {
				_ = m.manager.RecordUsage(context.Background(), &apikey.APIKeyUsage{
					Timestamp: time.Now(),
				})
			}()
			return
		}

		// 3. Rate Limiting (Dynamic)
		if m.limiter != nil {
			// Minute Limit
			if apiKeyRecord.RateLimitPerMinute > 0 && !m.limiter.AllowWithLimit(apiKeyRecord.KeyID, apiKeyRecord.RateLimitPerMinute, time.Minute) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "minute rate limit exceeded"})
				return
			}
			// Hour Limit
			if apiKeyRecord.RateLimitPerHour > 0 && !m.limiter.AllowWithLimit(apiKeyRecord.KeyID, apiKeyRecord.RateLimitPerHour, time.Hour) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "hourly rate limit exceeded"})
				return
			}
			// Daily Limit
			if apiKeyRecord.RateLimitPerDay > 0 && !m.limiter.AllowWithLimit(apiKeyRecord.KeyID, apiKeyRecord.RateLimitPerDay, 24*time.Hour) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "daily rate limit exceeded"})
				return
			}
		}

		// 4. Set context variables
		c.Set("api_key", apiKeyRecord)
		c.Set("api_key_id", apiKeyRecord.KeyID)
		if apiKeyRecord.UserID != "" {
			c.Set("user_id", apiKeyRecord.UserID)
		}
		// If scopes are in metadata, extract them (assuming "scopes" key list of strings)
		if scopes, ok := apiKeyRecord.Metadata["scopes"]; ok {
			c.Set("scopes", scopes)
		}

		// 5. Process Request
		c.Next()

		// 5. Record Usage (Async)
		// We use a background context because the request context might be cancelled
		go func(record *apikey.APIKey, path, method, ip, ua string, status int, latency time.Duration) {
			usage := &apikey.APIKeyUsage{
				KeyID:          record.KeyID,
				Endpoint:       path,
				Method:         method,
				StatusCode:     status,
				ResponseTimeMs: int(latency.Milliseconds()),
				RequestIP:      ip,
				UserAgent:      ua,
				Timestamp:      time.Now(),
			}

			if status >= 400 {
				// We could capture error message from context if set
				if errs := c.Errors; len(errs) > 0 {
					usage.ErrorMessage = errs.String()
				}
			}

			_ = m.manager.RecordUsage(context.Background(), usage)
		}(apiKeyRecord, c.Request.URL.Path, c.Request.Method, c.ClientIP(), c.Request.UserAgent(), c.Writer.Status(), time.Since(start))
	}
}

func extractAPIKey(c *gin.Context) string {
	// 1. Check X-API-Key header
	if key := c.GetHeader("X-API-Key"); key != "" {
		return key
	}

	// 2. Check Authorization header (Bearer)
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}
