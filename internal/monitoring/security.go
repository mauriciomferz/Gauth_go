package monitoring

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Security monitoring and alerting enhancements for GAuth

var (
	// Security metrics
	authFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_auth_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"client_id", "reason", "source_ip"},
	)

	suspiciousTokenRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_suspicious_token_requests_total",
			Help: "Total number of suspicious token requests",
		},
		[]string{"client_id", "attack_type", "source_ip"},
	)

	auditLogTamperAttempts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gauth_audit_tamper_attempts_total",
			Help: "Total number of audit log tampering attempts",
		},
	)

	rateLimitExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_rate_limit_exceeded_total",
			Help: "Total number of rate limit violations",
		},
		[]string{"client_id", "endpoint", "source_ip"},
	)

	privilegeEscalationAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_privilege_escalation_attempts_total",
			Help: "Total number of privilege escalation attempts",
		},
		[]string{"client_id", "requested_scope", "source_ip"},
	)
)

// SecurityMonitor provides real-time security monitoring and alerting
type SecurityMonitor struct {
	alertThresholds map[string]int
	alertHandlers   []AlertHandler
}

// AlertHandler defines the interface for handling security alerts
type AlertHandler interface {
	HandleAlert(ctx context.Context, alert SecurityAlert) error
}

// SecurityAlert represents a security incident
type SecurityAlert struct {
	Type      string
	Severity  string
	Message   string
	ClientID  string
	SourceIP  string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// NewSecurityMonitor creates a new security monitor with default thresholds
func NewSecurityMonitor() *SecurityMonitor {
	return &SecurityMonitor{
		alertThresholds: map[string]int{
			"auth_failures":         10, // Alert after 10 failures
			"suspicious_requests":   5,  // Alert after 5 suspicious requests
			"rate_limit_violations": 20, // Alert after 20 rate limit hits
			"privilege_escalation":  1,  // Alert immediately
			"audit_tamper_attempts": 1,  // Alert immediately
		},
		alertHandlers: []AlertHandler{},
	}
}

// AddAlertHandler adds a new alert handler
func (sm *SecurityMonitor) AddAlertHandler(handler AlertHandler) {
	sm.alertHandlers = append(sm.alertHandlers, handler)
}

// RecordAuthFailure records an authentication failure and checks for alerts
func (sm *SecurityMonitor) RecordAuthFailure(clientID, reason, sourceIP string) {
	authFailures.WithLabelValues(clientID, reason, sourceIP).Inc()

	// Check if threshold exceeded
	if sm.shouldAlert("auth_failures", clientID) {
		alert := SecurityAlert{
			Type:      "auth_failures",
			Severity:  "medium",
			Message:   fmt.Sprintf("Multiple authentication failures for client %s", clientID),
			ClientID:  clientID,
			SourceIP:  sourceIP,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"reason":             reason,
				"threshold_exceeded": sm.alertThresholds["auth_failures"],
			},
		}
		sm.sendAlert(alert)
	}
}

// RecordSuspiciousRequest records a suspicious token request
func (sm *SecurityMonitor) RecordSuspiciousRequest(clientID, attackType, sourceIP string) {
	suspiciousTokenRequests.WithLabelValues(clientID, attackType, sourceIP).Inc()

	if sm.shouldAlert("suspicious_requests", clientID) {
		alert := SecurityAlert{
			Type:      "suspicious_requests",
			Severity:  "high",
			Message:   fmt.Sprintf("Suspicious %s attack detected from client %s", attackType, clientID),
			ClientID:  clientID,
			SourceIP:  sourceIP,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"attack_type": attackType,
			},
		}
		sm.sendAlert(alert)
	}
}

// RecordAuditTamperAttempt records an audit log tampering attempt
func (sm *SecurityMonitor) RecordAuditTamperAttempt(sourceIP string) {
	auditLogTamperAttempts.Inc()

	// Always alert on audit tampering attempts
	alert := SecurityAlert{
		Type:      "audit_tamper",
		Severity:  "critical",
		Message:   "Audit log tampering attempt detected",
		SourceIP:  sourceIP,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"immediate_alert": true,
		},
	}
	sm.sendAlert(alert)
}

// RecordRateLimitViolation records a rate limit violation
func (sm *SecurityMonitor) RecordRateLimitViolation(clientID, endpoint, sourceIP string) {
	rateLimitExceeded.WithLabelValues(clientID, endpoint, sourceIP).Inc()

	if sm.shouldAlert("rate_limit_violations", clientID) {
		alert := SecurityAlert{
			Type:      "rate_limit_violation",
			Severity:  "low",
			Message:   fmt.Sprintf("Rate limit exceeded for client %s on endpoint %s", clientID, endpoint),
			ClientID:  clientID,
			SourceIP:  sourceIP,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"endpoint": endpoint,
			},
		}
		sm.sendAlert(alert)
	}
}

// RecordPrivilegeEscalationAttempt records a privilege escalation attempt
func (sm *SecurityMonitor) RecordPrivilegeEscalationAttempt(clientID, requestedScope, sourceIP string) {
	privilegeEscalationAttempts.WithLabelValues(clientID, requestedScope, sourceIP).Inc()

	// Always alert on privilege escalation attempts
	alert := SecurityAlert{
		Type:      "privilege_escalation",
		Severity:  "critical",
		Message:   fmt.Sprintf("Privilege escalation attempt: client %s requested scope %s", clientID, requestedScope),
		ClientID:  clientID,
		SourceIP:  sourceIP,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"requested_scope": requestedScope,
			"immediate_alert": true,
		},
	}
	sm.sendAlert(alert)
}

// shouldAlert determines if an alert should be sent based on thresholds
func (sm *SecurityMonitor) shouldAlert(alertType, clientID string) bool {
	// In a real implementation, this would check current metrics
	// against thresholds and return true if threshold is exceeded
	threshold, exists := sm.alertThresholds[alertType]
	if !exists {
		return false
	}

	// Simplified check - in practice, you'd query Prometheus metrics
	// or maintain internal counters
	return threshold > 0 // Placeholder logic
}

// sendAlert sends an alert to all registered handlers
func (sm *SecurityMonitor) sendAlert(alert SecurityAlert) {
	ctx := context.Background()

	for _, handler := range sm.alertHandlers {
		go func(h AlertHandler) {
			if err := h.HandleAlert(ctx, alert); err != nil {
				log.Printf("Failed to send alert via handler: %v", err)
			}
		}(handler)
	}
}

// EmailAlertHandler sends alerts via email
type EmailAlertHandler struct {
	SMTPServer string
	From       string
	To         []string
}

func (h *EmailAlertHandler) HandleAlert(ctx context.Context, alert SecurityAlert) error {
	// Implementation would send email alert
	log.Printf("EMAIL ALERT: [%s] %s from %s", alert.Severity, alert.Message, alert.SourceIP)
	return nil
}

// SlackAlertHandler sends alerts to Slack
type SlackAlertHandler struct {
	WebhookURL string
	Channel    string
}

func (h *SlackAlertHandler) HandleAlert(ctx context.Context, alert SecurityAlert) error {
	// Implementation would send Slack notification
	log.Printf("SLACK ALERT: [%s] %s from %s", alert.Severity, alert.Message, alert.SourceIP)
	return nil
}

// WebhookAlertHandler sends alerts to a webhook endpoint
type WebhookAlertHandler struct {
	URL string
}

func (h *WebhookAlertHandler) HandleAlert(ctx context.Context, alert SecurityAlert) error {
	// Implementation would POST to webhook
	log.Printf("WEBHOOK ALERT: [%s] %s from %s", alert.Severity, alert.Message, alert.SourceIP)
	return nil
}

// SecurityDashboard provides a real-time security dashboard
type SecurityDashboard struct {
	monitor *SecurityMonitor
}

func NewSecurityDashboard(monitor *SecurityMonitor) *SecurityDashboard {
	return &SecurityDashboard{monitor: monitor}
}

// GetSecurityMetrics returns current security metrics for dashboard
func (sd *SecurityDashboard) GetSecurityMetrics() map[string]interface{} {
	return map[string]interface{}{
		"auth_failures":                 "query from Prometheus",
		"suspicious_requests":           "query from Prometheus",
		"rate_limit_violations":         "query from Prometheus",
		"privilege_escalation_attempts": "query from Prometheus",
		"audit_tamper_attempts":         "query from Prometheus",
		"active_alerts":                 "count of active alerts",
		"last_incident":                 "timestamp of last incident",
	}
}
