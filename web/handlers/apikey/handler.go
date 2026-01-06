package apikey

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/apikey"
)

// Handler handles API key requests
type Handler struct {
	manager *apikey.Manager
}

// NewHandler creates a new API key handler
func NewHandler(manager *apikey.Manager) *Handler {
	return &Handler{manager: manager}
}

// RegisterRoutes registers the API key routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/v1/apikeys")
	h.RegisterGroupRoutes(group)
}

// RegisterGroupRoutes registers routes to a specific group
func (h *Handler) RegisterGroupRoutes(group *gin.RouterGroup) {
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/:id", h.Get)
	group.PUT("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
	group.POST("/:id/rotate", h.RotateSecret)
	group.GET("/:id/stats", h.Stats)
}

// Create creates a new API key
func (h *Handler) Create(c *gin.Context) {
	var req apikey.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine user ID (from auth context or header for now)
	userID := c.GetString("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
			return
		}
	}

	apiKey, err := h.manager.CreateAPIKey(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

// List lists API keys
func (h *Handler) List(c *gin.Context) {
	var query apikey.ListAPIKeysQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force user_id filter if not admin (placeholder logic)
	// contextUserID := c.GetString("user_id")
	// if contextUserID != "" {
	// 	query.UserID = contextUserID
	// }

	keys, total, err := h.manager.ListAPIKeys(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  keys,
		"total": total,
	})
}

// Get gets an API key
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	apiKey, err := h.manager.GetAPIKey(c.Request.Context(), id)
	if err == apikey.ErrAPIKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// Update updates an API key
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req apikey.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.manager.UpdateAPIKey(c.Request.Context(), id, &req)
	if err == apikey.ErrAPIKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// Delete deletes an API key
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.manager.DeleteAPIKey(c.Request.Context(), id)
	if err == apikey.ErrAPIKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	c.Status(http.StatusNoContent)
}

// RotateSecret rotates the secret for an API key
func (h *Handler) RotateSecret(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.manager.RegenerateSecret(c.Request.Context(), id)
	if err == apikey.ErrAPIKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate secret"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Stats gets statistics for an API key
func (h *Handler) Stats(c *gin.Context) {
	id := c.Param("id")
	stats, err := h.manager.GetAPIKeyStats(c.Request.Context(), id)
	if err == apikey.ErrAPIKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
