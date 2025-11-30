package admin

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mauriciomferz/Gauth_go/pkg/redis"
	"github.com/mauriciomferz/Gauth_go/pkg/tokens"
)

// TokenHandler manages token operations for the admin portal
type TokenHandler struct {
	repo  *tokens.Repository
	redis *redis.Client
}

// NewTokenHandler creates a new token handler instance
func NewTokenHandler(db *pgxpool.Pool, redisClient *redis.Client) *TokenHandler {
	return &TokenHandler{
		repo:  tokens.NewRepository(db),
		redis: redisClient,
	}
}

// TokenRequest represents the request to create a new token
type TokenRequest struct {
	SubscriberID string   `json:"subscriberId" binding:"required"`
	TokenType    string   `json:"tokenType" binding:"required,oneof=access refresh"`
	TTL          int      `json:"ttl" binding:"required,min=60"`
	Scopes       []string `json:"scopes"`
}

// Token represents a token in the system
type Token struct {
	ID             string    `json:"id"`
	SubscriberID   string    `json:"subscriberId"`
	SubscriberName string    `json:"subscriberName"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	LastUsed       time.Time `json:"lastUsed"`
	UsageCount     int       `json:"usageCount"`
}

// TokenResponse represents the response when creating a token
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	Type      string `json:"type"`
}

// TokenListResponse represents the list of tokens
type TokenListResponse struct {
	Tokens []Token `json:"tokens"`
	Total  int     `json:"total"`
}

// ValidateTokenRequest represents the request to validate a token
type ValidateTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// ValidationResult represents the token validation result
type ValidationResult struct {
	Valid        bool      `json:"valid"`
	SubscriberID string    `json:"subscriberId,omitempty"`
	Type         string    `json:"type,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// CreateToken creates a new token for a subscriber
// POST /api/admin/tokens/create
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Generate token
	tokenBytes := make([]byte, 64)
	_, _ = rand.Read(tokenBytes) // crypto/rand.Read always succeeds on supported platforms
	tokenString := base64.URLEncoding.EncodeToString(tokenBytes)

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(time.Duration(req.TTL) * time.Second)

	// Create token in database
	token := &tokens.Token{
		TokenID:   "tok_" + tokenString[:16],
		TenantID:  tenantID,
		TokenType: req.TokenType,
		Subject:   req.SubscriberID,
		Scope:     req.Scopes,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		UsageCount: 0,
	}

	err := h.repo.CreateToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		Type:      req.TokenType,
	})
}

// ListTokens returns a paginated list of tokens
// GET /api/admin/tokens
func (h *TokenHandler) ListTokens(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Parse query parameters
	subscriberID := c.Query("subscriberId")
	tokenType := c.Query("type")
	status := c.Query("status")

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filters := tokens.TokenFilters{
		TenantID:     tenantID,
		SubscriberID: subscriberID,
		TokenType:    tokenType,
		Status:       status,
		Limit:        limit,
		Offset:       offset,
	}

	dbTokens, total, err := h.repo.ListTokens(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tokens"})
		return
	}

	// Convert to response format
	responseTokens := make([]Token, len(dbTokens))
	for i, dbToken := range dbTokens {
		statusStr := "active"
		if dbToken.RevokedAt != nil {
			statusStr = "revoked"
		} else if dbToken.ExpiresAt.Before(time.Now()) {
			statusStr = "expired"
		}

		lastUsed := time.Time{}
		if dbToken.LastUsedAt != nil {
			lastUsed = *dbToken.LastUsedAt
		}

		responseTokens[i] = Token{
			ID:             dbToken.TokenID,
			SubscriberID:   dbToken.Subject,
			SubscriberName: dbToken.SubscriberName,
			Type:           dbToken.TokenType,
			Status:         statusStr,
			CreatedAt:      dbToken.IssuedAt,
			ExpiresAt:      dbToken.ExpiresAt,
			LastUsed:       lastUsed,
			UsageCount:     dbToken.UsageCount,
		}
	}

	c.JSON(http.StatusOK, TokenListResponse{
		Tokens: responseTokens,
		Total:  total,
	})
}

// GetToken retrieves details of a specific token
// GET /api/admin/tokens/:id
func (h *TokenHandler) GetToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	tokenID := c.Param("id")

	dbToken, err := h.repo.GetToken(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve token"})
		return
	}

	statusStr := "active"
	if dbToken.RevokedAt != nil {
		statusStr = "revoked"
	} else if dbToken.ExpiresAt.Before(time.Now()) {
		statusStr = "expired"
	}

	lastUsed := time.Time{}
	if dbToken.LastUsedAt != nil {
		lastUsed = *dbToken.LastUsedAt
	}

	token := Token{
		ID:             dbToken.TokenID,
		SubscriberID:   dbToken.Subject,
		SubscriberName: dbToken.SubscriberName,
		Type:           dbToken.TokenType,
		Status:         statusStr,
		CreatedAt:      dbToken.IssuedAt,
		ExpiresAt:      dbToken.ExpiresAt,
		LastUsed:       lastUsed,
		UsageCount:     dbToken.UsageCount,
	}

	c.JSON(http.StatusOK, token)
}

// ValidateToken validates a token and returns its details
// POST /api/admin/tokens/validate
func (h *TokenHandler) ValidateToken(c *gin.Context) {
	var req ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Extract token ID from token string
	tokenID := ""
	if len(req.Token) > 4 && req.Token[:4] == "tok_" {
		if len(req.Token) > 20 {
			tokenID = req.Token[:20]
		} else {
			tokenID = req.Token
		}
	} else {
		c.JSON(http.StatusOK, ValidationResult{
			Valid: false,
			Error: "Invalid token format",
		})
		return
	}

	// Check if token exists
	dbToken, err := h.repo.GetToken(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, ValidationResult{
				Valid: false,
				Error: "Token not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate token"})
		return
	}

	// Check if revoked
	if dbToken.RevokedAt != nil {
		c.JSON(http.StatusOK, ValidationResult{
			Valid: false,
			Error: "Token has been revoked",
		})
		return
	}

	// Check if expired
	if dbToken.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusOK, ValidationResult{
			Valid: false,
			Error: "Token has expired",
		})
		return
	}

	// Check blacklist
	blacklisted, err := h.repo.IsBlacklisted(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check blacklist"})
		return
	}
	if blacklisted {
		c.JSON(http.StatusOK, ValidationResult{
			Valid: false,
			Error: "Token is blacklisted",
		})
		return
	}

	// Update last used
	err = h.repo.UpdateLastUsed(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		fmt.Printf("Failed to update last used: %v\n", err)
	}

	c.JSON(http.StatusOK, ValidationResult{
		Valid:        true,
		SubscriberID: dbToken.Subject,
		Type:         dbToken.TokenType,
		ExpiresAt:    dbToken.ExpiresAt,
	})
}

// RevokeToken revokes a specific token
// POST /api/admin/tokens/:id/revoke
func (h *TokenHandler) RevokeToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	tokenID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // Optional request body; use defaults if malformed

	revokedBy := c.GetString("user_id")
	if revokedBy == "" {
		revokedBy = "admin"
	}

	reason := req.Reason
	if reason == "" {
		reason = "Revoked by administrator"
	}

	// Revoke in database
	err := h.repo.RevokeToken(c.Request.Context(), tenantID, tokenID, revokedBy, reason)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	// Get token details for blacklist
	dbToken, err := h.repo.GetToken(c.Request.Context(), tenantID, tokenID)
	if err == nil {
		// Add to blacklist
		revokedAtTime := time.Now()
		blacklistEntry := &tokens.BlacklistEntry{
			TokenID:   tokenID,
			TenantID:  tenantID,
			Reason:    &reason,
			RevokedAt: revokedAtTime,
			RevokedBy: &revokedBy,
			ExpiresAt: dbToken.ExpiresAt,
		}
		err = h.repo.AddToBlacklist(c.Request.Context(), blacklistEntry)
		if err != nil {
			fmt.Printf("Failed to add to blacklist: %v\n", err)
		}

		// Add to Redis for fast lookup
		ttl := time.Until(dbToken.ExpiresAt)
		if ttl > 0 && ttl < 24*time.Hour {
			key := fmt.Sprintf("blacklist:%s:%s", tenantID, tokenID)
			_ = h.redis.Set(c.Request.Context(), key, reason, ttl) // Best effort cache; core revocation in DB
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token revoked successfully",
		"tokenId": tokenID,
	})
}

// RefreshToken refreshes an existing token (only for refresh tokens)
// POST /api/admin/tokens/:id/refresh
func (h *TokenHandler) RefreshToken(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	tokenID := c.Param("id")

	// Get existing token
	oldToken, err := h.repo.GetToken(c.Request.Context(), tenantID, tokenID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve token"})
		return
	}

	// Check if it's a refresh token
	if oldToken.TokenType != "refresh" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only refresh tokens can be refreshed"})
		return
	}

	// Check if expired or revoked
	if oldToken.RevokedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token has been revoked"})
		return
	}
	if oldToken.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token has expired"})
		return
	}

	// Generate new access token
	tokenBytes := make([]byte, 64)
	_, _ = rand.Read(tokenBytes) // crypto/rand.Read always succeeds on supported platforms
	tokenString := base64.URLEncoding.EncodeToString(tokenBytes)

	newToken := &tokens.Token{
		TokenID:    "tok_" + tokenString[:16],
		TenantID:   tenantID,
		TokenType:  "access",
		Subject:    oldToken.Subject,
		Audience:   oldToken.Audience,
		Issuer:     oldToken.Issuer,
		Scope:      oldToken.Scope,
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		UsageCount: 0,
	}

	err = h.repo.CreateToken(c.Request.Context(), newToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new token"})
		return
	}

	// Update old token usage
	_ = h.repo.UpdateLastUsed(c.Request.Context(), tenantID, tokenID) // Best effort tracking

	c.JSON(http.StatusOK, TokenResponse{
		Token:     tokenString,
		ExpiresAt: newToken.ExpiresAt.Format(time.RFC3339),
		Type:      "access",
	})
}

// GetTokenMetrics returns aggregated token metrics
// GET /api/admin/tokens/metrics
func (h *TokenHandler) GetTokenMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	metrics, err := h.repo.GetTokenMetrics(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// SearchTokens searches for tokens based on criteria
// GET /api/admin/tokens/search
func (h *TokenHandler) SearchTokens(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	subscriberID := c.Query("subscriberId")
	status := c.Query("status")
	tokenType := c.Query("type")
	subject := c.Query("subject")

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	filters := tokens.TokenFilters{
		TenantID:     tenantID,
		SubscriberID: subscriberID,
		TokenType:    tokenType,
		Status:       status,
		Subject:      subject,
		Limit:        limit,
		Offset:       offset,
	}

	dbTokens, total, err := h.repo.ListTokens(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search tokens"})
		return
	}

	responseTokens := make([]Token, len(dbTokens))
	for i, dbToken := range dbTokens {
		statusStr := "active"
		if dbToken.RevokedAt != nil {
			statusStr = "revoked"
		} else if dbToken.ExpiresAt.Before(time.Now()) {
			statusStr = "expired"
		}

		lastUsed := time.Time{}
		if dbToken.LastUsedAt != nil {
			lastUsed = *dbToken.LastUsedAt
		}

		responseTokens[i] = Token{
			ID:             dbToken.TokenID,
			SubscriberID:   dbToken.Subject,
			SubscriberName: dbToken.SubscriberName,
			Type:           dbToken.TokenType,
			Status:         statusStr,
			CreatedAt:      dbToken.IssuedAt,
			ExpiresAt:      dbToken.ExpiresAt,
			LastUsed:       lastUsed,
			UsageCount:     dbToken.UsageCount,
		}
	}

	c.JSON(http.StatusOK, TokenListResponse{
		Tokens: responseTokens,
		Total:  total,
	})
}

// RegisterRoutes registers all token management routes
func (h *TokenHandler) RegisterRoutes(router *gin.RouterGroup) {
	tokens := router.Group("/tokens")
	{
		tokens.POST("/create", h.CreateToken)
		tokens.GET("", h.ListTokens)
		tokens.GET("/:id", h.GetToken)
		tokens.POST("/validate", h.ValidateToken)
		tokens.POST("/:id/revoke", h.RevokeToken)
		tokens.POST("/:id/refresh", h.RefreshToken)
		tokens.GET("/metrics", h.GetTokenMetrics)
		tokens.GET("/search", h.SearchTokens)
	}
}
