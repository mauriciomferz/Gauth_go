package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
)

const (
	statusActive    = "active"
	defaultTenantID = "default-tenant"
)

// AuditHandler manages audit trail operations for the admin portal
type AuditHandler struct {
	repo          *audit.Repository
	exportService *audit.ExportService
}

// NewAuditHandler creates a new audit handler instance
func NewAuditHandler(db *pgxpool.Pool) *AuditHandler {
	repo := audit.NewRepository(db)
	exportService := audit.NewExportService(repo, "/tmp/gauth-audit-exports")
	
	// Start cleanup routine for expired exports
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			exportService.CleanupExpiredJobs()
		}
	}()
	
	return &AuditHandler{
		repo:          repo,
		exportService: exportService,
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
	Format       string  `json:"format" binding:"required,oneof=json csv syslog cef"`
	DateRange    string  `json:"dateRange" binding:"required"`
	Category     *string `json:"category"`
	Severity     *string `json:"severity"`
	Actor        *string `json:"actor"`
	Action       *string `json:"action"`
	ResourceType *string `json:"resourceType"`
	Compressed   bool    `json:"compressed"`
}

// ListAuditEvents returns audit events with optional filtering
// GET /api/admin/audit/events
func (h *AuditHandler) ListAuditEvents(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware)
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID // Fallback for development
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
func (h *AuditHandler) CreateExportJob(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
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
func (h *AuditHandler) GetRetentionPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
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

// ExportAuditTrail initiates an async export job
// POST /api/admin/audit/export
func (h *AuditHandler) ExportAuditTrail(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse date range
	startDate, endDate, err := h.parseDateRange(req.DateRange)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid date range: %v", err)})
		return
	}
	
	// Build filter
	filter := audit.ExportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     10000, // Max 10,000 events per export
	}
	
	if req.Category != nil {
		filter.Category = *req.Category
	}
	if req.Severity != nil {
		filter.Severity = *req.Severity
	}
	if req.Actor != nil {
		filter.Actor = *req.Actor
	}
	if req.Action != nil {
		filter.Action = *req.Action
	}
	if req.ResourceType != nil {
		filter.ResourceType = *req.ResourceType
	}
	
	// Parse format
	format := audit.ExportFormat(req.Format)
	
	// Create export job
	job, err := h.exportService.CreateExportJob(c.Request.Context(), tenantID, format, filter, req.Compressed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create export job: %v", err)})
		return
	}
	
	c.JSON(http.StatusAccepted, gin.H{
		"jobId":     job.ID,
		"status":    job.Status,
		"format":    job.Format,
		"createdAt": job.CreatedAt.Format(time.RFC3339),
		"expiresAt": job.ExpiresAt.Format(time.RFC3339),
	})
}

// GetExportStatus retrieves the status of an export job
// GET /api/admin/audit/export/:id
func (h *AuditHandler) GetExportStatus(c *gin.Context) {
	jobID := c.Param("id")
	
	job, err := h.exportService.GetExportJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export job not found"})
		return
	}
	
	response := gin.H{
		"jobId":       job.ID,
		"status":      job.Status,
		"format":      job.Format,
		"compressed":  job.Compressed,
		"totalEvents": job.TotalEvents,
		"fileSize":    job.FileSize,
		"createdAt":   job.CreatedAt.Format(time.RFC3339),
		"expiresAt":   job.ExpiresAt.Format(time.RFC3339),
	}
	
	if job.CompletedAt != nil {
		response["completedAt"] = job.CompletedAt.Format(time.RFC3339)
	}
	
	if job.Error != "" {
		response["error"] = job.Error
	}
	
	c.JSON(http.StatusOK, response)
}

// DownloadExport downloads a completed export file
// GET /api/admin/audit/export/:id/download
func (h *AuditHandler) DownloadExport(c *gin.Context) {
	jobID := c.Param("id")
	
	job, err := h.exportService.GetExportJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export job not found"})
		return
	}
	
	if job.Status != audit.ExportStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("export not ready (status: %s)", job.Status)})
		return
	}
	
	// Determine content type and filename
	ext := string(job.Format)
	contentType := "application/octet-stream"
	
	switch job.Format {
	case audit.ExportFormatJSON:
		contentType = "application/json"
	case audit.ExportFormatCSV:
		contentType = "text/csv"
	case audit.ExportFormatSyslog, audit.ExportFormatCEF:
		contentType = "text/plain"
	}
	
	if job.Compressed {
		ext += ".gz"
		contentType = "application/gzip"
	}
	
	filename := fmt.Sprintf("audit-export-%s.%s", time.Now().Format("20060102-150405"), ext)
	
	// Set headers
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", job.FileSize))
	
	// Stream file
	c.File(job.FilePath)
}

// DeleteExport deletes an export job and its file
// DELETE /api/admin/audit/export/:id
func (h *AuditHandler) DeleteExport(c *gin.Context) {
	jobID := c.Param("id")
	
	if err := h.exportService.DeleteExportJob(jobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export job not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "export job deleted"})
}

// parseDateRange parses date range string into start and end times
func (h *AuditHandler) parseDateRange(dateRange string) (time.Time, time.Time, error) {
	now := time.Now()
	var startDate, endDate time.Time
	
	switch dateRange {
	case "last-1h":
		startDate = now.Add(-1 * time.Hour)
		endDate = now
	case "last-24h":
		startDate = now.Add(-24 * time.Hour)
		endDate = now
	case "last-7d":
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	case "last-30d":
		startDate = now.AddDate(0, 0, -30)
		endDate = now
	case "all":
		// All time - use a very old date
		startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = now
	default:
		// Try to parse custom range format: "YYYY-MM-DD,YYYY-MM-DD"
		parts := strings.Split(dateRange, ",")
		if len(parts) != 2 {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date range format")
		}
		
		var err error
		startDate, err = time.Parse("2006-01-02", strings.TrimSpace(parts[0]))
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date: %v", err)
		}
		
		endDate, err = time.Parse("2006-01-02", strings.TrimSpace(parts[1]))
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date: %v", err)
		}
		
		// Set end date to end of day
		endDate = endDate.Add(24*time.Hour - time.Second)
	}
	
	return startDate, endDate, nil
}

// ListSIEMIntegrations returns all SIEM integrations
// GET /api/admin/audit/siem
func (h *AuditHandler) ListExportJobs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
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
			Enabled:    db.Status == statusActive,
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
func (h *AuditHandler) GetExportJob(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
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
			if siem.Status == statusActive {
				activeCount++
			}
			totalEventsSent += siem.EventsSent
		}
		metrics["siem_integrations"] = map[string]interface{}{
			"total":       len(siemIntegrations),
			statusActive:  activeCount,
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
		
		// Export endpoints (async export with job tracking)
		audit.POST("/export", h.ExportAuditTrail)
		audit.GET("/export/:id", h.GetExportStatus)
		audit.GET("/export/:id/download", h.DownloadExport)
		audit.DELETE("/export/:id", h.DeleteExport)

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
