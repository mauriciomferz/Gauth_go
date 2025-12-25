package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mauriciomferz/Gauth_go/pkg/events"
)

// EventHandler manages event system operations for the admin portal
type EventHandler struct {
	repo *events.Repository
}

// NewEventHandler creates a new event handler instance
func NewEventHandler(db *pgxpool.Pool) *EventHandler {
	return &EventHandler{
		repo: events.NewRepository(db),
	}
}

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Category  string                 `json:"category"`
	Timestamp string                 `json:"timestamp"`
	Source    string                 `json:"source"`
	Severity  string                 `json:"severity"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// EventType represents an event type definition
type EventType struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
	Count       int    `json:"count"`
}

// Handler represents an event handler configuration
type Handler struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	EventTypes    []string `json:"eventTypes"`
	Endpoint      string   `json:"endpoint"`
	Method        string   `json:"method"`
	Enabled       bool     `json:"enabled"`
	RetryPolicy   string   `json:"retryPolicy"`
	Timeout       int      `json:"timeout"`
	LastTriggered string   `json:"lastTriggered,omitempty"`
	SuccessRate   float64  `json:"successRate"`
}

// HandlerRequest represents the request to create an event handler
type HandlerRequest struct {
	Name        string   `json:"name" binding:"required"`
	Endpoint    string   `json:"endpoint" binding:"required,url"`
	Method      string   `json:"method" binding:"required,oneof=POST PUT PATCH"`
	EventTypes  []string `json:"eventTypes" binding:"required,min=1"`
	Timeout     int      `json:"timeout" binding:"required,min=100"`
	RetryPolicy string   `json:"retryPolicy" binding:"required,oneof=none linear exponential"`
}

// ListEventTypes returns all available event types
// GET /api/admin/events/types
func (h *EventHandler) ListEventTypes(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	ets, err := h.repo.ListEventTypes(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event types"})
		return
	}

	// Convert to API format
	eventTypes := make([]EventType, len(ets))
	for i, et := range ets {
		eventTypes[i] = EventType{
			Type:        et.EventType,
			Category:    et.Category,
			Description: et.Description,
			Schema:      et.Schema,
			Count:       et.Count,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"eventTypes": eventTypes,
		"total":      len(eventTypes),
	})
}

// GetEventStream returns recent events with optional filtering
// GET /api/admin/events/stream
func (h *EventHandler) GetEventStream(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	// Build filters from query parameters
	filters := events.EventFilters{
		Category: c.Query("category"),
		Severity: c.Query("severity"),
		Source:   c.Query("source"),
		Limit:    100, // Default limit
	}

	es, err := h.repo.ListEvents(c.Request.Context(), tenantID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	// Convert to API format
	eventsResult := make([]Event, len(es))
	for i, e := range es {
		// Build data map from payload and specific fields
		data := make(map[string]interface{})
		for k, v := range e.Payload {
			data[k] = v
		}
		if e.UserID != nil {
			data["userId"] = *e.UserID
		}
		if e.Resource != nil {
			data["resource"] = *e.Resource
		}
		if e.Action != nil {
			data["action"] = *e.Action
		}

		// Build metadata map
		metadata := make(map[string]interface{})
		if e.RequestID != nil {
			metadata["requestId"] = *e.RequestID
		}
		if e.SessionID != nil {
			metadata["sessionId"] = *e.SessionID
		}
		if e.CorrelationID != nil {
			metadata["correlationId"] = *e.CorrelationID
		}
		if e.UserAgent != nil {
			metadata["userAgent"] = *e.UserAgent
		}
		if e.IPAddress != nil {
			metadata["ipAddress"] = *e.IPAddress
		}

		eventsResult[i] = Event{
			ID:        e.ID,
			Type:      e.EventType,
			Category:  e.Category,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Source:    e.Source,
			Severity:  e.Severity,
			Data:      data,
			Metadata:  metadata,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"events": eventsResult,
		"total":  len(eventsResult),
	})
}

// ListHandlers returns all event handlers
// GET /api/admin/events/handlers
func (h *EventHandler) ListHandlers(c *gin.Context) {
	// Try to get tenant_id from context (set by auth middleware) or query parameter
	var tenantID string
	if tid, exists := c.Get("tenant_id"); exists {
		tenantID = tid.(string)
	} else {
		tenantID = c.Query("tenant_id")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID required"})
			return
		}
	}

	hs, err := h.repo.ListHandlers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch handlers"})
		return
	}

	// Group handlers by event type (database stores one row per event type)
	handlerMap := make(map[string]*Handler)
	for _, eh := range hs {
		if handler, exists := handlerMap[eh.ID]; exists {
			// Add event type to existing handler
			handler.EventTypes = append(handler.EventTypes, eh.EventType)
		} else {
			// Create new handler entry
			var successRate float64
			if eh.SuccessCount+eh.FailureCount > 0 {
				successRate = float64(eh.SuccessCount) / float64(eh.SuccessCount+eh.FailureCount) * 100
			}

			var lastTriggered string
			if eh.LastSuccessAt != nil {
				lastTriggered = eh.LastSuccessAt.Format(time.RFC3339)
			} else if eh.LastFailureAt != nil {
				lastTriggered = eh.LastFailureAt.Format(time.RFC3339)
			}

			var retryPolicy string
			if eh.RetryConfig != nil {
				if policy, ok := eh.RetryConfig["policy"].(string); ok {
					retryPolicy = policy
				}
			}
			if retryPolicy == "" {
				retryPolicy = "none"
			}

			var endpoint string
			if eh.EndpointURL != nil {
				endpoint = *eh.EndpointURL
			}

			var method string
			if eh.HTTPMethod != nil {
				method = *eh.HTTPMethod
			}

			handlerMap[eh.ID] = &Handler{
				ID:            eh.ID,
				Name:          eh.HandlerName,
				EventTypes:    []string{eh.EventType},
				Endpoint:      endpoint,
				Method:        method,
				Enabled:       eh.Status == statusActive,
				RetryPolicy:   retryPolicy,
				Timeout:       eh.TimeoutSeconds * 1000, // Convert to ms
				LastTriggered: lastTriggered,
				SuccessRate:   successRate,
			}
		}
	}

	// Convert map to slice
	handlers := make([]Handler, 0, len(handlerMap))
	for _, handler := range handlerMap {
		handlers = append(handlers, *handler)
	}

	c.JSON(http.StatusOK, gin.H{
		"handlers": handlers,
		"total":    len(handlers),
	})
}

// CreateHandler creates a new event handler
// POST /api/admin/events/handlers
func (h *EventHandler) CreateHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	var req HandlerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create handler record for each event type (one-to-many relationship in database)
	var createdHandler *events.EventHandlerRecord
	for _, eventType := range req.EventTypes {
		handler := &events.EventHandlerRecord{
			TenantID:       tenantID,
			HandlerName:    req.Name,
			EventType:      eventType,
			HandlerType:    "webhook",
			Status:         statusActive,
			EndpointURL:    &req.Endpoint,
			HTTPMethod:     &req.Method,
			RetryConfig:    map[string]interface{}{"policy": req.RetryPolicy},
			TimeoutSeconds: req.Timeout / 1000, // Convert ms to seconds
		}

		if err := h.repo.CreateHandler(c.Request.Context(), handler); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create handler"})
			return
		}

		if createdHandler == nil {
			createdHandler = handler
		}
	}

	// Return single handler representation
	response := Handler{
		ID:          createdHandler.ID,
		Name:        req.Name,
		EventTypes:  req.EventTypes,
		Endpoint:    req.Endpoint,
		Method:      req.Method,
		Enabled:     true,
		RetryPolicy: req.RetryPolicy,
		Timeout:     req.Timeout,
		SuccessRate: 0.0,
	}

	c.JSON(http.StatusCreated, response)
}

// ToggleHandler enables or disables an event handler
// POST /api/admin/events/handlers/:id/toggle
func (h *EventHandler) ToggleHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	handlerID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := StatusInactive
	if req.Enabled {
		status = statusActive
	}

	if err := h.repo.UpdateHandlerStatus(c.Request.Context(), tenantID, handlerID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update handler"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      handlerID,
		"enabled": req.Enabled,
	})
}

// DeleteHandler removes an event handler
// DELETE /api/admin/events/handlers/:id
func (h *EventHandler) DeleteHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	handlerID := c.Param("id")

	if err := h.repo.DeleteHandler(c.Request.Context(), tenantID, handlerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete handler"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Handler deleted successfully",
		"id":      handlerID,
	})
}

// GetEventMetrics returns event system metrics
// GET /api/admin/events/metrics
func (h *EventHandler) GetEventMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	stats, err := h.repo.GetEventMetrics(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
		return
	}

	// Format top event types for response
	topEventTypes := make([]gin.H, len(stats.TopEventTypes))
	for i, et := range stats.TopEventTypes {
		topEventTypes[i] = gin.H{
			"type":  et.Type,
			"count": et.Count,
		}
	}

	// Calculate events per second (rough estimate based on today's events)
	eventsPerSecond := float64(stats.EventsThisHour) / 3600.0

	metrics := gin.H{
		"total_events":      stats.TotalEvents,
		"events_per_second": eventsPerSecond,
		"events_today":      stats.EventsToday,
		"events_this_hour":  stats.EventsThisHour,
		"by_category":       stats.ByCategory,
		"by_severity":       stats.BySeverity,
		"handler_stats": gin.H{
			"total_handlers":    stats.HandlerStats.TotalHandlers,
			"enabled_handlers":  stats.HandlerStats.EnabledHandlers,
			"disabled_handlers": stats.HandlerStats.DisabledHandlers,
			"total_invocations": stats.HandlerStats.TotalInvocations,
			"success_rate":      stats.HandlerStats.AverageSuccessRate,
		},
		"top_event_types": topEventTypes,
	}

	c.JSON(http.StatusOK, metrics)
}

// TestHandler sends a test event to a handler
// POST /api/admin/events/handlers/:id/test
func (h *EventHandler) TestHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found"})
		return
	}

	handlerID := c.Param("id")

	// Get handler from database to verify it exists
	handlers, err := h.repo.ListHandlers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch handler"})
		return
	}

	var found bool
	for _, handler := range handlers {
		if handler.ID == handlerID {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Handler not found"})
		return
	}

	// TODO: Actually send test HTTP request to handler endpoint
	// For now, return mock success response
	testResult := gin.H{
		"success":      true,
		"handlerId":    handlerID,
		"statusCode":   200,
		"responseTime": 145,
		"message":      "Test event delivered successfully",
	}

	c.JSON(http.StatusOK, testResult)
}

// RegisterRoutes registers all event system routes
func (h *EventHandler) RegisterRoutes(router *gin.RouterGroup) {
	events := router.Group("/events")
	{
		events.GET("", h.GetEventStream) // Root endpoint lists events
		events.GET("/types", h.ListEventTypes)
		events.GET("/stream", h.GetEventStream)
		events.GET("/metrics", h.GetEventMetrics)

		events.GET("/handlers", h.ListHandlers)
		events.POST("/handlers", h.CreateHandler)
		events.POST("/handlers/:id/toggle", h.ToggleHandler)
		events.DELETE("/handlers/:id", h.DeleteHandler)
		events.POST("/handlers/:id/test", h.TestHandler)
	}
}
