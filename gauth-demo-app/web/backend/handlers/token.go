package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// TokenHandler handles token management operations
type TokenHandler struct {
	service *services.GAuthService
	logger  *logrus.Logger
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(service *services.GAuthService, logger *logrus.Logger) *TokenHandler {
	return &TokenHandler{
		service: service,
		logger:  logger,
	}
}

// CreateToken creates a new token
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req services.CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid token creation request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Set default duration if not provided
	if req.Duration == 0 {
		req.Duration = time.Hour // 1 hour default
	}

	// Create token through service
	token, err := h.service.CreateToken(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create token",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Token created successfully")
	c.JSON(http.StatusCreated, token)
}

// GetTokens retrieves tokens with pagination
func (h *TokenHandler) GetTokens(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	ownerID := c.Query("owner_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := services.GetTokensRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		OwnerID:  ownerID,
	}

	tokens, err := h.service.GetTokens(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tokens")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get tokens",
		})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// RevokeToken revokes a specific token
func (h *TokenHandler) RevokeToken(c *gin.Context) {
	tokenID := c.Param("id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token ID is required",
		})
		return
	}

	err := h.service.RevokeToken(c.Request.Context(), tokenID)
	if err != nil {
		h.logger.WithError(err).WithField("token_id", tokenID).Error("Failed to revoke token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to revoke token",
		})
		return
	}

	h.logger.WithField("token_id", tokenID).Info("Token revoked successfully")
	c.JSON(http.StatusOK, gin.H{
		"message":   "Token revoked successfully",
		"token_id":  tokenID,
		"revoked_at": time.Now(),
	})
}

// ValidateToken validates a token
func (h *TokenHandler) ValidateToken(c *gin.Context) {
	var req struct {
		AccessToken string `json:"access_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	claims, err := h.service.ValidateToken(c.Request.Context(), req.AccessToken)
	if err != nil {
		h.logger.WithError(err).Error("Token validation failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"claims": claims,
	})
}

// RefreshToken refreshes an access token
func (h *TokenHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	newToken, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Error("Token refresh failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token refresh failed",
		})
		return
	}

	h.logger.Info("Token refreshed successfully")
	c.JSON(http.StatusOK, newToken)
}

// GetTokenMetrics returns token usage metrics
func (h *TokenHandler) GetTokenMetrics(c *gin.Context) {
	metrics, err := h.service.GetTokenMetrics(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get token metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get token metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}
