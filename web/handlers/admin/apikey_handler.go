package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeyHandler handles API key management endpoints
type APIKeyHandler struct {
	db *pgxpool.Pool
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(db *pgxpool.Pool) *APIKeyHandler {
	return &APIKeyHandler{db: db}
}

// APIKey represents an API key
type APIKey struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenantId"`
	KeyName          string    `json:"keyName"`
	KeyPrefix        string    `json:"keyPrefix"`
	Description      string    `json:"description"`
	Scopes           []string  `json:"scopes"`
	Status           string    `json:"status"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	LastUsedAt       *time.Time `json:"lastUsedAt"`
	RevokedAt        *time.Time `json:"revokedAt"`
	RevokedBy        *string    `json:"revokedBy"`
	RevocationReason *string    `json:"revocationReason"`
	TotalRequests    int64     `json:"totalRequests"`
	RateLimitPerMin  int       `json:"rateLimitPerMinute"`
	RateLimitPerHour int       `json:"rateLimitPerHour"`
}

// CreateAPIKeyRequest represents API key creation request
type CreateAPIKeyRequest struct {
	TenantID         string    `json:"tenantId" binding:"required"`
	KeyName          string    `json:"keyName" binding:"required"`
	Description      string    `json:"description"`
	Scopes           []string  `json:"scopes" binding:"required"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	RateLimitPerMin  int       `json:"rateLimitPerMinute"`
	RateLimitPerHour int       `json:"rateLimitPerHour"`
	CreatedBy        string    `json:"createdBy" binding:"required"`
}

// CreateAPIKeyResponse represents API key creation response
type CreateAPIKeyResponse struct {
	APIKey    APIKey `json:"apiKey"`
	SecretKey string `json:"secretKey"` // Only returned once during creation
	Message   string `json:"message"`
}

// generateAPIKey generates a cryptographically secure API key
func generateAPIKey() (string, string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	// Encode to base64
	fullKey := base64.URLEncoding.EncodeToString(bytes)
	
	// Add prefix for identification
	prefixedKey := fmt.Sprintf("gauth_sk_%s", fullKey)
	
	// Extract first 16 chars for display prefix
	keyPrefix := prefixedKey[:16]
	
	return prefixedKey, keyPrefix, nil
}

// hashAPIKey creates a SHA-256 hash of the API key
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// CreateAPIKey creates a new API key
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Generate new API key
	secretKey, keyPrefix, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Hash the key for storage
	keyHash := hashAPIKey(secretKey)

	// Set defaults
	if req.RateLimitPerMin == 0 {
		req.RateLimitPerMin = 60
	}
	if req.RateLimitPerHour == 0 {
		req.RateLimitPerHour = 1000
	}

	// Insert into database
	id := uuid.New()
	query := `
		INSERT INTO api_keys (
			id, tenant_id, key_name, key_prefix, key_hash, 
			description, scopes, status, created_by, expires_at,
			rate_limit_per_minute, rate_limit_per_hour
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`

	var createdAt time.Time
	err = h.db.QueryRow(ctx, query,
		id, req.TenantID, req.KeyName, keyPrefix, keyHash,
		req.Description, req.Scopes, "active", req.CreatedBy, req.ExpiresAt,
		req.RateLimitPerMin, req.RateLimitPerHour,
	).Scan(&createdAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key", "details": err.Error()})
		return
	}

	apiKey := APIKey{
		ID:               id.String(),
		TenantID:         req.TenantID,
		KeyName:          req.KeyName,
		KeyPrefix:        keyPrefix,
		Description:      req.Description,
		Scopes:           req.Scopes,
		Status:           "active",
		CreatedBy:        req.CreatedBy,
		CreatedAt:        createdAt,
		ExpiresAt:        req.ExpiresAt,
		TotalRequests:    0,
		RateLimitPerMin:  req.RateLimitPerMin,
		RateLimitPerHour: req.RateLimitPerHour,
	}

	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKey:    apiKey,
		SecretKey: secretKey,
		Message:   "API key created successfully. Store the secret key securely - it won't be shown again.",
	})
}

// ListAPIKeys returns all API keys for a tenant
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	ctx := c.Request.Context()

	query := `
		SELECT 
			id, tenant_id, key_name, key_prefix, description, 
			scopes, status, created_by, created_at, expires_at,
			last_used_at, revoked_at, revoked_by, revocation_reason,
			total_requests, rate_limit_per_minute, rate_limit_per_hour
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(ctx, query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}
	defer rows.Close()

	apiKeys := []APIKey{}
	for rows.Next() {
		var key APIKey
		err := rows.Scan(
			&key.ID, &key.TenantID, &key.KeyName, &key.KeyPrefix, &key.Description,
			&key.Scopes, &key.Status, &key.CreatedBy, &key.CreatedAt, &key.ExpiresAt,
			&key.LastUsedAt, &key.RevokedAt, &key.RevokedBy, &key.RevocationReason,
			&key.TotalRequests, &key.RateLimitPerMin, &key.RateLimitPerHour,
		)
		if err != nil {
			continue
		}
		apiKeys = append(apiKeys, key)
	}

	c.JSON(http.StatusOK, gin.H{
		"apiKeys": apiKeys,
		"total":   len(apiKeys),
	})
}

// GetAPIKey returns details of a specific API key
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	ctx := c.Request.Context()

	query := `
		SELECT 
			id, tenant_id, key_name, key_prefix, description, 
			scopes, status, created_by, created_at, expires_at,
			last_used_at, revoked_at, revoked_by, revocation_reason,
			total_requests, rate_limit_per_minute, rate_limit_per_hour
		FROM api_keys
		WHERE id = $1 AND tenant_id = $2
	`

	var key APIKey
	err := h.db.QueryRow(ctx, query, keyID, tenantID).Scan(
		&key.ID, &key.TenantID, &key.KeyName, &key.KeyPrefix, &key.Description,
		&key.Scopes, &key.Status, &key.CreatedBy, &key.CreatedAt, &key.ExpiresAt,
		&key.LastUsedAt, &key.RevokedAt, &key.RevokedBy, &key.RevocationReason,
		&key.TotalRequests, &key.RateLimitPerMin, &key.RateLimitPerHour,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API key"})
		return
	}

	c.JSON(http.StatusOK, key)
}

// RevokeAPIKeyRequest represents API key revocation request
type RevokeAPIKeyRequest struct {
	RevokedBy string `json:"revokedBy" binding:"required"`
	Reason    string `json:"reason"`
}

// RevokeAPIKey revokes an API key
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req RevokeAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	query := `
		UPDATE api_keys
		SET 
			status = 'revoked',
			revoked_at = $1,
			revoked_by = $2,
			revocation_reason = $3
		WHERE id = $4 AND tenant_id = $5 AND status = 'active'
		RETURNING id
	`

	var returnedID string
	err := h.db.QueryRow(ctx, query,
		time.Now(), req.RevokedBy, req.Reason, keyID, tenantID,
	).Scan(&returnedID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found or already revoked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key revoked successfully",
		"id":      returnedID,
	})
}

// UpdateAPIKeyRequest represents API key update request
type UpdateAPIKeyRequest struct {
	KeyName          *string    `json:"keyName"`
	Description      *string    `json:"description"`
	Scopes           []string   `json:"scopes"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	RateLimitPerMin  *int       `json:"rateLimitPerMinute"`
	RateLimitPerHour *int       `json:"rateLimitPerHour"`
}

// UpdateAPIKey updates API key metadata
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Build dynamic update query
	query := `UPDATE api_keys SET `
	params := []interface{}{}
	paramIndex := 1

	if req.KeyName != nil {
		query += fmt.Sprintf("key_name = $%d, ", paramIndex)
		params = append(params, *req.KeyName)
		paramIndex++
	}
	if req.Description != nil {
		query += fmt.Sprintf("description = $%d, ", paramIndex)
		params = append(params, *req.Description)
		paramIndex++
	}
	if req.Scopes != nil {
		query += fmt.Sprintf("scopes = $%d, ", paramIndex)
		params = append(params, req.Scopes)
		paramIndex++
	}
	if req.ExpiresAt != nil {
		query += fmt.Sprintf("expires_at = $%d, ", paramIndex)
		params = append(params, *req.ExpiresAt)
		paramIndex++
	}
	if req.RateLimitPerMin != nil {
		query += fmt.Sprintf("rate_limit_per_minute = $%d, ", paramIndex)
		params = append(params, *req.RateLimitPerMin)
		paramIndex++
	}
	if req.RateLimitPerHour != nil {
		query += fmt.Sprintf("rate_limit_per_hour = $%d, ", paramIndex)
		params = append(params, *req.RateLimitPerHour)
		paramIndex++
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d RETURNING id", paramIndex, paramIndex+1)
	params = append(params, keyID, tenantID)

	var returnedID string
	err := h.db.QueryRow(ctx, query, params...).Scan(&returnedID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key updated successfully",
		"id":      returnedID,
	})
}

// GetAPIKeyUsage returns usage statistics for an API key
func (h *APIKeyHandler) GetAPIKeyUsage(c *gin.Context) {
	keyID := c.Param("id")
	tenantID := c.Query("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	ctx := c.Request.Context()

	query := `
		SELECT 
			COUNT(*) as total_requests,
			COUNT(*) FILTER (WHERE status_code >= 200 AND status_code < 300) as successful_requests,
			COUNT(*) FILTER (WHERE status_code >= 400) as failed_requests,
			AVG(response_time_ms) as avg_response_time,
			MAX(timestamp) as last_used
		FROM api_key_usage_logs
		WHERE api_key_id = $1 AND tenant_id = $2
	`

	var stats struct {
		TotalRequests      int64     `json:"totalRequests"`
		SuccessfulRequests int64     `json:"successfulRequests"`
		FailedRequests     int64     `json:"failedRequests"`
		AvgResponseTime    float64   `json:"avgResponseTime"`
		LastUsed           *time.Time `json:"lastUsed"`
	}

	err := h.db.QueryRow(ctx, query, keyID, tenantID).Scan(
		&stats.TotalRequests,
		&stats.SuccessfulRequests,
		&stats.FailedRequests,
		&stats.AvgResponseTime,
		&stats.LastUsed,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve usage statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterRoutes registers API key management routes
func (h *APIKeyHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/api-keys", h.CreateAPIKey)
	router.GET("/api-keys", h.ListAPIKeys)
	router.GET("/api-keys/:id", h.GetAPIKey)
	router.PUT("/api-keys/:id", h.UpdateAPIKey)
	router.POST("/api-keys/:id/revoke", h.RevokeAPIKey)
	router.GET("/api-keys/:id/usage", h.GetAPIKeyUsage)
}
