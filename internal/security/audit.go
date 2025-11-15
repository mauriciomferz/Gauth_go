// Package security - Comprehensive audit logging for security events
package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	Severity     string                 `json:"severity"`
	Actor        string                 `json:"actor,omitempty"`
	ClientIP     string                 `json:"client_ip,omitempty"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource,omitempty"`
	Result       string                 `json:"result"`
	Message      string                 `json:"message,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Method       string                 `json:"method,omitempty"`
	Path         string                 `json:"path,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ResponseTime int64                  `json:"response_time_ms,omitempty"`
}

// AuditLogger manages security audit logging
type AuditLogger struct {
	events     []AuditEvent
	mu         sync.RWMutex
	maxEvents  int
	logToFile  bool
	logFile    *os.File
	logToStdout bool
}

// Global audit logger instance
var globalAuditLogger *AuditLogger
var auditLoggerOnce sync.Once

// InitAuditLogger initializes the global audit logger
func InitAuditLogger() *AuditLogger {
	auditLoggerOnce.Do(func() {
		maxEvents := getEnvInt("GAUTH_AUDIT_MAX_EVENTS", 10000)
		logToFile := os.Getenv("GAUTH_AUDIT_LOG_FILE") != ""
		logToStdout := os.Getenv("GAUTH_AUDIT_LOG_STDOUT") == "1"
		
		logger := &AuditLogger{
			events:      make([]AuditEvent, 0, maxEvents),
			maxEvents:   maxEvents,
			logToStdout: logToStdout,
		}
		
		// Open log file if configured
		if logToFile {
			logPath := os.Getenv("GAUTH_AUDIT_LOG_FILE")
			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				log.Printf("Failed to open audit log file: %v", err)
			} else {
				logger.logFile = file
				logger.logToFile = true
			}
		}
		
		globalAuditLogger = logger
	})
	
	return globalAuditLogger
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(event AuditEvent) {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// Store in memory
	a.mu.Lock()
	if len(a.events) >= a.maxEvents {
		// Remove oldest event
		a.events = a.events[1:]
	}
	a.events = append(a.events, event)
	a.mu.Unlock()
	
	// Log to file if configured
	if a.logToFile && a.logFile != nil {
		data, err := json.Marshal(event)
		if err == nil {
			fmt.Fprintf(a.logFile, "%s\n", data)
		}
	}
	
	// Log to stdout if configured
	if a.logToStdout {
		data, _ := json.MarshalIndent(event, "", "  ")
		log.Printf("[AUDIT] %s\n", data)
	}
}

// GetEvents returns recent audit events
func (a *AuditLogger) GetEvents(limit int) []AuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if limit <= 0 || limit > len(a.events) {
		limit = len(a.events)
	}
	
	// Return most recent events
	start := len(a.events) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]AuditEvent, limit)
	copy(result, a.events[start:])
	
	return result
}

// GetEventsByType returns events of a specific type
func (a *AuditLogger) GetEventsByType(eventType string, limit int) []AuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	result := make([]AuditEvent, 0, limit)
	
	// Iterate backwards to get most recent first
	for i := len(a.events) - 1; i >= 0 && len(result) < limit; i-- {
		if a.events[i].EventType == eventType {
			result = append(result, a.events[i])
		}
	}
	
	return result
}

// Close closes the audit logger
func (a *AuditLogger) Close() error {
	if a.logFile != nil {
		return a.logFile.Close()
	}
	return nil
}

// AuditMiddleware creates middleware for comprehensive request auditing
func AuditMiddleware(logger *AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Header("X-Request-ID", requestID)
		}
		
		// Store request ID in context
		c.Set("request_id", requestID)
		
		// Process request
		c.Next()
		
		// Calculate response time
		responseTime := time.Since(startTime).Milliseconds()
		
		// Determine if this should be audited
		if shouldAuditRequest(c) {
			event := AuditEvent{
				Timestamp:    startTime,
				EventType:    "http_request",
				Severity:     getSeverityForStatus(c.Writer.Status()),
				ClientIP:     getClientIP(c),
				Action:       c.Request.Method,
				Resource:     c.Request.URL.Path,
				Result:       getResultForStatus(c.Writer.Status()),
				RequestID:    requestID,
				UserAgent:    c.Request.UserAgent(),
				Method:       c.Request.Method,
				Path:         c.Request.URL.Path,
				StatusCode:   c.Writer.Status(),
				ResponseTime: responseTime,
			}
			
			// Add security event details if present
			if secEvent, exists := c.Get("security_event"); exists {
				if event.Details == nil {
					event.Details = make(map[string]interface{})
				}
				event.Details["security_event"] = secEvent
				event.EventType = "security_event"
				event.Severity = "warning"
			}
			
			// Add rate limit details
			if c.GetBool("rate_limited") {
				if event.Details == nil {
					event.Details = make(map[string]interface{})
				}
				event.Details["rate_limited"] = true
				event.Severity = "warning"
			}
			
			logger.LogEvent(event)
		}
	}
}

// shouldAuditRequest determines if a request should be audited
func shouldAuditRequest(c *gin.Context) bool {
	// Always audit errors
	if c.Writer.Status() >= 400 {
		return true
	}
	
	// Always audit security events
	if _, exists := c.Get("security_event"); exists {
		return true
	}
	
	// Always audit authentication and sensitive endpoints
	path := c.Request.URL.Path
	sensitivePatterns := []string{
		"/api/v1/beta/tokens",
		"/api/v1/beta/delegation",
		"/api/v1/beta/pvp",
		"/api/v1/beta/audit",
		"/api/v1/beta/subscriptions",
	}
	
	for _, pattern := range sensitivePatterns {
		if matchesPattern(path, pattern) {
			return true
		}
	}
	
	// Audit POST, PUT, DELETE operations
	method := c.Request.Method
	if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
		return true
	}
	
	return false
}

// getSeverityForStatus returns severity based on HTTP status code
func getSeverityForStatus(status int) string {
	if status >= 500 {
		return "error"
	} else if status >= 400 {
		return "warning"
	} else if status >= 300 {
		return "info"
	}
	return "info"
}

// getResultForStatus returns result string based on HTTP status code
func getResultForStatus(status int) string {
	if status >= 200 && status < 300 {
		return "success"
	} else if status >= 400 && status < 500 {
		return "client_error"
	} else if status >= 500 {
		return "server_error"
	}
	return "unknown"
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// LogSecurityEvent logs a security-specific event
func LogSecurityEvent(eventType, action, result, message string, details map[string]interface{}) {
	if globalAuditLogger == nil {
		return
	}
	
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		Severity:  "warning",
		Action:    action,
		Result:    result,
		Message:   message,
		Details:   details,
	}
	
	globalAuditLogger.LogEvent(event)
}

// LogAuthenticationAttempt logs an authentication attempt
func LogAuthenticationAttempt(clientIP, actor, result, message string) {
	if globalAuditLogger == nil {
		return
	}
	
	severity := "info"
	if result == "failure" {
		severity = "warning"
	}
	
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "authentication",
		Severity:  severity,
		ClientIP:  clientIP,
		Actor:     actor,
		Action:    "authenticate",
		Result:    result,
		Message:   message,
	}
	
	globalAuditLogger.LogEvent(event)
}

// LogTokenOperation logs a token-related operation
func LogTokenOperation(clientIP, actor, operation, result string, details map[string]interface{}) {
	if globalAuditLogger == nil {
		return
	}
	
	severity := "info"
	if result == "failure" {
		severity = "warning"
	}
	
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "token_operation",
		Severity:  severity,
		ClientIP:  clientIP,
		Actor:     actor,
		Action:    operation,
		Result:    result,
		Details:   details,
	}
	
	globalAuditLogger.LogEvent(event)
}

// LogAdministrativeAction logs an administrative action
func LogAdministrativeAction(clientIP, actor, action, resource, result string, details map[string]interface{}) {
	if globalAuditLogger == nil {
		return
	}
	
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "administrative",
		Severity:  "warning",
		ClientIP:  clientIP,
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		Result:    result,
		Details:   details,
	}
	
	globalAuditLogger.LogEvent(event)
}

// GetAuditSummary returns a summary of audit events
func GetAuditSummary() map[string]interface{} {
	if globalAuditLogger == nil {
		return nil
	}
	
	globalAuditLogger.mu.RLock()
	defer globalAuditLogger.mu.RUnlock()
	
	summary := make(map[string]interface{})
	summary["total_events"] = len(globalAuditLogger.events)
	
	// Count by event type
	eventTypes := make(map[string]int)
	severityCounts := make(map[string]int)
	
	for _, event := range globalAuditLogger.events {
		eventTypes[event.EventType]++
		severityCounts[event.Severity]++
	}
	
	summary["event_types"] = eventTypes
	summary["severity_counts"] = severityCounts
	summary["max_capacity"] = globalAuditLogger.maxEvents
	
	return summary
}
