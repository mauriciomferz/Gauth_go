package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/cache"
)

// CacheHandler handles cache-related HTTP requests
type CacheHandler struct {
	cache      cache.Cache
	keyBuilder *cache.KeyBuilder
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(c cache.Cache) *CacheHandler {
	return &CacheHandler{
		cache:      c,
		keyBuilder: cache.NewKeyBuilder(),
	}
}

// RegisterRoutes registers cache management routes
func (h *CacheHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Admin routes for cache management
	r.GET("/cache/stats", h.GetCacheStats)
	r.POST("/cache/clear", h.ClearCache)
	r.POST("/cache/clear/:pattern", h.ClearCachePattern)
	r.GET("/cache/health", h.CheckCacheHealth)
	r.POST("/cache/invalidate/poa/:id", h.InvalidatePoA)
	r.POST("/cache/invalidate/user/:id", h.InvalidateUser)
}

// GetCacheStats handles GET /api/v1/admin/cache/stats
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	stats, err := h.cache.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get cache stats",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ClearCache handles POST /api/v1/admin/cache/clear
func (h *CacheHandler) ClearCache(c *gin.Context) {
	// Clear all keys with gauth: prefix
	err := h.cache.DeletePattern(c.Request.Context(), "gauth:*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clear cache",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache cleared successfully",
	})
}

// ClearCachePattern handles POST /api/v1/admin/cache/clear/:pattern
func (h *CacheHandler) ClearCachePattern(c *gin.Context) {
	pattern := c.Param("pattern")
	if pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Pattern is required",
		})
		return
	}

	// Add gauth: prefix if not present
	if len(pattern) < 6 || pattern[:6] != "gauth:" {
		pattern = "gauth:" + pattern
	}

	err := h.cache.DeletePattern(c.Request.Context(), pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clear cache pattern",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache pattern cleared successfully",
		"pattern": pattern,
	})
}

// CheckCacheHealth handles GET /api/v1/admin/cache/health
func (h *CacheHandler) CheckCacheHealth(c *gin.Context) {
	err := h.cache.Ping(c.Request.Context())
	if err != nil {
		stats, _ := h.cache.GetStats(c.Request.Context())
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": err.Error(),
			"stats":   stats,
		})
		return
	}

	stats, _ := h.cache.GetStats(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"stats":  stats,
	})
}

// InvalidatePoA handles POST /api/v1/admin/cache/invalidate/poa/:id
func (h *CacheHandler) InvalidatePoA(c *gin.Context) {
	poaID := c.Param("id")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "PoA ID is required",
		})
		return
	}

	// Invalidate all PoA-related cache entries
	pattern := h.keyBuilder.InvalidatePoAPattern(poaID)
	err := h.cache.DeletePattern(c.Request.Context(), pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to invalidate PoA cache",
			"message": err.Error(),
		})
		return
	}

	// Also delete specific keys
	keys := []string{
		h.keyBuilder.PoAKey(poaID),
		h.keyBuilder.VerificationKey(poaID),
		h.keyBuilder.BlockchainSyncKey(poaID),
		h.keyBuilder.BlockchainVerifyKey(poaID),
	}
	for _, key := range keys {
		_ = h.cache.Delete(c.Request.Context(), key)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "PoA cache invalidated successfully",
		"poa_id":  poaID,
	})
}

// InvalidateUser handles POST /api/v1/admin/cache/invalidate/user/:id
func (h *CacheHandler) InvalidateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	// Invalidate all user-related cache entries
	pattern := h.keyBuilder.InvalidateUserPattern(userID)
	err := h.cache.DeletePattern(c.Request.Context(), pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to invalidate user cache",
			"message": err.Error(),
		})
		return
	}

	// Also delete specific keys
	keys := []string{
		h.keyBuilder.UserKey(userID),
		h.keyBuilder.PoAListKey(userID),
	}
	for _, key := range keys {
		_ = h.cache.Delete(c.Request.Context(), key)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User cache invalidated successfully",
		"user_id": userID,
	})
}
