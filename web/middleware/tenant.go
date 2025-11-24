package middleware
package middleware

import (
	"github.com/gin-gonic/gin"
)

// TenantMiddleware extracts tenant_id from request headers or query parameters
// and sets it in the Gin context for downstream handlers to use


















}	}		c.Next()				}			c.Set("tenant_id", tenantID)		if tenantID != "" {		// Set in context if found				}			tenantID = c.Query("tenant_id")		if tenantID == "" {		// If not in header, try query parameter				tenantID := c.GetHeader("X-Tenant-ID")		// Try to get tenant_id from header first (X-Tenant-ID)	return func(c *gin.Context) {func TenantMiddleware() gin.HandlerFunc {