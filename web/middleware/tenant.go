package middleware

import (
	"github.com/gin-gonic/gin"
)

// TenantMiddleware extracts tenant_id from request headers or query parameters
// and sets it in the Gin context for downstream handlers to use
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get tenant_id from header first (X-Tenant-ID)
		tenantID := c.GetHeader("X-Tenant-ID")

		// If not in header, try query parameter
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}

		// Set in context if found
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}

		c.Next()
	}
}
