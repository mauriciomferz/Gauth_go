package beta

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
)

// CreatePoARequest represents the request to create a Proof of Authorization
type CreatePoARequest struct {
	Grantor      string            `json:"grantor" binding:"required"`
	Grantee      string            `json:"grantee" binding:"required"`
	Scope        []string          `json:"scope" binding:"required"`
	Restrictions map[string]string `json:"restrictions,omitempty"`
	ValidFrom    string            `json:"valid_from" binding:"required"`  // ISO8601
	ValidUntil   string            `json:"valid_until" binding:"required"` // ISO8601
	AgentType    string            `json:"agent_type,omitempty"`
	Sector       string            `json:"sector,omitempty"`
	ActionClass  string            `json:"action_class,omitempty"`
	Jurisdiction string            `json:"jurisdiction,omitempty"`
	Witnesses    []string          `json:"witnesses,omitempty"`
}

// UpdatePoARequest represents the request to update a Proof of Authorization
type UpdatePoARequest struct {
	Scope        []string          `json:"scope,omitempty"`
	Restrictions map[string]string `json:"restrictions,omitempty"`
	ValidUntil   string            `json:"valid_until,omitempty"` // ISO8601
	Status       string            `json:"status,omitempty"`
}

// ValidatePoARequest represents the request to validate a Proof of Authorization
type ValidatePoARequest struct {
	Action    string `json:"action" binding:"required"`
	Context   string `json:"context,omitempty"`
	Timestamp string `json:"timestamp,omitempty"` // ISO8601
}

// PoAResponse represents a Proof of Authorization response
type PoAResponse struct {
	Success bool                               `json:"success"`
	PoA     *agentauth_aap_001.PowerOfAttorney `json:"poa,omitempty"`
	Error   string                             `json:"error,omitempty"`
}

// PoAListResponse represents a list of Proof of Authorization documents
type PoAListResponse struct {
	Success bool                                 `json:"success"`
	PoAs    []*agentauth_aap_001.PowerOfAttorney `json:"poas,omitempty"`
	Total   int                                  `json:"total"`
	Error   string                               `json:"error,omitempty"`
}

// PoAValidationResponse represents the validation result
type PoAValidationResponse struct {
	Success   bool                               `json:"success"`
	Valid     bool                               `json:"valid"`
	PoA       *agentauth_aap_001.PowerOfAttorney `json:"poa,omitempty"`
	Reason    string                             `json:"reason,omitempty"`
	Timestamp time.Time                          `json:"timestamp"`
	Error     string                             `json:"error,omitempty"`
}

// PoAHandler provides HTTP handlers for Proof of Authorization CRUD operations
type PoAHandler struct {
	store map[string]*agentauth_aap_001.PowerOfAttorney // In-memory store for demo
}

// NewPoAHandler creates a new PoA HTTP handler
func NewPoAHandler() *PoAHandler {
	return &PoAHandler{
		store: make(map[string]*agentauth_aap_001.PowerOfAttorney),
	}
}

// HandleCreate creates a new Proof of Authorization
//
// POST /api/v1/beta/poa
//
// Request Body:
//
//	{
//	  "grantor": "entity-123",
//	  "grantee": "person-456",
//	  "scope": ["read", "write", "delete"],
//	  "valid_from": "2025-01-01T00:00:00Z",
//	  "valid_until": "2026-01-01T00:00:00Z",
//	  "jurisdiction": "AT"
//	}
//
// Success Response (201 Created):
//
//	{
//	  "success": true,
//	  "poa": {
//	    "id": "uuid",
//	    "grantor": "entity-123",
//	    "grantee": "person-456",
//	    "scope": ["read", "write", "delete"],
//	    "status": "active",
//	    ...
//	  }
//	}
func (h *PoAHandler) HandleCreate(c *gin.Context) {
	var req CreatePoARequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PoAResponse{
			Success: false,
			Error:   "invalid payload: " + err.Error(),
		})
		return
	}

	// Parse timestamps
	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, PoAResponse{
			Success: false,
			Error:   "invalid valid_from timestamp: " + err.Error(),
		})
		return
	}

	validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
	if err != nil {
		c.JSON(http.StatusBadRequest, PoAResponse{
			Success: false,
			Error:   "invalid valid_until timestamp: " + err.Error(),
		})
		return
	}

	// Create new PoA
	now := time.Now()
	poa := &agentauth_aap_001.PowerOfAttorney{
		ID:           uuid.New().String(),
		Version:      3,
		Grantor:      req.Grantor,
		Grantee:      req.Grantee,
		Scope:        req.Scope,
		Restrictions: req.Restrictions,
		AgentType:    req.AgentType,
		Sector:       req.Sector,
		ActionClass:  req.ActionClass,
		ValidFrom:    validFrom,
		ValidUntil:   validUntil,
		Status:       agentauth_aap_001.POAStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
		Jurisdiction: req.Jurisdiction,
		Witnesses:    req.Witnesses,
	}

	// Store in memory
	h.store[poa.ID] = poa

	c.JSON(http.StatusCreated, PoAResponse{
		Success: true,
		PoA:     poa,
	})
}

// HandleGet retrieves a Proof of Authorization by ID
//
// GET /api/v1/beta/poa/:id
//
// Success Response (200 OK):
//
//	{
//	  "success": true,
//	  "poa": { ... }
//	}
//
// Error Response (404 Not Found):
//
//	{
//	  "success": false,
//	  "error": "PoA not found"
//	}
func (h *PoAHandler) HandleGet(c *gin.Context) {
	poaID := c.Param("id")

	poa, exists := h.store[poaID]
	if !exists {
		c.JSON(http.StatusNotFound, PoAResponse{
			Success: false,
			Error:   "PoA not found",
		})
		return
	}

	c.JSON(http.StatusOK, PoAResponse{
		Success: true,
		PoA:     poa,
	})
}

// HandleList lists all Proof of Authorization documents with optional filters
//
// GET /api/v1/beta/poa?grantor=xxx&grantee=yyy&status=active
//
// Success Response (200 OK):
//
//	{
//	  "success": true,
//	  "poas": [...],
//	  "total": 5
//	}
func (h *PoAHandler) HandleList(c *gin.Context) {
	grantor := c.Query("grantor")
	grantee := c.Query("grantee")
	status := c.Query("status")

	var filtered []*agentauth_aap_001.PowerOfAttorney

	for _, poa := range h.store {
		// Apply filters
		if grantor != "" && poa.Grantor != grantor {
			continue
		}
		if grantee != "" && poa.Grantee != grantee {
			continue
		}
		if status != "" && string(poa.Status) != status {
			continue
		}

		filtered = append(filtered, poa)
	}

	c.JSON(http.StatusOK, PoAListResponse{
		Success: true,
		PoAs:    filtered,
		Total:   len(filtered),
	})
}

// HandleUpdate updates an existing Proof of Authorization
//
// PUT /api/v1/beta/poa/:id
//
// Request Body:
//
//	{
//	  "scope": ["read", "write"],
//	  "valid_until": "2027-01-01T00:00:00Z",
//	  "status": "suspended"
//	}
//
// Success Response (200 OK):
//
//	{
//	  "success": true,
//	  "poa": { ... }
//	}
func (h *PoAHandler) HandleUpdate(c *gin.Context) {
	poaID := c.Param("id")

	poa, exists := h.store[poaID]
	if !exists {
		c.JSON(http.StatusNotFound, PoAResponse{
			Success: false,
			Error:   "PoA not found",
		})
		return
	}

	var req UpdatePoARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PoAResponse{
			Success: false,
			Error:   "invalid payload: " + err.Error(),
		})
		return
	}

	// Update fields
	if req.Scope != nil {
		poa.Scope = req.Scope
	}
	if req.Restrictions != nil {
		poa.Restrictions = req.Restrictions
	}
	if req.ValidUntil != "" {
		validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, PoAResponse{
				Success: false,
				Error:   "invalid valid_until timestamp: " + err.Error(),
			})
			return
		}
		poa.ValidUntil = validUntil
	}
	if req.Status != "" {
		poa.Status = agentauth_aap_001.POAStatus(req.Status)
	}

	poa.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, PoAResponse{
		Success: true,
		PoA:     poa,
	})
}

// HandleDelete revokes/deletes a Proof of Authorization
//
// DELETE /api/v1/beta/poa/:id
//
// Success Response (200 OK):
//
//	{
//	  "success": true,
//	  "poa": { ... }
//	}
func (h *PoAHandler) HandleDelete(c *gin.Context) {
	poaID := c.Param("id")

	poa, exists := h.store[poaID]
	if !exists {
		c.JSON(http.StatusNotFound, PoAResponse{
			Success: false,
			Error:   "PoA not found",
		})
		return
	}

	// Mark as revoked instead of deleting
	now := time.Now()
	poa.Status = agentauth_aap_001.POAStatusRevoked
	poa.RevokedAt = &now
	poa.RevocationReason = "Revoked via API"
	poa.UpdatedAt = now

	c.JSON(http.StatusOK, PoAResponse{
		Success: true,
		PoA:     poa,
	})
}

// HandleValidate validates a Proof of Authorization for a specific action
//
// POST /api/v1/beta/poa/:id/validate
//
// Request Body:
//
//	{
//	  "action": "read",
//	  "context": "resource-789",
//	  "timestamp": "2025-11-15T19:00:00Z"
//	}
//
// Success Response (200 OK):
//
//	{
//	  "success": true,
//	  "valid": true,
//	  "poa": { ... },
//	  "timestamp": "2025-11-15T19:00:00Z"
//	}
//
// Validation Failure (200 OK):
//
//	{
//	  "success": true,
//	  "valid": false,
//	  "reason": "PoA expired",
//	  "timestamp": "2025-11-15T19:00:00Z"
//	}
func (h *PoAHandler) HandleValidate(c *gin.Context) {
	poaID := c.Param("id")

	poa, exists := h.store[poaID]
	if !exists {
		c.JSON(http.StatusNotFound, PoAValidationResponse{
			Success: false,
			Valid:   false,
			Error:   "PoA not found",
		})
		return
	}

	var req ValidatePoARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PoAValidationResponse{
			Success: false,
			Valid:   false,
			Error:   "invalid payload: " + err.Error(),
		})
		return
	}

	// Parse timestamp or use now
	var checkTime time.Time
	if req.Timestamp != "" {
		var err error
		checkTime, err = time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, PoAValidationResponse{
				Success: false,
				Valid:   false,
				Error:   "invalid timestamp: " + err.Error(),
			})
			return
		}
	} else {
		checkTime = time.Now()
	}

	// Validate PoA
	ctx := context.Background()
	valid, reason := h.validatePoA(ctx, poa, req.Action, checkTime)

	c.JSON(http.StatusOK, PoAValidationResponse{
		Success:   true,
		Valid:     valid,
		PoA:       poa,
		Reason:    reason,
		Timestamp: checkTime,
	})
}

// validatePoA performs the actual validation logic
func (h *PoAHandler) validatePoA(ctx context.Context, poa *agentauth_aap_001.PowerOfAttorney, action string, checkTime time.Time) (bool, string) {
	// Check if PoA is active
	if poa.Status != agentauth_aap_001.POAStatusActive {
		return false, "PoA is not active (status: " + string(poa.Status) + ")"
	}

	// Check temporal validity
	if checkTime.Before(poa.ValidFrom) {
		return false, "PoA not yet valid"
	}
	if checkTime.After(poa.ValidUntil) {
		return false, "PoA has expired"
	}

	// Check if action is in scope
	actionInScope := false
	for _, scopeAction := range poa.Scope {
		if scopeAction == action || scopeAction == "*" {
			actionInScope = true
			break
		}
	}

	if !actionInScope {
		return false, "action '" + action + "' not in PoA scope"
	}

	return true, ""
}
