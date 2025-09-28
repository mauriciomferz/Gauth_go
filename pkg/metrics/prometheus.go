package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Registration state tracking
var (
	metricsRegistered = false
)

var (
	authLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_authentication_duration_seconds",
			Help:    "Authentication request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // from 1ms to ~1s
		},
		[]string{"method"},
	)

	tokenOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_token_operations_total",
			Help: "Total number of token operations",
		},
		[]string{"operation", "type", "status"},
	)

	tokenValidationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_token_validation_errors_total",
			Help: "Total number of token validation errors",
		},
		[]string{"type", "error"},
	)
	// Authentication metrics
	authAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_authentication_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"method", "status"},
	)
	// ... other metric vars ...
	customMetrics = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauth_custom_metrics",
			Help: "Custom metrics for GAuth resource/service usage.",
		},
		[]string{"name"},
	)
	activeTokens = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauth_active_tokens",
			Help: "Number of currently active tokens",
		},
		[]string{"type"},
	)

	// Authorization metrics
	authzDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_authorization_decisions_total",
			Help: "Total number of authorization decisions",
		},
		[]string{"allowed", "policy"},
	)

	authzLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_authorization_duration_seconds",
			Help:    "Authorization request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10), // from 0.1ms to ~0.1s
		},
		[]string{"policy"},
	)

	policyEvaluations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_policy_evaluations_total",
			Help: "Total number of policy evaluations",
		},
		[]string{"policy", "result"},
	)

	cacheOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "status"},
	)

	// Resource metrics
	resourceAccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_resource_access_total",
			Help: "Total number of resource access attempts",
		},
		[]string{"resource", "action", "allowed"},
	)
)

// RegisterMetrics registers all GAuth metrics with Prometheus
// This function is idempotent and safe to call multiple times
func RegisterMetrics() {
	if metricsRegistered {
		return
	}

	// Register all metrics
	prometheus.MustRegister(
		authAttempts,
		authLatency,
		tokenOperations,
		tokenValidationErrors,
		activeTokens,
		authzDecisions,
		authzLatency,
		policyEvaluations,
		cacheOperations,
		resourceAccess,
	)

	metricsRegistered = true
}

// MetricsCollector provides methods to record various metrics
type MetricsCollector struct {
	ctx context.Context
}

// RecordValue records a generic float64 value for a named metric (for resource/service metrics)
func (m *MetricsCollector) RecordValue(name string, value float64) {
	customMetrics.WithLabelValues(name).Set(value)
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(ctx context.Context) *MetricsCollector {
	return &MetricsCollector{ctx: ctx}
}

// RecordAuthAttempt records an authentication attempt
func (m *MetricsCollector) RecordAuthAttempt(method, status string) {
	authAttempts.WithLabelValues(method, status).Inc()
}

// ObserveAuthLatency records authentication latency
func (m *MetricsCollector) ObserveAuthLatency(method string, duration time.Duration) {
	authLatency.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordTokenOperation records a token operation
func (m *MetricsCollector) RecordTokenOperation(operation, tokenType, status string) {
	tokenOperations.WithLabelValues(operation, tokenType, status).Inc()
}

// RecordTokenValidationError records a token validation error
func (m *MetricsCollector) RecordTokenValidationError(tokenType, errorType string) {
	tokenValidationErrors.WithLabelValues(tokenType, errorType).Inc()
}

// SetActiveTokens sets the number of active tokens
func (m *MetricsCollector) SetActiveTokens(tokenType string, count float64) {
	activeTokens.WithLabelValues(tokenType).Set(count)
}

// RecordAuthzDecision records an authorization decision
func (m *MetricsCollector) RecordAuthzDecision(allowed bool, policy string) {
	authzDecisions.WithLabelValues(boolToString(allowed), policy).Inc()
}

// ObserveAuthzLatency records authorization latency
func (m *MetricsCollector) ObserveAuthzLatency(policy string, duration time.Duration) {
	authzLatency.WithLabelValues(policy).Observe(duration.Seconds())
}

// RecordPolicyEvaluation records a policy evaluation
func (m *MetricsCollector) RecordPolicyEvaluation(policy string, result string) {
	policyEvaluations.WithLabelValues(policy, result).Inc()
}

// RecordCacheOperation records a cache operation
func (m *MetricsCollector) RecordCacheOperation(operation, status string) {
	cacheOperations.WithLabelValues(operation, status).Inc()
}

// RecordResourceAccess records a resource access attempt
func (m *MetricsCollector) RecordResourceAccess(resource, action string, allowed bool) {
	resourceAccess.WithLabelValues(resource, action, boolToString(allowed)).Inc()
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Timer provides a convenient way to measure and record operation duration
type Timer struct {
	start     time.Time
	method    string
	collector *MetricsCollector
	isAuthz   bool
}

// NewTimer creates a new timer for measuring operation duration
func (m *MetricsCollector) NewTimer(method string, isAuthz bool) *Timer {
	return &Timer{
		start:     time.Now(),
		method:    method,
		collector: m,
		isAuthz:   isAuthz,
	}
}

// Stop stops the timer and records the duration
func (t *Timer) Stop() {
	duration := time.Since(t.start)
	if t.isAuthz {
		t.collector.ObserveAuthzLatency(t.method, duration)
	} else {
		t.collector.ObserveAuthLatency(t.method, duration)
	}
}
