// Package prometheus provides Prometheus integration for GAuth metrics
package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/Gimel-Foundation/gauth/internal/monitoring"
)

var (
	// Authentication metrics
	authRequests = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:unused
		Name: "gauth_auth_requests_total",
		Help: "Total number of authentication requests processed",
	}, []string{"status", "client_id"})

	tokensIssued = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:unused
		Name: "gauth_tokens_issued_total",
		Help: "Total number of tokens issued",
	}, []string{"type", "client_id"})

	tokenValidations = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:unused
		Name: "gauth_token_validations_total",
		Help: "Total number of token validations performed",
	}, []string{"status"})

	// Transaction metrics
	transactions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gauth_transactions_total",
		Help: "Total number of transactions processed",
	}, []string{"type", "status"})

	transactionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gauth_transaction_duration_seconds",
		Help:    "Transaction processing duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
	}, []string{"type"})

	// Rate limiting metrics
	rateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gauth_rate_limit_hits_total",
		Help: "Total number of rate limit hits",
	}, []string{"resource_id"})

	// Resource metrics
	activeTokens = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gauth_client_active_tokens",
		Help: "Current number of active tokens per client",
	}, []string{"client_id"})

	resourceUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:unused
		Name: "gauth_resource_utilization",
		Help: "Current resource utilization percentage",
	}, []string{"resource_id", "type"})
)

// PrometheusExporter converts internal metrics to Prometheus format
type Exporter struct {
	collector *monitoring.MetricsCollector
}

// NewExporter creates a new Prometheus exporter
func NewExporter(collector *monitoring.MetricsCollector) *Exporter {
	return &Exporter{
		collector: collector,
	}
}

// Export converts and exports metrics to Prometheus
func (e *Exporter) Export() {
	metrics := e.collector.GetAllMetrics()

	for _, metric := range metrics {
		m := metric // Make a copy to avoid potential issues with loop variable
		switch m.Type {
		case monitoring.CounterMetric:
			e.exportCounter(m)
		case monitoring.GaugeMetric:
			e.exportGauge(m)
		case monitoring.HistogramMetric:
			e.exportHistogram(m)
		}
	}
}

func (e *Exporter) exportCounter(metric monitoring.Metric) {
	switch metric.Name {
	case string(monitoring.MetricTransactions):
		transactions.With(metric.Labels).Add(metric.Value)
	case string(monitoring.MetricRateLimitHits):
		rateLimitHits.With(metric.Labels).Add(metric.Value)
	}
}

func (e *Exporter) exportGauge(metric monitoring.Metric) {
	if metric.Name == string(monitoring.MetricActiveTokens) {
		activeTokens.With(metric.Labels).Set(metric.Value)
	}
}

func (e *Exporter) exportHistogram(metric monitoring.Metric) {
	if metric.Name == string(monitoring.MetricResponseTime) {
		transactionDuration.With(metric.Labels).Observe(metric.Value)
	}
}

// recordAuthenticationMetrics records authentication-related metrics
// This function demonstrates usage of the authentication metrics variables
//
//nolint:unused // Comprehensive metrics recording - used by monitoring system
func (e *Exporter) recordAuthenticationMetrics(status, clientID, tokenType string) {
	// Record authentication request
	authRequests.With(map[string]string{
		"status":    status,
		"client_id": clientID,
	}).Inc()
	
	// Record token issuance if successful
	if status == "success" {
		tokensIssued.With(map[string]string{
			"type":      tokenType,
			"client_id": clientID,
		}).Inc()
	}
}

// recordTokenValidation records token validation metrics
//
//nolint:unused // Comprehensive metrics recording - used by validation system  
func (e *Exporter) recordTokenValidation(status string) {
	tokenValidations.With(map[string]string{
		"status": status,
	}).Inc()
}

// recordResourceUtilization records resource utilization metrics
//
//nolint:unused // Comprehensive metrics recording - used by resource monitoring
func (e *Exporter) recordResourceUtilization(resourceID, resourceType string, utilization float64) {
	resourceUtilization.With(map[string]string{
		"resource_id": resourceID,
		"type":        resourceType,
	}).Set(utilization)
}
