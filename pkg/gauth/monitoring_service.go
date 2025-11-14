// Package gauth - Comprehensive Monitoring and Alerting Service
// Task 7: Implements Prometheus metrics, health checks, compliance violation alerts,
// and performance monitoring dashboards
package gauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MonitoringService provides comprehensive monitoring and alerting
type MonitoringService struct {
	mu                    sync.RWMutex
	config                MonitoringConfig
	metricsRegistry       *prometheus.Registry
	healthChecks          map[string]HealthCheckFunc
	complianceAlerts      []ComplianceAlert
	performanceThresholds map[string]float64
	lastHealthCheckTime   time.Time
	lastComplianceCheck   time.Time
	systemStatus          SystemStatus

	// Prometheus metrics
	validationTotal      *prometheus.CounterVec
	validationDuration   *prometheus.HistogramVec
	validationErrors     *prometheus.CounterVec
	complianceViolations *prometheus.CounterVec
	jurisdictionChecks   *prometheus.CounterVec
	notaryVerifications  *prometheus.CounterVec
	healthCheckStatus    *prometheus.GaugeVec
	systemResourceUsage  *prometheus.GaugeVec
	activeValidations    prometheus.Gauge
	queueDepth           prometheus.Gauge
	cacheHitRate         prometheus.Gauge
	alertsTriggered      *prometheus.CounterVec
}

// MonitoringConfig configures the monitoring service
type MonitoringConfig struct {
	EnableHealthChecks       bool
	EnableComplianceAlerts   bool
	EnablePerformanceMetrics bool
	HealthCheckInterval      time.Duration
	ComplianceCheckInterval  time.Duration
	AlertThresholds          AlertThresholds
	DashboardRefreshRate     time.Duration
}

// AlertThresholds defines thresholds for alerting
type AlertThresholds struct {
	ValidationErrorRate     float64 // Errors per second threshold
	ValidationLatencyP95    float64 // 95th percentile latency in ms
	ComplianceViolationRate float64 // Violations per hour
	HealthCheckFailureCount int     // Consecutive failures before alert
	QueueDepthThreshold     int     // Max queue depth before alert
	CacheHitRateMin         float64 // Minimum cache hit rate
	CPUUsageMax             float64 // Maximum CPU usage percentage
	MemoryUsageMax          float64 // Maximum memory usage percentage
}

// HealthCheckFunc defines a health check function
type HealthCheckFunc func(ctx context.Context) HealthCheckResult

// HealthCheckResult contains health check results
type HealthCheckResult struct {
	Name      string
	Status    HealthStatus
	Message   string
	Duration  time.Duration
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// HealthStatus represents health check status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ComplianceAlert represents a compliance violation alert
type ComplianceAlert struct {
	AlertID      string
	Severity     AlertSeverity
	Type         ComplianceAlertType
	Jurisdiction string
	Message      string
	Details      map[string]interface{}
	Timestamp    time.Time
	Resolved     bool
	ResolvedAt   time.Time
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// ComplianceAlertType defines types of compliance alerts
type ComplianceAlertType string

const (
	AlertTypeJurisdictionViolation ComplianceAlertType = "jurisdiction_violation"
	AlertTypeNotarizationMissing   ComplianceAlertType = "notarization_missing"
	AlertTypeDocumentInvalid       ComplianceAlertType = "document_invalid"
	AlertTypeSignatureInvalid      ComplianceAlertType = "signature_invalid"
	AlertTypeValueLimitExceeded    ComplianceAlertType = "value_limit_exceeded"
	AlertTypeAuthorizationExpired  ComplianceAlertType = "authorization_expired"
	AlertTypeChainViolation        ComplianceAlertType = "chain_violation"
)

// SystemStatus represents overall system status
type SystemStatus struct {
	Overall          HealthStatus
	Components       map[string]HealthCheckResult
	LastUpdate       time.Time
	UptimeSeconds    float64
	TotalValidations int64
	TotalErrors      int64
	ErrorRate        float64
}

// DashboardMetrics contains metrics for dashboard display
type DashboardMetrics struct {
	// Performance metrics
	ValidationLatencyP50 float64
	ValidationLatencyP95 float64
	ValidationLatencyP99 float64
	ValidationsPerSecond float64
	ErrorRate            float64

	// Business metrics
	TotalValidations      int64
	SuccessfulValidations int64
	FailedValidations     int64
	JurisdictionBreakdown map[string]int64

	// Compliance metrics
	ComplianceScore    float64
	ViolationsLast24h  int64
	CriticalViolations int64
	NotarizationRate   float64

	// Resource metrics
	CPUUsagePercent   float64
	MemoryUsageMB     float64
	ActiveConnections int64
	QueueDepth        int64
	CacheHitRate      float64
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(config MonitoringConfig) *MonitoringService {
	registry := prometheus.NewRegistry()

	service := &MonitoringService{
		config:                config,
		metricsRegistry:       registry,
		healthChecks:          make(map[string]HealthCheckFunc),
		complianceAlerts:      []ComplianceAlert{},
		performanceThresholds: make(map[string]float64),
		systemStatus: SystemStatus{
			Overall:    HealthStatusHealthy,
			Components: make(map[string]HealthCheckResult),
			LastUpdate: time.Now(),
		},
	}

	service.initializeMetrics()
	return service
}

// initializeMetrics initializes all Prometheus metrics
func (s *MonitoringService) initializeMetrics() {
	// Validation metrics
	s.validationTotal = promauto.With(s.metricsRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_formal_validation_total",
			Help: "Total number of formal requirement validations",
		},
		[]string{"jurisdiction", "status"},
	)

	s.validationDuration = promauto.With(s.metricsRegistry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_formal_validation_duration_seconds",
			Help:    "Duration of formal requirement validations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 4s
		},
		[]string{"jurisdiction", "validation_type"},
	)

	s.validationErrors = promauto.With(s.metricsRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_formal_validation_errors_total",
			Help: "Total number of validation errors",
		},
		[]string{"error_type", "jurisdiction"},
	)

	// Compliance metrics
	s.complianceViolations = promauto.With(s.metricsRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_compliance_violations_total",
			Help: "Total number of compliance violations",
		},
		[]string{"violation_type", "severity", "jurisdiction"},
	)

	s.jurisdictionChecks = promauto.With(s.metricsRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_jurisdiction_checks_total",
			Help: "Total number of jurisdiction-specific checks",
		},
		[]string{"jurisdiction", "result"},
	)

	s.notaryVerifications = promauto.With(s.metricsRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_notary_verifications_total",
			Help: "Total number of notary verifications",
		},
		[]string{"jurisdiction", "result"},
	)

	// Health check metrics
	s.healthCheckStatus = promauto.With(s.metricsRegistry).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauth_health_check_status",
			Help: "Health check status (1=healthy, 0.5=degraded, 0=unhealthy)",
		},
		[]string{"component"},
	)

	// System resource metrics
	s.systemResourceUsage = promauto.With(s.metricsRegistry).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauth_system_resource_usage",
			Help: "System resource usage",
		},
		[]string{"resource_type"},
	)

	// Real-time metrics
	s.activeValidations = promauto.With(s.metricsRegistry).NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_active_validations",
			Help: "Number of currently active validations",
		},
	)

	s.queueDepth = promauto.With(s.metricsRegistry).NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_queue_depth",
			Help: "Current validation queue depth",
		},
	)

	s.cacheHitRate = promauto.With(s.metricsRegistry).NewGauge(
		prometheus.GaugeOpts{
			Name: "gauth_cache_hit_rate",
			Help: "Cache hit rate (0-1)",
		},
	)

	// Alert metrics
	s.alertsTriggered = promauto.With(s.metricsRegistry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_alerts_triggered_total",
			Help: "Total number of alerts triggered",
		},
		[]string{"alert_type", "severity"},
	)
}

// RegisterHealthCheck registers a health check function
func (s *MonitoringService) RegisterHealthCheck(name string, checkFunc HealthCheckFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthChecks[name] = checkFunc
}

// RunHealthChecks executes all registered health checks
func (s *MonitoringService) RunHealthChecks(ctx context.Context) SystemStatus {
	s.mu.Lock()
	s.lastHealthCheckTime = time.Now()
	s.mu.Unlock()

	results := make(map[string]HealthCheckResult)
	overallHealthy := true
	overallDegraded := false

	for name, checkFunc := range s.healthChecks {
		startTime := time.Now()
		result := checkFunc(ctx)
		result.Duration = time.Since(startTime)
		result.Timestamp = time.Now()

		results[name] = result

		// Update Prometheus metric
		statusValue := s.healthStatusToFloat(result.Status)
		s.healthCheckStatus.WithLabelValues(name).Set(statusValue)

		// Determine overall status
		if result.Status == HealthStatusUnhealthy {
			overallHealthy = false
		} else if result.Status == HealthStatusDegraded {
			overallDegraded = true
		}
	}

	// Determine overall system status
	var overallStatus HealthStatus
	if !overallHealthy {
		overallStatus = HealthStatusUnhealthy
	} else if overallDegraded {
		overallStatus = HealthStatusDegraded
	} else {
		overallStatus = HealthStatusHealthy
	}

	s.mu.Lock()
	s.systemStatus.Overall = overallStatus
	s.systemStatus.Components = results
	s.systemStatus.LastUpdate = time.Now()
	s.mu.Unlock()

	return s.systemStatus
}

// healthStatusToFloat converts health status to numeric value for Prometheus
func (s *MonitoringService) healthStatusToFloat(status HealthStatus) float64 {
	switch status {
	case HealthStatusHealthy:
		return 1.0
	case HealthStatusDegraded:
		return 0.5
	case HealthStatusUnhealthy:
		return 0.0
	default:
		return -1.0
	}
}

// RecordValidation records a validation attempt
func (s *MonitoringService) RecordValidation(
	jurisdiction string,
	success bool,
	duration time.Duration,
	validationType string,
) {
	status := "success"
	if !success {
		status = "failure"
	}

	s.validationTotal.WithLabelValues(jurisdiction, status).Inc()
	s.validationDuration.WithLabelValues(jurisdiction, validationType).Observe(duration.Seconds())

	s.mu.Lock()
	s.systemStatus.TotalValidations++
	if !success {
		s.systemStatus.TotalErrors++
	}
	s.systemStatus.ErrorRate = float64(s.systemStatus.TotalErrors) / float64(s.systemStatus.TotalValidations)
	s.mu.Unlock()
}

// RecordComplianceViolation records a compliance violation
func (s *MonitoringService) RecordComplianceViolation(
	violationType ComplianceAlertType,
	severity AlertSeverity,
	jurisdiction string,
	details map[string]interface{},
) {
	s.complianceViolations.WithLabelValues(string(violationType), string(severity), jurisdiction).Inc()

	alert := ComplianceAlert{
		AlertID:      fmt.Sprintf("%s-%d", violationType, time.Now().Unix()),
		Severity:     severity,
		Type:         violationType,
		Jurisdiction: jurisdiction,
		Message:      s.generateAlertMessage(violationType, jurisdiction),
		Details:      details,
		Timestamp:    time.Now(),
		Resolved:     false,
	}

	s.mu.Lock()
	s.complianceAlerts = append(s.complianceAlerts, alert)
	s.mu.Unlock()

	// Trigger alert if severity is high or critical
	if severity == AlertSeverityCritical || severity == AlertSeverityHigh {
		s.alertsTriggered.WithLabelValues(string(violationType), string(severity)).Inc()
	}
}

// generateAlertMessage generates a human-readable alert message
func (s *MonitoringService) generateAlertMessage(violationType ComplianceAlertType, jurisdiction string) string {
	switch violationType {
	case AlertTypeJurisdictionViolation:
		return fmt.Sprintf("Jurisdiction violation detected in %s", jurisdiction)
	case AlertTypeNotarizationMissing:
		return fmt.Sprintf("Required notarization missing for %s", jurisdiction)
	case AlertTypeDocumentInvalid:
		return fmt.Sprintf("Invalid document detected in %s", jurisdiction)
	case AlertTypeSignatureInvalid:
		return fmt.Sprintf("Invalid signature detected in %s", jurisdiction)
	case AlertTypeValueLimitExceeded:
		return fmt.Sprintf("Value limit exceeded in %s", jurisdiction)
	case AlertTypeAuthorizationExpired:
		return fmt.Sprintf("Expired authorization in %s", jurisdiction)
	case AlertTypeChainViolation:
		return fmt.Sprintf("Authorization chain violation in %s", jurisdiction)
	default:
		return fmt.Sprintf("Compliance violation in %s", jurisdiction)
	}
}

// RecordJurisdictionCheck records a jurisdiction check
func (s *MonitoringService) RecordJurisdictionCheck(jurisdiction string, success bool) {
	result := "pass"
	if !success {
		result = "fail"
	}
	s.jurisdictionChecks.WithLabelValues(jurisdiction, result).Inc()
}

// RecordNotaryVerification records a notary verification
func (s *MonitoringService) RecordNotaryVerification(jurisdiction string, success bool) {
	result := "verified"
	if !success {
		result = "failed"
	}
	s.notaryVerifications.WithLabelValues(jurisdiction, result).Inc()
}

// UpdateSystemResources updates system resource metrics
func (s *MonitoringService) UpdateSystemResources(
	cpuPercent float64,
	memoryMB float64,
	activeConnections int64,
	queueDepth int64,
	cacheHitRate float64,
) {
	s.systemResourceUsage.WithLabelValues("cpu_percent").Set(cpuPercent)
	s.systemResourceUsage.WithLabelValues("memory_mb").Set(memoryMB)
	s.systemResourceUsage.WithLabelValues("active_connections").Set(float64(activeConnections))
	s.queueDepth.Set(float64(queueDepth))
	s.cacheHitRate.Set(cacheHitRate)

	// Check thresholds and trigger alerts
	if cpuPercent > s.config.AlertThresholds.CPUUsageMax {
		s.alertsTriggered.WithLabelValues("cpu_high", "high").Inc()
	}
	if memoryMB > s.config.AlertThresholds.MemoryUsageMax {
		s.alertsTriggered.WithLabelValues("memory_high", "high").Inc()
	}
	if queueDepth > int64(s.config.AlertThresholds.QueueDepthThreshold) {
		s.alertsTriggered.WithLabelValues("queue_depth_high", "medium").Inc()
	}
	if cacheHitRate < s.config.AlertThresholds.CacheHitRateMin {
		s.alertsTriggered.WithLabelValues("cache_hit_rate_low", "low").Inc()
	}
}

// GetDashboardMetrics returns current dashboard metrics
func (s *MonitoringService) GetDashboardMetrics() DashboardMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return DashboardMetrics{
		TotalValidations:      s.systemStatus.TotalValidations,
		SuccessfulValidations: s.systemStatus.TotalValidations - s.systemStatus.TotalErrors,
		FailedValidations:     s.systemStatus.TotalErrors,
		ErrorRate:             s.systemStatus.ErrorRate,
		// Other metrics would be calculated from Prometheus queries
	}
}

// GetActiveAlerts returns all unresolved compliance alerts
func (s *MonitoringService) GetActiveAlerts() []ComplianceAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeAlerts := []ComplianceAlert{}
	for _, alert := range s.complianceAlerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	return activeAlerts
}

// ResolveAlert marks an alert as resolved
func (s *MonitoringService) ResolveAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.complianceAlerts {
		if s.complianceAlerts[i].AlertID == alertID {
			s.complianceAlerts[i].Resolved = true
			s.complianceAlerts[i].ResolvedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", alertID)
}

// GetSystemStatus returns current system status
func (s *MonitoringService) GetSystemStatus() SystemStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemStatus
}

// GetMetricsRegistry returns the Prometheus registry for HTTP handler
func (s *MonitoringService) GetMetricsRegistry() *prometheus.Registry {
	return s.metricsRegistry
}

// Start starts the monitoring service background tasks
func (s *MonitoringService) Start(ctx context.Context) {
	if s.config.EnableHealthChecks {
		go s.healthCheckLoop(ctx)
	}

	if s.config.EnableComplianceAlerts {
		go s.complianceCheckLoop(ctx)
	}
}

// healthCheckLoop runs health checks periodically
func (s *MonitoringService) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunHealthChecks(ctx)
		}
	}
}

// complianceCheckLoop runs compliance checks periodically
func (s *MonitoringService) complianceCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.ComplianceCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			s.lastComplianceCheck = time.Now()
			s.mu.Unlock()
			// Compliance checking logic would go here
		}
	}
}

// SetActiveValidations updates the active validations gauge
func (s *MonitoringService) SetActiveValidations(count int) {
	s.activeValidations.Set(float64(count))
}

// RecordValidationError records a validation error
func (s *MonitoringService) RecordValidationError(errorType string, jurisdiction string) {
	s.validationErrors.WithLabelValues(errorType, jurisdiction).Inc()
}
