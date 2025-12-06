package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MCP Metrics
	MCPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_mcp_requests_total",
			Help: "Total number of MCP requests processed",
		},
		[]string{"method", "status"},
	)

	MCPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_mcp_request_duration_seconds",
			Help:    "Duration of MCP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	MCPActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_mcp_active_connections",
			Help: "Number of active MCP connections",
		},
	)

	MCPMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_mcp_messages_received_total",
			Help: "Total number of MCP messages received",
		},
		[]string{"transport", "type"},
	)

	MCPMessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_mcp_messages_sent_total",
			Help: "Total number of MCP messages sent",
		},
		[]string{"transport", "type"},
	)

	// Identity Connector Metrics
	ConnectorValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_connector_validations_total",
			Help: "Total number of identity validations performed",
		},
		[]string{"country", "document_type", "result"},
	)

	ConnectorValidationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_connector_validation_duration_seconds",
			Help:    "Duration of identity validations in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"country", "document_type"},
	)

	ConnectorAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_connector_api_calls_total",
			Help: "Total number of API calls to government systems",
		},
		[]string{"country", "api_name", "status"},
	)

	ConnectorAPICallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_connector_api_call_duration_seconds",
			Help:    "Duration of government API calls in seconds",
			Buckets: []float64{.05, .1, .25, .5, 1, 2, 5, 10, 30},
		},
		[]string{"country", "api_name"},
	)

	ConnectorCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_connector_cache_hits_total",
			Help: "Total number of cache hits for validation results",
		},
		[]string{"country", "document_type"},
	)

	ConnectorCacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_connector_cache_misses_total",
			Help: "Total number of cache misses for validation results",
		},
		[]string{"country", "document_type"},
	)

	// RFC-0111 PoA Metrics
	PoACreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gauth_poa_created_total",
			Help: "Total number of Power of Attorney credentials created",
		},
	)

	PoAValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_poa_validations_total",
			Help: "Total number of PoA validations performed",
		},
		[]string{"result"},
	)

	PoARevokedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gauth_poa_revoked_total",
			Help: "Total number of PoA credentials revoked",
		},
	)

	PoAActiveCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_poa_active_count",
			Help: "Number of currently active PoA credentials",
		},
	)

	// Authorization Metrics
	AuthorizationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_authorizations_total",
			Help: "Total number of authorization requests",
		},
		[]string{"client_id", "status"},
	)

	AuthorizationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_authorization_duration_seconds",
			Help:    "Duration of authorization requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"client_id"},
	)

	TokensIssuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_tokens_issued_total",
			Help: "Total number of access tokens issued",
		},
		[]string{"grant_type", "client_id"},
	)

	TokenValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_token_validations_total",
			Help: "Total number of token validations",
		},
		[]string{"result"},
	)

	// Policy Metrics
	PolicyDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_policy_decisions_total",
			Help: "Total number of policy decisions made",
		},
		[]string{"policy_id", "effect"},
	)

	PolicyEvaluationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gauth_policy_evaluation_duration_seconds",
			Help:    "Duration of policy evaluation in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05},
		},
	)

	// System Health Metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_database_connections_active",
			Help: "Number of active database connections",
		},
	)

	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table", "status"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation", "table"},
	)

	RedisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	RedisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_redis_operation_duration_seconds",
			Help:    "Duration of Redis operations in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05},
		},
		[]string{"operation"},
	)

	// Error Metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "component"},
	)

	// Audit Log Metrics
	AuditLogsWritten = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_audit_logs_written_total",
			Help: "Total number of audit log entries written",
		},
		[]string{"event_type"},
	)

	AuditLogWriteDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gauth_audit_log_write_duration_seconds",
			Help:    "Duration of audit log writes in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1},
		},
	)

	// GAuth+ Metrics
	GAuthPlusValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthplus_validations_total",
			Help: "Total number of GAuth+ validations performed",
		},
		[]string{"feature", "result"},
	)

	GAuthPlusValidationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauthplus_validation_duration_seconds",
			Help:    "Duration of GAuth+ feature validations in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
		},
		[]string{"feature"},
	)

	GAuthPlusCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthplus_cache_hits_total",
			Help: "Total number of GAuth+ cache hits",
		},
		[]string{"cache_type"},
	)

	GAuthPlusCacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthplus_cache_misses_total",
			Help: "Total number of GAuth+ cache misses",
		},
		[]string{"cache_type"},
	)

	GAuthPlusCacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauthplus_cache_size",
			Help: "Current number of entries in GAuth+ caches",
		},
		[]string{"cache_type"},
	)

	GAuthPlusPolicyViolations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthplus_policy_violations_total",
			Help: "Total number of GAuth+ policy violations detected",
		},
		[]string{"policy_type", "severity"},
	)

	GAuthPlusSuccessorActivations = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gauthplus_successor_activations_total",
			Help: "Total number of successor AI activations",
		},
	)

	GAuthPlusDelegationDepth = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gauthplus_delegation_depth",
			Help:    "Distribution of delegation chain depths",
			Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	)

	GAuthPlusDualControlApprovals = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthplus_dual_control_approvals_total",
			Help: "Total number of dual control approvals",
		},
		[]string{"action_type", "status"},
	)

	GAuthPlusCapabilityLevel = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauthplus_capability_level",
			Help: "Current capability level of AI agents (L0-L5 mapped to 0-5)",
		},
		[]string{"agent_id"},
	)

	GAuthPlusFiduciaryViolations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthplus_fiduciary_violations_total",
			Help: "Total number of fiduciary duty violations",
		},
		[]string{"duty_type", "severity"},
	)

	// Rotation Business Metrics
	RotationSignatureVerifyLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gauth_rotation_signature_verify_latency_seconds",
			Help:    "Latency of rotation signature verification",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
		},
	)

	RotationContinuityUpdatesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gauth_rotation_v2_continuity_updates_total",
			Help: "Total number of rotation continuity updates",
		},
	)

	RotationChainLength = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_rotation_summary_chain_length",
			Help: "Length of the rotation chain",
		},
	)

	RotationHeadAge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_rotation_summary_head_age_seconds",
			Help: "Age of the rotation chain head in seconds",
		},
	)

	RotationLastAnchorAge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_rotation_summary_last_anchor_age_seconds",
			Help: "Age of the last rotation anchor in seconds",
		},
	)
)

// RecordMCPRequest records an MCP request with status
func RecordMCPRequest(method, status string, duration float64) {
	MCPRequestsTotal.WithLabelValues(method, status).Inc()
	MCPRequestDuration.WithLabelValues(method).Observe(duration)
}

// RecordConnectorValidation records an identity validation
func RecordConnectorValidation(country, docType, result string, duration float64) {
	ConnectorValidationsTotal.WithLabelValues(country, docType, result).Inc()
	ConnectorValidationDuration.WithLabelValues(country, docType).Observe(duration)
}

// RecordConnectorAPICall records a government API call
func RecordConnectorAPICall(country, apiName, status string, duration float64) {
	ConnectorAPICallsTotal.WithLabelValues(country, apiName, status).Inc()
	ConnectorAPICallDuration.WithLabelValues(country, apiName).Observe(duration)
}

// RecordCacheOperation records a cache hit or miss
func RecordCacheOperation(country, docType string, hit bool) {
	if hit {
		ConnectorCacheHits.WithLabelValues(country, docType).Inc()
	} else {
		ConnectorCacheMisses.WithLabelValues(country, docType).Inc()
	}
}

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordDatabaseQuery records a database operation
func RecordDatabaseQuery(operation, table, status string, duration float64) {
	DatabaseQueriesTotal.WithLabelValues(operation, table, status).Inc()
	DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

// RecordRedisOperation records a Redis operation
func RecordRedisOperation(operation, status string, duration float64) {
	RedisOperationsTotal.WithLabelValues(operation, status).Inc()
	RedisOperationDuration.WithLabelValues(operation).Observe(duration)
}

// RecordError records an error by type and component
func RecordError(errorType, component string) {
	ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordAuditLog records an audit log write
func RecordAuditLog(eventType string, duration float64) {
	AuditLogsWritten.WithLabelValues(eventType).Inc()
	AuditLogWriteDuration.Observe(duration)
}

// GAuth+ Metric Recording Functions

// RecordGAuthPlusValidation records a GAuth+ feature validation
func RecordGAuthPlusValidation(feature, result string, duration float64) {
	GAuthPlusValidationsTotal.WithLabelValues(feature, result).Inc()
	GAuthPlusValidationDuration.WithLabelValues(feature).Observe(duration)
}

// RecordGAuthPlusCacheOperation records a cache hit or miss
func RecordGAuthPlusCacheOperation(cacheType string, hit bool) {
	if hit {
		GAuthPlusCacheHits.WithLabelValues(cacheType).Inc()
	} else {
		GAuthPlusCacheMisses.WithLabelValues(cacheType).Inc()
	}
}

// UpdateGAuthPlusCacheSize updates the current cache size
func UpdateGAuthPlusCacheSize(cacheType string, size int) {
	GAuthPlusCacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// RecordGAuthPlusPolicyViolation records a policy violation
func RecordGAuthPlusPolicyViolation(policyType, severity string) {
	GAuthPlusPolicyViolations.WithLabelValues(policyType, severity).Inc()
}

// RecordGAuthPlusSuccessorActivation records a successor activation
func RecordGAuthPlusSuccessorActivation() {
	GAuthPlusSuccessorActivations.Inc()
}

// RecordGAuthPlusDelegationDepth records the depth of a delegation chain
func RecordGAuthPlusDelegationDepth(depth int) {
	GAuthPlusDelegationDepth.Observe(float64(depth))
}

// RecordGAuthPlusDualControlApproval records a dual control approval
func RecordGAuthPlusDualControlApproval(actionType, status string) {
	GAuthPlusDualControlApprovals.WithLabelValues(actionType, status).Inc()
}

// UpdateGAuthPlusCapabilityLevel updates an agent's capability level (L0-L5 → 0-5)
func UpdateGAuthPlusCapabilityLevel(agentID, level string) {
	var numericLevel float64
	switch level {
	case "L0":
		numericLevel = 0
	case "L1":
		numericLevel = 1
	case "L2":
		numericLevel = 2
	case "L3":
		numericLevel = 3
	case "L4":
		numericLevel = 4
	case "L5":
		numericLevel = 5
	default:
		numericLevel = 0
	}
	GAuthPlusCapabilityLevel.WithLabelValues(agentID).Set(numericLevel)
}

// RecordGAuthPlusFiduciaryViolation records a fiduciary duty violation
func RecordGAuthPlusFiduciaryViolation(dutyType, severity string) {
	GAuthPlusFiduciaryViolations.WithLabelValues(dutyType, severity).Inc()
}
