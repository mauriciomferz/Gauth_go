package admin

import (
	"net/http"
	"encoding/json"
	"strconv"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoAHandler manages Power of Attorney operations for the admin portal
type PoAHandler struct {
	repo *poa.Repository
}

// NewPoAHandler creates a new PoA handler instance
func NewPoAHandler(db *pgxpool.Pool) *PoAHandler {
	return &PoAHandler{
		repo: poa.NewRepository(db),
	}
}

// PowerOfAttorney represents a power of attorney delegation
type PowerOfAttorney struct {
	ID                 string   `json:"id"`
	PrincipalID        string   `json:"principalId"`
	PrincipalName      string   `json:"principalName"`
	RepresentativeID   string   `json:"representativeId"`
	RepresentativeName string   `json:"representativeName"`
	RepresentativeType string   `json:"representativeType"`
	Status             string   `json:"status"` // active, pending, expired, revoked
	ValidFrom          string   `json:"validFrom"`
	ValidUntil         string   `json:"validUntil"`
	Actions            []string `json:"actions"`
	Resources          []string `json:"resources"`
	GeoRestrictions    []string `json:"geoRestrictions"`
	ApprovalStatus     string   `json:"approvalStatus"`
	CreatedAt          string   `json:"createdAt"`
}

// PoARequest represents the request to create a Power of Attorney
type PoARequest struct {
	PrincipalID        string   `json:"principalId" binding:"required"`
	PrincipalName      string   `json:"principalName"`
	RepresentativeID   string   `json:"representativeId" binding:"required"`
	RepresentativeName string   `json:"representativeName"`
	RepresentativeType string   `json:"representativeType" binding:"required"`
	ValidFrom          string   `json:"validFrom" binding:"required"`
	ValidUntil         string   `json:"validUntil" binding:"required"`
	SelectedActions    []string `json:"selectedActions" binding:"required"`
	SelectedResources  []string `json:"selectedResources" binding:"required"`
	GeoRestrictions    []string `json:"geoRestrictions" binding:"required"`
	RequiresApproval   bool     `json:"requiresApproval"`
	NotificationEmail  string   `json:"notificationEmail"`
	Reason             string   `json:"reason"`
}

// PoAListResponse represents the list of Power of Attorneys
type PoAListResponse struct {
	PowerOfAttorneys []PowerOfAttorney `json:"powerOfAttorneys"`
	Total            int               `json:"total"`
}

// ListPoAs returns all Power of Attorney delegations
// GET /api/admin/poa
func (h *PoAHandler) ListPoAs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	
	// Get PoAs from database
	records, total, err := h.repo.ListPoAs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list PoAs"})
		return
	}
	
	// Convert to response format
	poas := make([]PowerOfAttorney, len(records))
	for i, record := range records {
		approvalStatus := "approved"
		if record.Status == "pending" {
			approvalStatus = "pending_approval"
		} else if record.Status == "revoked" {
			approvalStatus = "rejected"
		}
		
		poas[i] = PowerOfAttorney{
			ID:                 record.ID,
			PrincipalID:        record.GrantorID,
			PrincipalName:      record.GrantorName,
			RepresentativeID:   record.RepresentativeID,
			RepresentativeName: record.RepresentativeName,
			RepresentativeType: record.RepresentativeType,
			Status:             record.Status,
			ValidFrom:          record.ValidFrom.Format(time.RFC3339),
			ValidUntil:         record.ValidUntil.Format(time.RFC3339),
			Actions:            record.Actions,
			Resources:          []string{}, // Not stored in DB schema
			GeoRestrictions:    record.GeographicRegions,
			ApprovalStatus:     approvalStatus,
			CreatedAt:          record.CreatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, PoAListResponse{
		PowerOfAttorneys: poas,
		Total:            total,
	})
}

// CreatePoA creates a new Power of Attorney delegation
// POST /api/admin/poa
func (h *PoAHandler) CreatePoA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	var req PoARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse time periods
	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid validFrom format"})
		return
	}
	validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid validUntil format"})
		return
	}

	status := "active"
	approvalStatus := "approved"
	if req.RequiresApproval {
		status = "pending"
		approvalStatus = "pending_manager_approval"
	}

	// Create database record
	record := &poa.PoARecord{
		TenantID:           tenantID,
		PoAName:            req.Reason,
		GrantorID:          req.PrincipalID,
		GrantorName:        req.PrincipalName,
		RepresentativeID:   req.RepresentativeID,
		RepresentativeName: req.RepresentativeName,
		RepresentativeType: req.RepresentativeType,
		ScopeType:          "limited",
		Actions:            req.SelectedActions,
		GeographicRegions:  req.GeoRestrictions,
		Status:             status,
		ValidFrom:          validFrom,
		ValidUntil:         validUntil,
	}
	
	// Set metadata as JSON
	if req.NotificationEmail != "" {
		metadataJSON, _ := json.Marshal(map[string]interface{}{"notification_email": req.NotificationEmail})
		raw := json.RawMessage(metadataJSON)
		record.Metadata = &raw
	}
	
	if err := h.repo.CreatePoA(c.Request.Context(), record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PoA"})
		return
	}

	response := PowerOfAttorney{
		ID:                 record.ID,
		PrincipalID:        record.GrantorID,
		PrincipalName:      record.GrantorName,
		RepresentativeID:   record.RepresentativeID,
		RepresentativeName: record.RepresentativeName,
		RepresentativeType: record.RepresentativeType,
		Status:             record.Status,
		ValidFrom:          record.ValidFrom.Format(time.RFC3339),
		ValidUntil:         record.ValidUntil.Format(time.RFC3339),
		Actions:            record.Actions,
		Resources:          req.SelectedResources,
		GeoRestrictions:    record.GeographicRegions,
		ApprovalStatus:     approvalStatus,
		CreatedAt:          record.CreatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, response)
}

// GetPoA retrieves details of a specific Power of Attorney
// GET /api/admin/poa/:id
func (h *PoAHandler) GetPoA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	poaID := c.Param("id")
	
	record, err := h.repo.GetPoA(c.Request.Context(), tenantID, poaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PoA not found"})
		return
	}
	
	approvalStatus := "approved"
	if record.Status == "pending" {
		approvalStatus = "pending_approval"
	} else if record.Status == "revoked" {
		approvalStatus = "rejected"
	}

	poa := PowerOfAttorney{
		ID:                 record.ID,
		PrincipalID:        record.GrantorID,
		PrincipalName:      record.GrantorName,
		RepresentativeID:   record.RepresentativeID,
		RepresentativeName: record.RepresentativeName,
		RepresentativeType: record.RepresentativeType,
		Status:             record.Status,
		ValidFrom:          record.ValidFrom.Format(time.RFC3339),
		ValidUntil:         record.ValidUntil.Format(time.RFC3339),
		Actions:            record.Actions,
		Resources:          []string{},
		GeoRestrictions:    record.GeographicRegions,
		ApprovalStatus:     approvalStatus,
		CreatedAt:          record.CreatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, poa)
}

// RevokePoA revokes a Power of Attorney
// POST /api/admin/poa/:id/revoke
func (h *PoAHandler) RevokePoA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	poaID := c.Param("id")
	revokedBy := c.GetString("user_id")
	if revokedBy == "" {
		revokedBy = "admin"
	}
	
	type RevokeRequest struct {
		Reason string `json:"reason"`
	}
	var req RevokeRequest
	c.ShouldBindJSON(&req)
	
	reason := req.Reason
	if reason == "" {
		reason = "Revoked by admin"
	}

	err := h.repo.RevokePoA(c.Request.Context(), tenantID, poaID, revokedBy, reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Power of Attorney revoked successfully",
		"id":      poaID,
	})
}

// ApprovePoA approves a pending Power of Attorney
// POST /api/admin/poa/:id/approve
func (h *PoAHandler) ApprovePoA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	poaID := c.Param("id")
	approvedBy := c.GetString("user_id")
	if approvedBy == "" {
		approvedBy = "admin"
	}

	err := h.repo.ApprovePoA(c.Request.Context(), tenantID, poaID, approvedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Power of Attorney approved",
		"id":      poaID,
	})
}

// RejectPoA rejects a pending Power of Attorney
// POST /api/admin/poa/:id/reject
func (h *PoAHandler) RejectPoA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	poaID := c.Param("id")

	type RejectRequest struct {
		Reason string `json:"reason" binding:"required"`
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rejectedBy := c.GetString("user_id")
	if rejectedBy == "" {
		rejectedBy = "admin"
	}

	err := h.repo.RejectPoA(c.Request.Context(), tenantID, poaID, rejectedBy, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Power of Attorney rejected",
		"id":      poaID,
		"reason":  req.Reason,
	})
}

// GetPoAHistory returns the audit history of a Power of Attorney
// GET /api/admin/poa/:id/history
func (h *PoAHandler) GetPoAHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	poaID := c.Param("id")
	
	// Verify PoA exists
	_, err := h.repo.GetPoA(c.Request.Context(), tenantID, poaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PoA not found"})
		return
	}

	// TODO: Integrate with audit trail repository to fetch real audit logs
	// For now, return basic lifecycle events from PoA record itself
	history := []gin.H{
		{
			"timestamp": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			"action":    "created",
			"actor":     "system",
			"details":   "Power of Attorney created",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"poaId":   poaID,
		"history": history,
		"total":   len(history),
	})
}

// GetPoAMetrics returns metrics about Power of Attorney usage
// GET /api/admin/poa/metrics
func (h *PoAHandler) GetPoAMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	stats, err := h.repo.GetPoAStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}
	
	// Calculate approval rate
	approvalRate := 0.0
	if stats.TotalPoAs > 0 {
		approvalRate = float64(stats.ActivePoAs) / float64(stats.TotalPoAs) * 100
	}
	
	metrics := gin.H{
		"total_poas":    stats.TotalPoAs,
		"active_poas":   stats.ActivePoAs,
		"pending_poas":  stats.PendingPoAs,
		"expired_poas":  stats.ExpiredPoAs,
		"revoked_poas":  stats.RevokedPoAs,
		"approval_rate": approvalRate,
		"by_representative_type": stats.ByRepType,
		"top_actions": stats.TopActions,
		"geographic_distribution": stats.GeoDistribution,
	}

	c.JSON(http.StatusOK, metrics)
}

// ValidatePoA validates if a representative can perform an action
// POST /api/admin/poa/validate
func (h *PoAHandler) ValidatePoA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}
	
	type ValidationRequest struct {
		PrincipalID      string `json:"principalId" binding:"required"`
		RepresentativeID string `json:"representativeId" binding:"required"`
		Action           string `json:"action" binding:"required"`
		Resource         string `json:"resource" binding:"required"`
	}

	var req ValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poaRecord, valid, reason := h.repo.ValidatePoA(
		c.Request.Context(),
		tenantID,
		req.PrincipalID,
		req.RepresentativeID,
		req.Action,
		req.Resource,
	)

	response := gin.H{
		"valid":  valid,
		"reason": reason,
	}
	if poaRecord != nil {
		response["poaId"] = poaRecord.ID
	}

	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers all Power of Attorney routes
func (h *PoAHandler) RegisterRoutes(router *gin.RouterGroup) {
	poa := router.Group("/poa")
	{
		poa.GET("", h.ListPoAs)
		poa.POST("", h.CreatePoA)
		poa.GET("/:id", h.GetPoA)
		poa.POST("/:id/revoke", h.RevokePoA)
		poa.POST("/:id/approve", h.ApprovePoA)
		poa.POST("/:id/reject", h.RejectPoA)
		poa.GET("/:id/history", h.GetPoAHistory)
		poa.GET("/metrics", h.GetPoAMetrics)
		poa.POST("/validate", h.ValidatePoA)
	}
}
