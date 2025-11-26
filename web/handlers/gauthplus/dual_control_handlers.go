package gauthplus

import (
	"net/http"
	"strings"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauthplus"
	"github.com/gin-gonic/gin"
)

// DualControlHandlers handles HTTP requests for dual control approvals
type DualControlHandlers struct {
	service gauthplus.DualControlService
}

// NewDualControlHandlers creates a new dual control handlers instance
func NewDualControlHandlers(service gauthplus.DualControlService) *DualControlHandlers {
	return &DualControlHandlers{service: service}
}

// RequestApprovalRequest represents the request to create an approval workflow
type RequestApprovalRequest struct {
	Approval *gauthplus.DualControlApproval `json:"approval" binding:"required"`
}

// ApproveActionRequest represents the request to approve an action
type ApproveActionRequest struct {
	ApproverID string `json:"approver_id" binding:"required"`
	Comments   string `json:"comments"`
}

// RejectActionRequest represents the request to reject an action
type RejectActionRequest struct {
	ApproverID string `json:"approver_id" binding:"required"`
	Comments   string `json:"comments"`
}

// RequestApproval handles POST /api/v1/gauthplus/dual-control/approvals
func (h *DualControlHandlers) RequestApproval(c *gin.Context) {
	var req RequestApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	approvalID, err := h.service.RequestApproval(c.Request.Context(), req.Approval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "request_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"approval_id": approvalID,
		"approval":    req.Approval,
	})
}

// ApproveAction handles POST /api/v1/gauthplus/dual-control/approvals/:id/approve
func (h *DualControlHandlers) ApproveAction(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "approval id parameter required",
		})
		return
	}

	var req ApproveActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.ApproveAction(c.Request.Context(), approvalID, req.ApproverID, req.Comments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "approval_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Action approved successfully",
	})
}

// RejectAction handles POST /api/v1/gauthplus/dual-control/approvals/:id/reject
func (h *DualControlHandlers) RejectAction(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "approval id parameter required",
		})
		return
	}

	var req RejectActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.RejectAction(c.Request.Context(), approvalID, req.ApproverID, req.Comments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "rejection_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Action rejected successfully",
	})
}

// GetApprovalStatus handles GET /api/v1/gauthplus/dual-control/approvals/:id/status
func (h *DualControlHandlers) GetApprovalStatus(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "approval id parameter required",
		})
		return
	}

	status, err := h.service.CheckApprovalStatus(c.Request.Context(), approvalID)
	if err != nil {
		// Check if error is "no rows" which means approval doesn't exist
		errMsg := err.Error()
		if errMsg == "sql: no rows in result set" || 
		   errMsg == "approval not found" ||
		   strings.Contains(errMsg, "sql: no rows in result set") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"detail":  "Approval not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"approval_id": approvalID,
		"status":      status,
	})
}

// GetPendingApprovals handles GET /api/v1/gauthplus/dual-control/approvals/pending
func (h *DualControlHandlers) GetPendingApprovals(c *gin.Context) {
	approverID := c.Query("approver_id")
	
	approvals, err := h.service.GetPendingApprovals(c.Request.Context(), approverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"approvals": approvals,
		"count":     len(approvals),
	})
}

// FindApprovalsByPoAAndAction handles GET /api/v1/gauthplus/dual-control/approvals/query
func (h *DualControlHandlers) FindApprovalsByPoAAndAction(c *gin.Context) {
	poaID := c.Query("poa_id")
	actionType := c.Query("action_type")

	if poaID == "" || actionType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "poa_id and action_type query parameters required",
		})
		return
	}

	approvals, err := h.service.FindApprovalsByPoAAndAction(c.Request.Context(), poaID, actionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"poa_id":      poaID,
		"action_type": actionType,
		"approvals":   approvals,
		"count":       len(approvals),
	})
}
