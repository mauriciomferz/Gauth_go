// Package security - Security metrics for Prometheus integration
package security

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	// Rate limiting metrics
	RateLimitViolations *prometheus.CounterVec
	DDoSBlocked         prometheus.Counter

	// Authentication metrics
	AuthenticationAttempts *prometheus.CounterVec
	AuthenticationFailures prometheus.Counter

	// Input validation metrics
	SQLInjectionAttempts  prometheus.Counter
	XSSAttempts           prometheus.Counter
	PathTraversalAttempts prometheus.Counter

	// CORS metrics
	CORSRejected prometheus.Counter

	// Token operation metrics
	TokenCreated            prometheus.Counter
	TokenValidationFailures prometheus.Counter

	// Audit metrics
	SecurityEvents         *prometheus.CounterVec
	CriticalSecurityEvents prometheus.Counter

	// Request metrics by security status
	SecureRequests   prometheus.Counter
	InsecureRequests prometheus.Counter

	// Gauge metrics
	ActiveSessions     prometheus.Gauge
	RateLimitedClients prometheus.Gauge
}

var (
	globalSecurityMetrics *SecurityMetrics
	metricsInitialized    uint32
)

// InitSecurityMetrics initializes security metrics
func InitSecurityMetrics() *SecurityMetrics {
	// Use atomic to ensure single initialization
	if !atomic.CompareAndSwapUint32(&metricsInitialized, 0, 1) {
		return globalSecurityMetrics
	}

	metrics := &SecurityMetrics{
		RateLimitViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gauth_rate_limit_violations_total",
				Help: "Total number of rate limit violations",
			},
			[]string{"endpoint", "client_ip"},
		),

		DDoSBlocked: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_ddos_blocked_total",
				Help: "Total number of requests blocked by DDoS protection",
			},
		),

		AuthenticationAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gauth_authentication_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"result"},
		),

		AuthenticationFailures: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_authentication_failures_total",
				Help: "Total number of authentication failures",
			},
		),

		SQLInjectionAttempts: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_sql_injection_attempts_total",
				Help: "Total number of SQL injection attempts detected",
			},
		),

		XSSAttempts: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_xss_attempts_total",
				Help: "Total number of XSS attempts detected",
			},
		),

		PathTraversalAttempts: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_path_traversal_attempts_total",
				Help: "Total number of path traversal attempts detected",
			},
		),

		CORSRejected: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_cors_rejected_total",
				Help: "Total number of rejected CORS requests",
			},
		),

		TokenCreated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_token_created_total",
				Help: "Total number of tokens created",
			},
		),

		TokenValidationFailures: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_token_validation_failures_total",
				Help: "Total number of token validation failures",
			},
		),

		SecurityEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gauth_security_events_total",
				Help: "Total number of security events",
			},
			[]string{"event_type", "severity"},
		),

		CriticalSecurityEvents: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_critical_security_events_total",
				Help: "Total number of critical security events",
			},
		),

		SecureRequests: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_secure_requests_total",
				Help: "Total number of secure requests (passed all security checks)",
			},
		),

		InsecureRequests: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gauth_insecure_requests_total",
				Help: "Total number of insecure requests (failed security checks)",
			},
		),

		ActiveSessions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "gauth_active_sessions",
				Help: "Current number of active sessions",
			},
		),

		RateLimitedClients: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "gauth_rate_limited_clients",
				Help: "Current number of rate-limited clients",
			},
		),
	}

	globalSecurityMetrics = metrics
	return metrics
}

// GetSecurityMetrics returns the global security metrics instance
func GetSecurityMetrics() *SecurityMetrics {
	if globalSecurityMetrics == nil {
		return InitSecurityMetrics()
	}
	return globalSecurityMetrics
}

// RecordRateLimitViolation records a rate limit violation
func RecordRateLimitViolation(endpoint, clientIP string) {
	metrics := GetSecurityMetrics()
	metrics.RateLimitViolations.WithLabelValues(endpoint, clientIP).Inc()
}

// RecordDDoSBlock records a DDoS block
func RecordDDoSBlock() {
	metrics := GetSecurityMetrics()
	metrics.DDoSBlocked.Inc()
}

// RecordAuthenticationAttempt records an authentication attempt
func RecordAuthenticationAttempt(success bool) {
	metrics := GetSecurityMetrics()
	result := "success"
	if !success {
		result = "failure"
		metrics.AuthenticationFailures.Inc()
	}
	metrics.AuthenticationAttempts.WithLabelValues(result).Inc()
}

// RecordSQLInjectionAttempt records a SQL injection attempt
func RecordSQLInjectionAttempt() {
	metrics := GetSecurityMetrics()
	metrics.SQLInjectionAttempts.Inc()
}

// RecordXSSAttempt records an XSS attempt
func RecordXSSAttempt() {
	metrics := GetSecurityMetrics()
	metrics.XSSAttempts.Inc()
}

// RecordPathTraversalAttempt records a path traversal attempt
func RecordPathTraversalAttempt() {
	metrics := GetSecurityMetrics()
	metrics.PathTraversalAttempts.Inc()
}

// RecordCORSRejection records a CORS rejection
func RecordCORSRejection() {
	metrics := GetSecurityMetrics()
	metrics.CORSRejected.Inc()
}

// RecordTokenCreation records a token creation
func RecordTokenCreation() {
	metrics := GetSecurityMetrics()
	metrics.TokenCreated.Inc()
}

// RecordTokenValidationFailure records a token validation failure
func RecordTokenValidationFailure() {
	metrics := GetSecurityMetrics()
	metrics.TokenValidationFailures.Inc()
}

// RecordSecurityEvent records a security event
func RecordSecurityEvent(eventType, severity string) {
	metrics := GetSecurityMetrics()
	metrics.SecurityEvents.WithLabelValues(eventType, severity).Inc()

	if severity == "critical" {
		metrics.CriticalSecurityEvents.Inc()
	}
}

// RecordSecureRequest records a secure request
func RecordSecureRequest() {
	metrics := GetSecurityMetrics()
	metrics.SecureRequests.Inc()
}

// RecordInsecureRequest records an insecure request
func RecordInsecureRequest() {
	metrics := GetSecurityMetrics()
	metrics.InsecureRequests.Inc()
}

// UpdateActiveSessions updates the active sessions gauge
func UpdateActiveSessions(count int) {
	metrics := GetSecurityMetrics()
	metrics.ActiveSessions.Set(float64(count))
}

// UpdateRateLimitedClients updates the rate-limited clients gauge
func UpdateRateLimitedClients(count int) {
	metrics := GetSecurityMetrics()
	metrics.RateLimitedClients.Set(float64(count))
}
