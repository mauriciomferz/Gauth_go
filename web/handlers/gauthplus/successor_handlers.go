package gauthplus

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
)

// SuccessorHandlers handles HTTP requests for successor management
type SuccessorHandlers struct {
	service gauthplus.SuccessorManagementService
}

// NewSuccessorHandlers creates a new successor handlers instance
func NewSuccessorHandlers(service gauthplus.SuccessorManagementService) *SuccessorHandlers {
	return &SuccessorHandlers{service: service}
}

// ActivateSuccessorRequest represents the request to activate a successor AI
type ActivateSuccessorRequest struct {
	POAID            string `json:"poa_id" binding:"required"`
	PrimaryAgentID   string `json:"primary_agent_id" binding:"required"`
	SuccessorAgentID string `json:"successor_agent_id" binding:"required"`
	Reason           string `json:"reason" binding:"required"`
	ActivatedBy      string `json:"activated_by" binding:"required"`
}

// DeactivateSuccessorRequest represents the request to deactivate a successor
type DeactivateSuccessorRequest struct {
	ActivationID  string `json:"activation_id" binding:"required"`
	DeactivatedBy string `json:"deactivated_by" binding:"required"`
}

// ActivateSuccessor handles POST /api/v1/gauthplus/successors/activate
func (h *SuccessorHandlers) ActivateSuccessor(c *gin.Context) {
	var req ActivateSuccessorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	activation, err := h.service.ActivateSuccessor(
		c.Request.Context(),
		req.POAID,
		req.PrimaryAgentID,
		req.SuccessorAgentID,
		req.Reason,
		req.ActivatedBy,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "activation_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"activation": activation,
	})
}

// DeactivateSuccessor handles POST /api/v1/gauthplus/successors/deactivate
func (h *SuccessorHandlers) DeactivateSuccessor(c *gin.Context) {
	var req DeactivateSuccessorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.DeactivateSuccessor(
		c.Request.Context(),
		req.ActivationID,
		req.DeactivatedBy,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "deactivation_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successor deactivated successfully",
	})
}

// GetActiveSuccessor handles GET /api/v1/gauthplus/successors/active/:poaID
func (h *SuccessorHandlers) GetActiveSuccessor(c *gin.Context) {
	poaID := c.Param("poaID")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "poa_id parameter required",
		})
		return
	}

	activation, err := h.service.GetActiveSuccessor(c.Request.Context(), poaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	if activation == nil {
		c.JSON(http.StatusOK, gin.H{
			"success":          true,
			"active_successor": nil,
			"message":          "No active successor",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"active_successor": activation,
	})
}

// ListSuccessorHistory handles GET /api/v1/gauthplus/successors/history/:poaID
func (h *SuccessorHandlers) ListSuccessorHistory(c *gin.Context) {
	poaID := c.Param("poaID")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "poa_id parameter required",
		})
		return
	}

	history, err := h.service.ListSuccessorHistory(c.Request.Context(), poaID)
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
		"history": history,
		"count":   len(history),
	})
}
