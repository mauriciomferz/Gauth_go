package admin

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
)

// GAuthPlusHandler handles GAuth+ enhanced authorization features
type GAuthPlusHandler struct {
	successorService   gauthplus.SuccessorManagementService
	delegationService  gauthplus.DelegationService
	dualControlService gauthplus.DualControlService
	fiduciaryService   gauthplus.FiduciaryDutyService
	capabilityService  gauthplus.CapabilityAssessmentService
}

// NewGAuthPlusHandler creates a new GAuth+ handler
func NewGAuthPlusHandler(pool *pgxpool.Pool) *GAuthPlusHandler {
	// Convert pgxpool to database/sql for service compatibility
	var db *sql.DB
	if pool != nil {
		db = stdlib.OpenDBFromPool(pool)
	}

	return &GAuthPlusHandler{
		successorService:   gauthplus.NewPostgreSQLSuccessorService(db),
		delegationService:  gauthplus.NewPostgreSQLDelegationService(db),
		dualControlService: gauthplus.NewPostgreSQLDualControlService(db),
		fiduciaryService:   gauthplus.NewPostgreSQLFiduciaryDutyService(db),
		capabilityService:  gauthplus.NewPostgreSQLCapabilityAssessmentService(db),
	}
}

// ====== Successor Management Endpoints ======

// ActivateSuccessor activates successor AI when primary agent fails
// POST /api/admin/gauthplus/successor/:id/activate
func (h *GAuthPlusHandler) ActivateSuccessor(c *gin.Context) {
	poaID := c.Param("id")

	var req struct {
		PrimaryAgentID   string `json:"primary_agent_id" binding:"required"`
		SuccessorAgentID string `json:"successor_agent_id" binding:"required"`
		Reason           string `json:"reason" binding:"required,oneof=unavailable failure manual timeout"`
		ActivatedBy      string `json:"activated_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activation, err := h.successorService.ActivateSuccessor(
		c.Request.Context(),
		poaID,
		req.PrimaryAgentID,
		req.SuccessorAgentID,
		req.Reason,
		req.ActivatedBy,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"activation": activation,
	})
}

// DeactivateSuccessor returns control to primary AI
// POST /api/admin/gauthplus/successor/:id/deactivate
func (h *GAuthPlusHandler) DeactivateSuccessor(c *gin.Context) {
	var req struct {
		ActivationID  string `json:"activation_id" binding:"required"`
		DeactivatedBy string `json:"deactivated_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.successorService.DeactivateSuccessor(
		c.Request.Context(),
		req.ActivationID,
		req.DeactivatedBy,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successor deactivated successfully",
	})
}

// GetActiveSuccessor returns currently active successor
// GET /api/admin/gauthplus/successor/:id/active
func (h *GAuthPlusHandler) GetActiveSuccessor(c *gin.Context) {
	poaID := c.Param("id")

	activation, err := h.successorService.GetActiveSuccessor(c.Request.Context(), poaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if activation == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"active":  false,
			"message": "No active successor",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":     true,
		"activation": activation,
	})
}

// ListSuccessorHistory returns activation history
// GET /api/admin/gauthplus/successor/:id/history
func (h *GAuthPlusHandler) ListSuccessorHistory(c *gin.Context) {
	poaID := c.Param("id")

	history, err := h.successorService.ListSuccessorHistory(c.Request.Context(), poaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   len(history),
	})
}

// ====== Delegation Management Endpoints ======

// CreateDelegation creates AI-to-AI delegation
// POST /api/admin/delegations
func (h *GAuthPlusHandler) CreateDelegation(c *gin.Context) {
	var delegation gauthplus.AIDelegation

	if err := c.ShouldBindJSON(&delegation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if delegation.ID == "" {
		delegation.ID = uuid.New().String()
	}
	delegation.CreatedAt = time.Now().UTC()
	delegation.UpdatedAt = time.Now().UTC()
	delegation.Status = statusActive

	if err := h.delegationService.CreateDelegation(c.Request.Context(), &delegation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"delegation": delegation,
	})
}

// ValidateDelegation checks if delegation is allowed
// POST /api/admin/delegations/validate
func (h *GAuthPlusHandler) ValidateDelegation(c *gin.Context) {
	var req struct {
		SourceAgentID string   `json:"source_agent_id" binding:"required"`
		TargetAgentID string   `json:"target_agent_id" binding:"required"`
		Scope         []string `json:"scope" binding:"required"`
		Depth         int      `json:"depth" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.delegationService.ValidateDelegation(
		c.Request.Context(),
		req.SourceAgentID,
		req.TargetAgentID,
		req.Scope,
		req.Depth,
	)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"reason": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// GetDelegationChain returns full delegation chain
// GET /api/admin/delegations/chain/:agentId
func (h *GAuthPlusHandler) GetDelegationChain(c *gin.Context) {
	agentID := c.Param("agentId")

	chain, err := h.delegationService.GetDelegationChain(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"chain":    chain,
		"depth":    len(chain),
	})
}

// RevokeDelegation revokes active delegation
// DELETE /api/admin/delegations/:id
func (h *GAuthPlusHandler) RevokeDelegation(c *gin.Context) {
	delegationID := c.Param("id")

	var req struct {
		RevokedBy string `json:"revoked_by" binding:"required"`
		Reason    string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.delegationService.RevokeDelegation(
		c.Request.Context(),
		delegationID,
		req.RevokedBy,
		req.Reason,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delegation revoked successfully",
	})
}

// ====== Dual Control Approval Endpoints ======

// RequestApproval initiates approval workflow
// POST /api/admin/approvals
func (h *GAuthPlusHandler) RequestApproval(c *gin.Context) {
	var approval gauthplus.DualControlApproval

	if err := c.ShouldBindJSON(&approval); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approvalID, err := h.dualControlService.RequestApproval(c.Request.Context(), &approval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"approval_id": approvalID,
		"status":      "pending",
	})
}

// ApproveAction records approver's approval
// POST /api/admin/approvals/:id/approve
func (h *GAuthPlusHandler) ApproveAction(c *gin.Context) {
	approvalID := c.Param("id")

	var req struct {
		ApproverID string `json:"approver_id" binding:"required"`
		Comments   string `json:"comments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.dualControlService.ApproveAction(
		c.Request.Context(),
		approvalID,
		req.ApproverID,
		req.Comments,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check final status
	status, err := h.dualControlService.CheckApprovalStatus(c.Request.Context(), approvalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  status,
		"message": "Approval recorded",
	})
}

// RejectAction records approver's rejection
// POST /api/admin/approvals/:id/reject
func (h *GAuthPlusHandler) RejectAction(c *gin.Context) {
	approvalID := c.Param("id")

	var req struct {
		ApproverID string `json:"approver_id" binding:"required"`
		Comments   string `json:"comments" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.dualControlService.RejectAction(
		c.Request.Context(),
		approvalID,
		req.ApproverID,
		req.Comments,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "rejected",
		"message": "Rejection recorded",
	})
}

// GetPendingApprovals returns approvals awaiting decision
// GET /api/admin/approvals/pending
func (h *GAuthPlusHandler) GetPendingApprovals(c *gin.Context) {
	approverID := c.Query("approver_id")

	approvals, err := h.dualControlService.GetPendingApprovals(c.Request.Context(), approverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"approvals": approvals,
		"total":     len(approvals),
	})
}

// CheckApprovalStatus checks approval status
// GET /api/admin/approvals/:id/status
func (h *GAuthPlusHandler) CheckApprovalStatus(c *gin.Context) {
	approvalID := c.Param("id")

	status, err := h.dualControlService.CheckApprovalStatus(c.Request.Context(), approvalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"approval_id": approvalID,
		"status":      status,
	})
}

// ====== Fiduciary Duty Endpoints ======

// RecordViolation records fiduciary duty breach
// POST /api/admin/violations
func (h *GAuthPlusHandler) RecordViolation(c *gin.Context) {
	var violation gauthplus.FiduciaryDutyViolation

	if err := c.ShouldBindJSON(&violation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fiduciaryService.RecordViolation(c.Request.Context(), &violation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"violation": violation,
	})
}

// GetViolations returns violations for PoA or agent
// GET /api/admin/violations
func (h *GAuthPlusHandler) GetViolations(c *gin.Context) {
	poaID := c.Query("poa_id")
	agentID := c.Query("agent_id")

	violations, err := h.fiduciaryService.GetViolations(c.Request.Context(), poaID, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"violations": violations,
		"total":      len(violations),
	})
}

// GetViolationsBySeverity returns violations above severity threshold
// GET /api/admin/violations/severity/:level
func (h *GAuthPlusHandler) GetViolationsBySeverity(c *gin.Context) {
	minSeverity := c.Param("level")

	violations, err := h.fiduciaryService.GetViolationsBySeverity(c.Request.Context(), minSeverity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"min_severity": minSeverity,
		"violations":   violations,
		"total":        len(violations),
	})
}

// ResolveViolation marks violation as resolved
// PUT /api/admin/violations/:id/resolve
func (h *GAuthPlusHandler) ResolveViolation(c *gin.Context) {
	violationID := c.Param("id")

	var req struct {
		ReviewedBy string `json:"reviewed_by" binding:"required"`
		Notes      string `json:"notes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fiduciaryService.ResolveViolation(
		c.Request.Context(),
		violationID,
		req.ReviewedBy,
		req.Notes,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Violation resolved successfully",
	})
}

// ====== Capability Assessment Endpoints ======

// CreateAssessment creates new capability assessment
// POST /api/admin/assessments
func (h *GAuthPlusHandler) CreateAssessment(c *gin.Context) {
	var assessment gauthplus.AICapabilityAssessment

	if err := c.ShouldBindJSON(&assessment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.capabilityService.CreateAssessment(c.Request.Context(), &assessment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"assessment": assessment,
	})
}

// GetLatestAssessment returns most recent assessment
// GET /api/admin/assessments/agent/:agentId/latest
func (h *GAuthPlusHandler) GetLatestAssessment(c *gin.Context) {
	agentID := c.Param("agentId")

	assessment, err := h.capabilityService.GetLatestAssessment(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if assessment == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "No assessment found for agent",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assessment": assessment,
	})
}

// CheckCapabilityMatch checks if agent meets requirements
// POST /api/admin/assessments/check-match
func (h *GAuthPlusHandler) CheckCapabilityMatch(c *gin.Context) {
	var req struct {
		AgentID      string                           `json:"agent_id" binding:"required"`
		Requirements gauthplus.CapabilityRequirements `json:"requirements" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	matches, reasons, err := h.capabilityService.CheckCapabilityMatch(
		c.Request.Context(),
		req.AgentID,
		&req.Requirements,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": req.AgentID,
		"matches":  matches,
		"reasons":  reasons,
	})
}

// GetExpiringAssessments returns assessments expiring soon
// GET /api/admin/assessments/expiring?days=30
func (h *GAuthPlusHandler) GetExpiringAssessments(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days := 30
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
		return
	}

	assessments, err := h.capabilityService.GetExpiringAssessments(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"days_until_expiry": days,
		"count":             len(assessments),
		"assessments":       assessments,
	})
}

// RegisterRoutes registers all GAuth+ routes
func (h *GAuthPlusHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Successor management routes
	router.POST("/gauthplus/successor/:id/activate", h.ActivateSuccessor)
	router.POST("/gauthplus/successor/:id/deactivate", h.DeactivateSuccessor)
	router.GET("/gauthplus/successor/:id/active", h.GetActiveSuccessor)
	router.GET("/gauthplus/successor/:id/history", h.ListSuccessorHistory)

	// Delegation routes
	router.POST("/gauthplus/delegations", h.CreateDelegation)
	router.POST("/gauthplus/delegations/validate", h.ValidateDelegation)
	router.GET("/gauthplus/delegations/chain/:agentId", h.GetDelegationChain)
	router.DELETE("/gauthplus/delegations/:id", h.RevokeDelegation)

	// Dual control approval routes
	router.POST("/gauthplus/approvals", h.RequestApproval)
	router.POST("/gauthplus/approvals/:id/approve", h.ApproveAction)
	router.POST("/gauthplus/approvals/:id/reject", h.RejectAction)
	router.GET("/gauthplus/approvals/pending", h.GetPendingApprovals)
	router.GET("/gauthplus/approvals/:id/status", h.CheckApprovalStatus)

	// Fiduciary duty violation routes
	router.POST("/gauthplus/violations", h.RecordViolation)
	router.GET("/gauthplus/violations", h.GetViolations)
	router.GET("/gauthplus/violations/severity/:level", h.GetViolationsBySeverity)
	router.PUT("/gauthplus/violations/:id/resolve", h.ResolveViolation)

	// Capability assessment routes
	router.POST("/gauthplus/assessments", h.CreateAssessment)
	router.GET("/gauthplus/assessments/agent/:agentId/latest", h.GetLatestAssessment)
	router.POST("/gauthplus/assessments/check-match", h.CheckCapabilityMatch)
	router.GET("/gauthplus/assessments/expiring", h.GetExpiringAssessments)
}
