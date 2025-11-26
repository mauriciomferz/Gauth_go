package gauthplus

import (
	"net/http"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauthplus"
	"github.com/gin-gonic/gin"
)

// FiduciaryHandlers handles HTTP requests for fiduciary duty management
type FiduciaryHandlers struct {
	service gauthplus.FiduciaryDutyService
}

// NewFiduciaryHandlers creates a new fiduciary handlers instance
func NewFiduciaryHandlers(service gauthplus.FiduciaryDutyService) *FiduciaryHandlers {
	return &FiduciaryHandlers{service: service}
}

// RecordViolationRequest represents the request to record a fiduciary duty violation
type RecordViolationRequest struct {
	Violation *gauthplus.FiduciaryDutyViolation `json:"violation" binding:"required"`
}

// ResolveViolationRequest represents the request to resolve a violation
type ResolveViolationRequest struct {
	ReviewedBy string `json:"reviewed_by" binding:"required"`
	Notes      string `json:"notes"`
}

// RecordViolation handles POST /api/v1/gauthplus/fiduciary/violations
func (h *FiduciaryHandlers) RecordViolation(c *gin.Context) {
	var req RecordViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.RecordViolation(c.Request.Context(), req.Violation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "record_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"violation": req.Violation,
	})
}

// ResolveViolation handles POST /api/v1/gauthplus/fiduciary/violations/:id/resolve
func (h *FiduciaryHandlers) ResolveViolation(c *gin.Context) {
	violationID := c.Param("id")
	if violationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  "violation id parameter required",
		})
		return
	}

	var req ResolveViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"detail":  err.Error(),
		})
		return
	}

	err := h.service.ResolveViolation(c.Request.Context(), violationID, req.ReviewedBy, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "resolution_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Violation resolved successfully",
	})
}

// GetViolations handles GET /api/v1/gauthplus/fiduciary/violations
func (h *FiduciaryHandlers) GetViolations(c *gin.Context) {
	poaID := c.Query("poa_id")
	agentID := c.Query("agent_id")

	violations, err := h.service.GetViolations(c.Request.Context(), poaID, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"violations": violations,
		"count":      len(violations),
	})
}

// GetViolationsBySeverity handles GET /api/v1/gauthplus/fiduciary/violations/by-severity
func (h *FiduciaryHandlers) GetViolationsBySeverity(c *gin.Context) {
	minSeverity := c.Query("min_severity")
	if minSeverity == "" {
		minSeverity = "moderate"
	}

	violations, err := h.service.GetViolationsBySeverity(c.Request.Context(), minSeverity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query_failed",
			"detail":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"min_severity": minSeverity,
		"violations":   violations,
		"count":        len(violations),
	})
}
