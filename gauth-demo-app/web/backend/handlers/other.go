package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// AuditHandler handles audit related endpoints
type AuditHandler struct {
	service *services.GAuthService
	logger  *logrus.Logger
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(service *services.GAuthService, logger *logrus.Logger) *AuditHandler {
	return &AuditHandler{
		service: service,
		logger:  logger,
	}
}

// GetEvents handles audit events retrieval
func (h *AuditHandler) GetEvents(c *gin.Context) {
	limit := 10
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	events, err := h.service.GetAuditEvents(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"limit":  limit,
		"offset": offset,
	})
}

// GetEvent handles single audit event retrieval
func (h *AuditHandler) GetEvent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event ID required"})
		return
	}

	// For demo purposes, return mock event
	event := gin.H{
		"id":          id,
		"type":        "authorization_request",
		"actor_id":    "demo_client",
		"resource_id": "demo_user",
		"action":      "authorize",
		"outcome":     "success",
		"timestamp":   "2024-01-01T00:00:00Z",
		"metadata":    gin.H{"scope": "read write"},
	}

	c.JSON(http.StatusOK, event)
}

// GetComplianceReport handles compliance report generation
func (h *AuditHandler) GetComplianceReport(c *gin.Context) {
	report := gin.H{
		"period":     "2024-01",
		"total_events": 1500,
		"successful_authentications": 1450,
		"failed_authentications":     50,
		"compliance_score":          97.5,
		"violations": []gin.H{
			{
				"type":        "rate_limit_exceeded",
				"count":       5,
				"severity":    "medium",
				"description": "Rate limit exceeded on authentication endpoint",
			},
		},
		"generated_at": "2024-01-01T00:00:00Z",
	}

	c.JSON(http.StatusOK, report)
}

// GetAuditTrail handles audit trail retrieval for an entity
func (h *AuditHandler) GetAuditTrail(c *gin.Context) {
	entity := c.Param("entity")
	if entity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity ID required"})
		return
	}

	trail := gin.H{
		"entity_id": entity,
		"events": []gin.H{
			{
				"timestamp":   "2024-01-01T10:00:00Z",
				"action":      "entity_creation",
				"description": "Legal entity created",
				"actor":       "system",
			},
			{
				"timestamp":   "2024-01-01T10:30:00Z",
				"action":      "authorization_request",
				"description": "Authorization requested",
				"actor":       "demo_client",
			},
		},
		"total_events": 2,
	}

	c.JSON(http.StatusOK, trail)
}

// RateHandler handles rate limiting related endpoints
type RateHandler struct {
	service *services.GAuthService
	logger  *logrus.Logger
}

// NewRateHandler creates a new rate handler
func NewRateHandler(service *services.GAuthService, logger *logrus.Logger) *RateHandler {
	return &RateHandler{
		service: service,
		logger:  logger,
	}
}

// GetLimits handles rate limits retrieval
func (h *RateHandler) GetLimits(c *gin.Context) {
	limits := gin.H{
		"global": gin.H{
			"requests_per_minute": 1000,
			"requests_per_hour":   10000,
			"requests_per_day":    100000,
		},
		"per_client": gin.H{
			"requests_per_minute": 60,
			"requests_per_hour":   1000,
			"requests_per_day":    10000,
		},
		"endpoints": gin.H{
			"/api/v1/auth/authorize": gin.H{
				"requests_per_minute": 30,
			},
			"/api/v1/auth/token": gin.H{
				"requests_per_minute": 10,
			},
		},
	}

	c.JSON(http.StatusOK, limits)
}

// SetLimits handles rate limits configuration
func (h *RateHandler) SetLimits(c *gin.Context) {
	var limitsReq struct {
		ClientID          string `json:"client_id"`
		RequestsPerMinute int    `json:"requests_per_minute"`
		RequestsPerHour   int    `json:"requests_per_hour"`
		RequestsPerDay    int    `json:"requests_per_day"`
	}

	if err := c.ShouldBindJSON(&limitsReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo purposes, just return success
	result := gin.H{
		"client_id":           limitsReq.ClientID,
		"requests_per_minute": limitsReq.RequestsPerMinute,
		"requests_per_hour":   limitsReq.RequestsPerHour,
		"requests_per_day":    limitsReq.RequestsPerDay,
		"updated_at":          "2024-01-01T00:00:00Z",
	}

	h.logger.WithFields(logrus.Fields{
		"client_id":           limitsReq.ClientID,
		"requests_per_minute": limitsReq.RequestsPerMinute,
	}).Info("Rate limits updated")

	c.JSON(http.StatusOK, result)
}

// GetStatus handles rate limit status for a client
func (h *RateHandler) GetStatus(c *gin.Context) {
	client := c.Param("client")
	if client == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID required"})
		return
	}

	status := gin.H{
		"client_id": client,
		"current_period": gin.H{
			"requests_made":   45,
			"requests_limit":  60,
			"requests_remaining": 15,
			"reset_time":      "2024-01-01T00:01:00Z",
		},
		"daily_stats": gin.H{
			"requests_made":  1500,
			"requests_limit": 10000,
			"requests_remaining": 8500,
			"reset_time":     "2024-01-02T00:00:00Z",
		},
		"last_request": "2024-01-01T00:00:30Z",
	}

	c.JSON(http.StatusOK, status)
}

// DemoHandler handles demo scenario endpoints
type DemoHandler struct {
	service *services.GAuthService
	logger  *logrus.Logger
}

// NewDemoHandler creates a new demo handler
func NewDemoHandler(service *services.GAuthService, logger *logrus.Logger) *DemoHandler {
	return &DemoHandler{
		service: service,
		logger:  logger,
	}
}

// GetScenarios handles demo scenarios listing
func (h *DemoHandler) GetScenarios(c *gin.Context) {
	scenarios, err := h.service.GetDemoScenarios(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get demo scenarios")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get demo scenarios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"scenarios": scenarios})
}

// RunScenario handles demo scenario execution
func (h *DemoHandler) RunScenario(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scenario ID required"})
		return
	}

	// For demo purposes, simulate scenario execution
	execution := gin.H{
		"scenario_id":   id,
		"execution_id":  "exec_" + id + "_" + strconv.FormatInt(12345, 10),
		"status":        "running",
		"started_at":    "2024-01-01T00:00:00Z",
		"steps_total":   3,
		"steps_completed": 0,
	}

	h.logger.WithField("scenario_id", id).Info("Demo scenario started")

	c.JSON(http.StatusAccepted, execution)
}

// GetScenarioStatus handles demo scenario status retrieval
func (h *DemoHandler) GetScenarioStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scenario ID required"})
		return
	}

	// For demo purposes, return mock status
	status := gin.H{
		"scenario_id":     id,
		"execution_id":    "exec_" + id + "_12345",
		"status":          "completed",
		"started_at":      "2024-01-01T00:00:00Z",
		"completed_at":    "2024-01-01T00:01:00Z",
		"steps_total":     3,
		"steps_completed": 3,
		"steps": []gin.H{
			{
				"id":     "step_1",
				"name":   "Authorization Request",
				"status": "completed",
				"result": gin.H{"code": "auth_code_12345"},
			},
			{
				"id":     "step_2",
				"name":   "Token Exchange",
				"status": "completed",
				"result": gin.H{"access_token": "access_token_12345"},
			},
			{
				"id":     "step_3",
				"name":   "User Info Retrieval",
				"status": "completed",
				"result": gin.H{"user_id": "demo_user"},
			},
		},
	}

	c.JSON(http.StatusOK, status)
}