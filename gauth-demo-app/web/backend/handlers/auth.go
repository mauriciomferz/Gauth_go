package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// AuthHandler handles authentication related endpoints
type AuthHandler struct {
	service *services.GAuthService
	logger  *logrus.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(service *services.GAuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		logger:  logger,
	}
}

// Authorize handles authorization requests
func (h *AuthHandler) Authorize(c *gin.Context) {
	var req services.AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Authorize(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Authorization failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Token handles token requests
func (h *AuthHandler) Token(c *gin.Context) {
	var req services.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Token(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Token exchange failed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token exchange failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Revoke handles token revocation
func (h *AuthHandler) Revoke(c *gin.Context) {
	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token required"})
		return
	}

	// For demo purposes, just log the revocation
	h.logger.WithField("token", token).Info("Token revoked")
	c.JSON(http.StatusOK, gin.H{"message": "Token revoked successfully"})
}

// UserInfo handles user info requests
func (h *AuthHandler) UserInfo(c *gin.Context) {
	// For demo purposes, return mock user info
	userInfo := gin.H{
		"sub":                "demo_user",
		"name":               "Demo User",
		"email":              "demo@example.com",
		"email_verified":     true,
		"preferred_username": "demouser",
	}

	c.JSON(http.StatusOK, userInfo)
}

// Validate handles token validation
func (h *AuthHandler) Validate(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	// For demo purposes, accept any bearer token
	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"user_id":  "demo_user",
		"client_id": "demo_client",
		"scope":    "read write",
	})
}

// LegalFrameworkHandler handles legal framework related endpoints
type LegalFrameworkHandler struct {
	service *services.GAuthService
	logger  *logrus.Logger
}

// NewLegalFrameworkHandler creates a new legal framework handler
func NewLegalFrameworkHandler(service *services.GAuthService, logger *logrus.Logger) *LegalFrameworkHandler {
	return &LegalFrameworkHandler{
		service: service,
		logger:  logger,
	}
}

// CreateEntity handles entity creation
func (h *LegalFrameworkHandler) CreateEntity(c *gin.Context) {
	var entity services.LegalEntity
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateLegalEntity(c.Request.Context(), &entity)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create entity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create entity"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetEntity handles entity retrieval
func (h *LegalFrameworkHandler) GetEntity(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity ID required"})
		return
	}

	entity, err := h.service.GetLegalEntity(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get entity")
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	c.JSON(http.StatusOK, entity)
}

// VerifyLegalCapacity handles legal capacity verification
func (h *LegalFrameworkHandler) VerifyLegalCapacity(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity ID required"})
		return
	}

	// For demo purposes, always return verified
	result := gin.H{
		"entity_id": id,
		"verified":  true,
		"capacity":  "full",
		"timestamp": "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusOK, result)
}

// CreatePowerOfAttorney handles power of attorney creation
func (h *LegalFrameworkHandler) CreatePowerOfAttorney(c *gin.Context) {
	var poa services.PowerOfAttorney
	if err := c.ShouldBindJSON(&poa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreatePowerOfAttorney(c.Request.Context(), &poa)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create power of attorney")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create power of attorney"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetPowerOfAttorney handles power of attorney retrieval
func (h *LegalFrameworkHandler) GetPowerOfAttorney(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Power of Attorney ID required"})
		return
	}

	// For demo purposes, return mock data
	poa := gin.H{
		"id":         id,
		"grantor":    "demo_grantor",
		"grantee":    "demo_grantee",
		"powers":     []string{"sign_contracts", "manage_finances"},
		"status":     "active",
		"created_at": "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusOK, poa)
}

// DelegatePower handles power delegation
func (h *LegalFrameworkHandler) DelegatePower(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Power of Attorney ID required"})
		return
	}

	var delegationReq struct {
		Delegate string   `json:"delegate" binding:"required"`
		Powers   []string `json:"powers" binding:"required"`
	}

	if err := c.ShouldBindJSON(&delegationReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo purposes, return success
	result := gin.H{
		"delegation_id": "delegation_" + id,
		"delegate":      delegationReq.Delegate,
		"powers":        delegationReq.Powers,
		"status":        "active",
		"created_at":    "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusCreated, result)
}

// CreateRequest handles legal framework request creation
func (h *LegalFrameworkHandler) CreateRequest(c *gin.Context) {
	var req struct {
		Type        string                 `json:"type" binding:"required"`
		Description string                 `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo purposes, create a mock request
	result := gin.H{
		"id":          "request_" + strconv.FormatInt(12345, 10),
		"type":        req.Type,
		"description": req.Description,
		"metadata":    req.Metadata,
		"status":      "pending",
		"created_at":  "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusCreated, result)
}

// GetRequest handles legal framework request retrieval
func (h *LegalFrameworkHandler) GetRequest(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request ID required"})
		return
	}

	// For demo purposes, return mock data
	request := gin.H{
		"id":          id,
		"type":        "contract_approval",
		"description": "Request to approve contract terms",
		"status":      "pending",
		"created_at":  "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusOK, request)
}

// ApproveRequest handles legal framework request approval
func (h *LegalFrameworkHandler) ApproveRequest(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request ID required"})
		return
	}

	var approvalReq struct {
		Decision string `json:"decision" binding:"required"`
		Comments string `json:"comments"`
	}

	if err := c.ShouldBindJSON(&approvalReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo purposes, return success
	result := gin.H{
		"request_id": id,
		"decision":   approvalReq.Decision,
		"comments":   approvalReq.Comments,
		"status":     "approved",
		"approved_at": "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusOK, result)
}

// GetJurisdictions handles jurisdiction listing
func (h *LegalFrameworkHandler) GetJurisdictions(c *gin.Context) {
	jurisdictions := []gin.H{
		{"id": "US", "name": "United States", "type": "federal"},
		{"id": "EU", "name": "European Union", "type": "supranational"},
		{"id": "UK", "name": "United Kingdom", "type": "national"},
		{"id": "CA", "name": "Canada", "type": "federal"},
	}

	c.JSON(http.StatusOK, gin.H{"jurisdictions": jurisdictions})
}

// GetJurisdictionRules handles jurisdiction rules retrieval
func (h *LegalFrameworkHandler) GetJurisdictionRules(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jurisdiction ID required"})
		return
	}

	// For demo purposes, return mock rules
	rules := gin.H{
		"jurisdiction": id,
		"rules": []gin.H{
			{"id": "rule_1", "name": "Contract Signing Authority", "description": "Rules for contract signing authority"},
			{"id": "rule_2", "name": "Financial Transaction Limits", "description": "Limits for financial transactions"},
			{"id": "rule_3", "name": "Data Protection Compliance", "description": "Requirements for data protection"},
		},
	}

	c.JSON(http.StatusOK, rules)
}