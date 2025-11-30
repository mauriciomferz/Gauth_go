package gauthplus

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
)

// DelegationHandlers handles HTTP requests for delegation management
type DelegationHandlers struct {
	service gauthplus.DelegationService
}

// NewDelegationHandlers creates a new delegation handlers instance
func NewDelegationHandlers(service gauthplus.DelegationService) *DelegationHandlers {
	return &DelegationHandlers{service: service}
}

// CreateDelegationRequest represents the request to create a delegation
type CreateDelegationRequest struct {
	Delegation *gauthplus.AIDelegation `json:"delegation" binding:"required"`
}

// RevokeDelegationRequest represents the request to revoke a delegation
type RevokeDelegationRequest struct {
	RevokedBy string `json:"revoked_by" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

// ValidateDelegationRequest represents the request to validate a delegation
type ValidateDelegationRequest struct {
	SourceAgentID string   `json:"source_agent_id" binding:"required"`
	TargetAgentID string   `json:"target_agent_id" binding:"required"`
	Scope         []string `json:"scope" binding:"required"`
	Depth         int      `json:"depth" binding:"required"`
}

// CheckMaxDepthRequest represents the request to check delegation depth
type CheckMaxDepthRequest struct {
	SourceAgentID string `json:"source_agent_id" binding:"required"`
	CurrentDepth  int    `json:"current_depth" binding:"required"`
}

// CreateDelegation handles POST /api/v1/gauthplus/delegations
func (h *DelegationHandlers) CreateDelegation(c *gin.Context) {
	var req CreateDelegationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.CreateDelegation(c.Request.Context(), req.Delegation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "creation_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"delegation": req.Delegation,
	})
}

// RevokeDelegation handles POST /api/v1/gauthplus/delegations/:id/revoke
func (h *DelegationHandlers) RevokeDelegation(c *gin.Context) {
	delegationID := c.Param("id")
	if delegationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "delegation id parameter required",
		})
		return
	}

	var req RevokeDelegationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.RevokeDelegation(c.Request.Context(), delegationID, req.RevokedBy, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "revocation_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delegation revoked successfully",
	})
}

// ValidateDelegation handles POST /api/v1/gauthplus/delegations/validate
func (h *DelegationHandlers) ValidateDelegation(c *gin.Context) {
	var req ValidateDelegationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.ValidateDelegation(
		c.Request.Context(),
		req.SourceAgentID,
		req.TargetAgentID,
		req.Scope,
		req.Depth,
	)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"valid":   true,
	})
}

// GetDelegationChain handles GET /api/v1/gauthplus/delegations/chain/:agentID
func (h *DelegationHandlers) GetDelegationChain(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "agent_id parameter required",
		})
		return
	}

	chain, err := h.service.GetDelegationChain(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chain":   chain,
		"depth":   len(chain),
	})
}

// CheckMaxDepth handles POST /api/v1/gauthplus/delegations/check-depth
func (h *DelegationHandlers) CheckMaxDepth(c *gin.Context) {
	var req CheckMaxDepthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	exceeded, err := h.service.CheckMaxDepthExceeded(
		c.Request.Context(),
		req.SourceAgentID,
		req.CurrentDepth,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "check_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"depth_exceeded": exceeded,
		"current_depth":  req.CurrentDepth,
	})
}
