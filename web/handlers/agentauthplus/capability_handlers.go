package agentauthplus

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauthplus"
)

// CapabilityHandlers handles HTTP requests for capability assessment
type CapabilityHandlers struct {
	service agentauthplus.CapabilityAssessmentService
}

// NewCapabilityHandlers creates a new capability handlers instance
func NewCapabilityHandlers(service agentauthplus.CapabilityAssessmentService) *CapabilityHandlers {
	return &CapabilityHandlers{service: service}
}

// CreateAssessmentRequest represents the request to create a capability assessment
type CreateAssessmentRequest struct {
	Assessment *agentauthplus.AICapabilityAssessment `json:"assessment" binding:"required"`
}

// CheckCapabilityMatchRequest represents the request to check capability match
type CheckCapabilityMatchRequest struct {
	AgentID      string                            `json:"agent_id" binding:"required"`
	Requirements *agentauthplus.CapabilityRequirements `json:"requirements" binding:"required"`
}

// GetExpiringAssessmentsRequest represents query parameters for expiring assessments
type GetExpiringAssessmentsRequest struct {
	DaysUntilExpiry int `form:"days_until_expiry" binding:"required,min=1"`
}

// CreateAssessment handles POST /api/v1/agentauthplus/capabilities/assess
func (h *CapabilityHandlers) CreateAssessment(c *gin.Context) {
	var req CreateAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	// Validate required fields in assessment
	if req.Assessment.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "agent_id is required",
		})
		return
	}

	if req.Assessment.OverallLevel == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "overall_level is required",
		})
		return
	}

	if req.Assessment.AssessedBy == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "assessed_by is required",
		})
		return
	}

	err := h.service.CreateAssessment(c.Request.Context(), req.Assessment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "assessment_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"assessment": req.Assessment,
	})
}

// GrantCertification handles POST /api/v1/agentauthplus/capabilities/certify
// Note: Certification management is embedded in capability assessments, not separate service methods
func (h *CapabilityHandlers) GrantCertification(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error":   "not_implemented",
		"detail":  "Certification management is embedded in capability assessments",
	})
}

// RevokeCertification handles POST /api/v1/agentauthplus/capabilities/certifications/:id/revoke
// Note: Certification management is embedded in capability assessments, not separate service methods
func (h *CapabilityHandlers) RevokeCertification(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error":   "not_implemented",
		"detail":  "Certification management is embedded in capability assessments",
	})
}

// GetLatestAssessment handles GET /api/v1/agentauthplus/capabilities/assessments/:agentID
func (h *CapabilityHandlers) GetLatestAssessment(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "agent_id parameter required",
		})
		return
	}

	assessment, err := h.service.GetLatestAssessment(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	if assessment == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "not_found",
			"detail":  "No assessment found for agent",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"agent_id":   agentID,
		"assessment": assessment,
	})
}

// ListCertifications handles GET /api/v1/agentauthplus/capabilities/certifications/:agentID
// Note: Certification information is available through GetLatestAssessment
func (h *CapabilityHandlers) ListCertifications(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "agent_id parameter required",
		})
		return
	}

	// Get latest assessment which includes certifications
	assessment, err := h.service.GetLatestAssessment(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	if assessment == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":        false,
			"error":          "not_found",
			"detail":         "No assessment found for agent",
			"certifications": []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"agent_id":       agentID,
		"certifications": assessment.Certifications,
		"count":          len(assessment.Certifications),
	})
}
