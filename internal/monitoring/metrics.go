// Package monitoring provides metrics and monitoring capabilities for GAuth
package monitoring

import (
	"sync"
	"time"
)

// MetricType represents the type of metric being tracked
type MetricType string

const (
	CounterMetric   MetricType = "counter"
	GaugeMetric     MetricType = "gauge"
	HistogramMetric MetricType = "histogram"
)

// Metric represents a single monitored metric
type Metric struct {
	Name        string
	Type        MetricType
	Value       float64
	Labels      map[string]string
	LastUpdated time.Time
}

// Common metric names
const (
	MetricAuthRequests      = "auth_requests_total"
	MetricTokensIssued      = "tokens_issued_total"
	MetricTokenValidations  = "token_validations_total"
	MetricTransactions      = "transactions_total"
	MetricTransactionErrors = "transaction_errors_total"
	MetricRateLimitHits     = "rate_limit_hits_total"
	MetricActiveTokens      = "active_tokens"
	MetricAuditEvents       = "audit_events_total"
	MetricResponseTime      = "response_time_seconds"
)

// MetricsCollector manages system-wide metrics
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]Metric
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]Metric),
	}
}

// Counter increments a counter metric by the given value
func (m *MetricsCollector) Counter(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.metricKey(name, labels)
	if metric, exists := m.metrics[key]; exists {
		metric.Value += value
		metric.LastUpdated = time.Now()
		m.metrics[key] = metric
	} else {
		m.metrics[key] = Metric{
			Name:        name,
			Type:        CounterMetric,
			Value:       value,
			Labels:      labels,
			LastUpdated: time.Now(),
		}
	}
}

// Gauge sets a gauge metric to the given value
func (m *MetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.metricKey(name, labels)
	m.metrics[key] = Metric{
		Name:        name,
		Type:        GaugeMetric,
		Value:       value,
		Labels:      labels,
		LastUpdated: time.Now(),
	}
}

// GetAllMetrics returns a map of all metrics
func (m *MetricsCollector) GetAllMetrics() map[string]Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]Metric, len(m.metrics))
	for k, v := range m.metrics {
		metrics[k] = v
	}
	return metrics
}

// metricKey generates a unique key for a metric based on its name and labels
func (m *MetricsCollector) metricKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += ";" + k + "=" + v
	}
	return key
}

// Reset clears all metrics
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = make(map[string]Metric)
}
