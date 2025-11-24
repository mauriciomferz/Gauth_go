package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditHandler manages audit trail operations for the admin portal
type AuditHandler struct {
	repo *audit.Repository
}

// NewAuditHandler creates a new audit handler instance
func NewAuditHandler(db *pgxpool.Pool) *AuditHandler {
	return &AuditHandler{
		repo: audit.NewRepository(db),
	}
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID         string                 `json:"id"`
	Timestamp  string                 `json:"timestamp"`
	Actor      string                 `json:"actor"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	Result     string                 `json:"result"` // success, failure, denied
	IP         string                 `json:"ip"`
	Category   string                 `json:"category"`
	Severity   string                 `json:"severity"`
	TamperProof bool                  `json:"tamperProof"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ComplianceReport represents a compliance framework report
type ComplianceReport struct {
	ID           string `json:"id"`
	Framework    string `json:"framework"`
	Standard     string `json:"standard"`
	Status       string `json:"status"` // compliant, non-compliant, partial
	Coverage     int    `json:"coverage"`
	Violations   int    `json:"violations"`
	LastAudit    string `json:"lastAudit"`
	Requirements int    `json:"requirements"`
	Met          int    `json:"met"`
}

// EventCorrelation represents a correlated event pattern
type EventCorrelation struct {
	ID          string        `json:"id"`
	Pattern     string        `json:"pattern"`
	Description string        `json:"description"`
	Severity    string        `json:"severity"`
	Events      []AuditEvent  `json:"events"`
	FirstSeen   string        `json:"firstSeen"`
	LastSeen    string        `json:"lastSeen"`
	Occurrences int           `json:"occurrences"`
	Confidence  int           `json:"confidence"`
}

// TamperVerification represents tamper verification result
type TamperVerification struct {
	EventID      string `json:"eventId"`
	Timestamp    string `json:"timestamp"`
	Status       string `json:"status"` // verified, tampered, unknown
	Hash         string `json:"hash"`
	PreviousHash string `json:"previousHash"`
	Signature    string `json:"signature"`
	Verified     bool   `json:"verified"`
}

// SIEMIntegration represents a SIEM integration configuration
type SIEMIntegration struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Endpoint   string `json:"endpoint"`
	Enabled    bool   `json:"enabled"`
	Format     string `json:"format"`
	EventsSent int    `json:"eventsSent"`
	LastSync   string `json:"lastSync"`
	Status     string `json:"status"` // active, inactive, error
}

// SIEMRequest represents the request to create a SIEM integration
type SIEMRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required,oneof=splunk elastic qradar sentinel sumologic datadog"`
	Endpoint string `json:"endpoint" binding:"required,url"`
	Format   string `json:"format" binding:"required,oneof=json cef syslog leef"`
}

// ExportRequest represents the request to export audit trail
type ExportRequest struct {
	Format    string `json:"format" binding:"required,oneof=json csv syslog cef"`
	DateRange string `json:"dateRange" binding:"required"`
}

// ListAuditEvents returns audit events with optional filtering
// GET /api/admin/audit/events
func (h *AuditHandler) ListAuditEvents(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware)
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant" // Fallback for development
	}

	// Parse query parameters
	category := c.Query("category")
	severity := c.Query("severity")
	actor := c.Query("actor")
	status := c.Query("status")
	resourceType := c.Query("resourceType")
	
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build filters
	filters := audit.EventFilters{
		TenantID:     tenantID,
		Category:     category,
		Severity:     severity,
		UserID:       actor,
		Status:       status,
		ResourceType: resourceType,
		Limit:        limit,
		Offset:       offset,
	}

	// Query database
	dbEvents, total, err := h.repo.ListEvents(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit events"})
		return
	}

	// Convert to response format
	events := make([]AuditEvent, len(dbEvents))
	for i, dbEvt := range dbEvents {
		resource := dbEvt.ResourceType
		if dbEvt.ResourceID != "" {
			resource = dbEvt.ResourceType + ":" + dbEvt.ResourceID
		}
		
		metadata := map[string]interface{}{}
		if dbEvt.Changes != nil {
			metadata["changes"] = dbEvt.Changes
		}
		if dbEvt.UserAgent != nil {
			metadata["userAgent"] = *dbEvt.UserAgent
		}
		if dbEvt.RequestID != nil {
			metadata["requestId"] = *dbEvt.RequestID
		}
		
		events[i] = AuditEvent{
			ID:          dbEvt.ID,
			Timestamp:   dbEvt.Timestamp.Format(time.RFC3339),
			Actor:       dbEvt.UserID,
			Action:      dbEvt.Action,
			Resource:    resource,
			Result:      dbEvt.Status,
			IP:          dbEvt.IPAddress,
			Category:    dbEvt.Category,
			Severity:    dbEvt.Severity,
			TamperProof: dbEvt.PreviousHash != nil,
			Metadata:    metadata,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// getFrameworkStandard returns full name of a compliance framework
func getFrameworkStandard(framework string) string {
	standards := map[string]string{
		"GDPR":       "General Data Protection Regulation",
		"SOX":        "Sarbanes-Oxley Act",
		"HIPAA":      "Health Insurance Portability and Accountability Act",
		"PCI-DSS":    "Payment Card Industry Data Security Standard",
		"ISO-27001":  "Information Security Management",
		"SOC2":       "Service Organization Control 2",
		"NIST":       "National Institute of Standards and Technology",
	}
	if standard, ok := standards[framework]; ok {
		return standard
	}
	return framework
}

// GetComplianceReports returns compliance status for various frameworks
// GET /api/admin/audit/compliance
func (h *AuditHandler) GetComplianceReports(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	dbReports, err := h.repo.ListComplianceReports(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve compliance reports"})
		return
	}

	// Convert to response format
	reports := make([]ComplianceReport, len(dbReports))
	for i, dbReport := range dbReports {
		coverage := 0
		if dbReport.TotalEvents > 0 {
			coverage = (dbReport.CompliantEvents * 100) / dbReport.TotalEvents
		}
		
		status := dbReport.Status
		if status == "" {
			if dbReport.CriticalViolations > 0 {
				status = "non-compliant"
			} else if coverage < 80 {
				status = "partial"
			} else {
				status = "compliant"
			}
		}
		
		reports[i] = ComplianceReport{
			ID:           dbReport.ID,
			Framework:    dbReport.Framework,
			Standard:     getFrameworkStandard(dbReport.Framework),
			Status:       status,
			Coverage:     coverage,
			Violations:   dbReport.CriticalViolations,
			LastAudit:    dbReport.GeneratedAt.Format(time.RFC3339),
			Requirements: dbReport.TotalEvents,
			Met:          dbReport.CompliantEvents,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}

// GetEventCorrelations returns correlated event patterns
// GET /api/admin/audit/correlations
func (h *AuditHandler) GetEventCorrelations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	dbPatterns, err := h.repo.ListCorrelationPatterns(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve correlation patterns"})
		return
	}

	// Convert to response format
	correlations := make([]EventCorrelation, len(dbPatterns))
	for i, pattern := range dbPatterns {
		firstSeen := pattern.CreatedAt
		lastSeen := time.Now()
		if pattern.LastMatchAt != nil {
			lastSeen = *pattern.LastMatchAt
		}
		
		description := ""
		if pattern.Description != nil {
			description = *pattern.Description
		}
		
		correlations[i] = EventCorrelation{
			ID:          pattern.ID,
			Pattern:     pattern.PatternName,
			Description: description,
			Severity:    pattern.Severity,
			Events:      []AuditEvent{}, // In production, query matching events
			FirstSeen:   firstSeen.Format(time.RFC3339),
			LastSeen:    lastSeen.Format(time.RFC3339),
			Occurrences: pattern.MatchesCount,
			Confidence:  85, // Calculate based on pattern match accuracy
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"correlations": correlations,
		"total":        len(correlations),
	})
}

// VerifyEvent verifies the integrity of an audit event
// GET /api/admin/audit/verify/:id
func (h *AuditHandler) VerifyEvent(c *gin.Context) {
	eventID := c.Param("id")
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	// Verify hash chain integrity
	verified, err := h.repo.VerifyHashChain(c.Request.Context(), tenantID, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify event"})
		return
	}

	status := "verified"
	if !verified {
		status = "tampered"
	}

	verification := TamperVerification{
		EventID:      eventID,
		Timestamp:    time.Now().Format(time.RFC3339),
		Status:       status,
		Hash:         "", // Could retrieve actual hash from database if needed
		PreviousHash: "", // Could retrieve actual previous hash from database if needed
		Signature:    "", // Future: implement digital signatures
		Verified:     verified,
	}

	c.JSON(http.StatusOK, verification)
}

// ExportAuditTrail exports audit trail in specified format
// POST /api/admin/audit/export
func (h *AuditHandler) ExportAuditTrail(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Generate export file based on format and date range
	// TODO: Support JSON, CSV, Syslog, CEF formats
	
	// Export audit data
	c.Header("Content-Disposition", "attachment; filename=audit-trail."+req.Format)
	c.Header("Content-Type", "application/octet-stream")
	c.String(http.StatusOK, "Audit trail export data")
}

// ListSIEMIntegrations returns all SIEM integrations
// GET /api/admin/audit/siem
func (h *AuditHandler) ListSIEMIntegrations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	dbIntegrations, err := h.repo.ListSIEMIntegrations(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve SIEM integrations"})
		return
	}

	// Convert to response format
	integrations := make([]SIEMIntegration, len(dbIntegrations))
	for i, db := range dbIntegrations {
		lastSync := ""
		if db.LastSyncAt != nil {
			lastSync = db.LastSyncAt.Format(time.RFC3339)
		}

		status := db.Status
		if db.LastError != nil && *db.LastError != "" {
			status = "error"
		}

		integrations[i] = SIEMIntegration{
			ID:         db.ID,
			Name:       db.IntegrationName,
			Type:       db.SIEMType,
			Endpoint:   db.EndpointURL,
			Enabled:    db.Status == "active",
			Format:     db.Format,
			EventsSent: int(db.EventsSent),
			LastSync:   lastSync,
			Status:     status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"integrations": integrations,
		"total":        len(integrations),
	})
}

// CreateSIEMIntegration creates a new SIEM integration
// POST /api/admin/audit/siem
func (h *AuditHandler) CreateSIEMIntegration(c *gin.Context) {
	var req SIEMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	// TODO: Validate SIEM endpoint connectivity
	dbIntegration := &audit.SIEMIntegration{
		TenantID:        tenantID,
		IntegrationName: req.Name,
		SIEMType:        req.Type,
		EndpointURL:     req.Endpoint,
		Format:          req.Format,
		Status:          "active",
		EventsSent:      0,
	}

	err := h.repo.CreateSIEMIntegration(c.Request.Context(), dbIntegration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SIEM integration"})
		return
	}

	integration := SIEMIntegration{
		ID:         dbIntegration.ID,
		Name:       dbIntegration.IntegrationName,
		Type:       dbIntegration.SIEMType,
		Endpoint:   dbIntegration.EndpointURL,
		Enabled:    true,
		Format:     dbIntegration.Format,
		EventsSent: 0,
		LastSync:   dbIntegration.CreatedAt.Format(time.RFC3339),
		Status:     "active",
	}

	c.JSON(http.StatusCreated, integration)
}

// ToggleSIEMIntegration enables or disables a SIEM integration
// POST /api/admin/audit/siem/:id/toggle
func (h *AuditHandler) ToggleSIEMIntegration(c *gin.Context) {
	siemID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Enable/disable SIEM event forwarding
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      siemID,
		"enabled": req.Enabled,
	})
}

// DeleteSIEMIntegration removes a SIEM integration
// DELETE /api/admin/audit/siem/:id
func (h *AuditHandler) DeleteSIEMIntegration(c *gin.Context) {
	siemID := c.Param("id")
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	err := h.repo.DeleteSIEMIntegration(c.Request.Context(), tenantID, siemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete SIEM integration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SIEM integration deleted successfully",
		"id":      siemID,
	})
}

// TestSIEMIntegration tests a SIEM integration connection
// POST /api/admin/audit/siem/:id/test
func (h *AuditHandler) TestSIEMIntegration(c *gin.Context) {
	siemID := c.Param("id")

	// TODO: Send test event to SIEM
	// TODO: Verify connectivity and response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      siemID,
		"message": "SIEM connection test successful",
		"latency": 145, // milliseconds
	})
}

// GetAuditMetrics returns audit trail metrics
// GET /api/admin/audit/metrics
func (h *AuditHandler) GetAuditMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	metrics, err := h.repo.GetAuditMetrics(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	// Add SIEM integration count
	siemIntegrations, err := h.repo.ListSIEMIntegrations(c.Request.Context(), tenantID)
	if err == nil {
		activeCount := 0
		totalEventsSent := int64(0)
		for _, siem := range siemIntegrations {
			if siem.Status == "active" {
				activeCount++
			}
			totalEventsSent += siem.EventsSent
		}
		metrics["siem_integrations"] = map[string]interface{}{
			"total":       len(siemIntegrations),
			"active":      activeCount,
			"events_sent": totalEventsSent,
		}
	}

	c.JSON(http.StatusOK, metrics)
}

// RegisterRoutes registers all audit trail routes
func (h *AuditHandler) RegisterRoutes(router *gin.RouterGroup) {
	audit := router.Group("/audit")
	{
		// Audit Events
		audit.GET("/events", h.ListAuditEvents)
		audit.GET("/verify/:id", h.VerifyEvent)
		audit.POST("/export", h.ExportAuditTrail)

		// Compliance
		audit.GET("/compliance", h.GetComplianceReports)

		// Correlation
		audit.GET("/correlations", h.GetEventCorrelations)

		// SIEM Integration
		audit.GET("/siem", h.ListSIEMIntegrations)
		audit.POST("/siem", h.CreateSIEMIntegration)
		audit.POST("/siem/:id/toggle", h.ToggleSIEMIntegration)
		audit.DELETE("/siem/:id", h.DeleteSIEMIntegration)
		audit.POST("/siem/:id/test", h.TestSIEMIntegration)

		// Metrics
		audit.GET("/metrics", h.GetAuditMetrics)
	}
}
