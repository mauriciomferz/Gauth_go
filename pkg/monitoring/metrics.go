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

// Counter increments a counter metric
func (mc *MetricsCollector) Counter(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	key := name + labelsKey(labels)
	metric, ok := mc.metrics[key]
	if !ok {
		metric = Metric{Name: name, Type: CounterMetric, Labels: labels, LastUpdated: time.Now()}
	}
	metric.Value += value
	metric.LastUpdated = time.Now()
	mc.metrics[key] = metric
}

// Gauge sets a gauge metric
func (mc *MetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	key := name + labelsKey(labels)
	metric := Metric{Name: name, Type: GaugeMetric, Value: value, Labels: labels, LastUpdated: time.Now()}
	mc.metrics[key] = metric
}

// GetAll returns all metrics
func (mc *MetricsCollector) GetAll() map[string]Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	copy := make(map[string]Metric, len(mc.metrics))
	for k, v := range mc.metrics {
		copy[k] = v
	}
	return copy
}

// Helper to create a unique key for a set of labels
func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	key := ""
	for k, v := range labels {
		key += ";" + k + "=" + v
	}
	return key
}
