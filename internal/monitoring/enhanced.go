package monitoring

import (
	"sync"
)

// Metric constants
const (
	MetricRateLimitHits = "rate_limit_hits"
	MetricResponseTime  = "response_time"
)

// MetricsCollector provides an enhanced interface for metrics collection
type MetricsCollector struct {
	metrics  *Metrics
	counters map[string]float64
	gauges   map[string]float64
	mutex    sync.RWMutex
}

// DefaultMetricsCollector is a compatibility alias expected by examples.
type DefaultMetricsCollector = MetricsCollector

// NewDefaultMetricsCollector creates a default collector.
func NewDefaultMetricsCollector() *DefaultMetricsCollector { return NewMetricsCollector() }

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics:  NewMetrics(),
		counters: make(map[string]float64),
		gauges:   make(map[string]float64),
	}
}

// Counter increments a counter metric
func (mc *MetricsCollector) Counter(name string, value float64, labels map[string]string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.counters[name] += value

	// Store in custom metrics
	mc.metrics.mu.Lock()
	if mc.metrics.CustomMetrics == nil {
		mc.metrics.CustomMetrics = make(map[string]interface{})
	}
	mc.metrics.CustomMetrics[name] = mc.counters[name]
	mc.metrics.mu.Unlock()
}

// Gauge sets a gauge metric value
func (mc *MetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.gauges[name] = value

	// Store in custom metrics
	mc.metrics.mu.Lock()
	if mc.metrics.CustomMetrics == nil {
		mc.metrics.CustomMetrics = make(map[string]interface{})
	}
	mc.metrics.CustomMetrics[name] = value
	mc.metrics.mu.Unlock()
}

// MetricValue represents a metric with its value and labels
type MetricValue struct {
	Value  float64
	Labels map[string]string
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]MetricValue {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]MetricValue)

	// Add counters
	for name, value := range mc.counters {
		result[name] = MetricValue{
			Value:  value,
			Labels: map[string]string{"type": "counter"},
		}
	}

	// Add gauges
	for name, value := range mc.gauges {
		result[name] = MetricValue{
			Value:  value,
			Labels: map[string]string{"type": "gauge"},
		}
	}

	return result
}

// IncrementWithLabels increments a counter metric with labels (compatibility for resilient/basic.go)
func (mc *MetricsCollector) IncrementWithLabels(name string, labels map[string]string) {
	mc.Counter(name, 1, labels)
}

// GaugeWithLabels sets a gauge metric value with labels (compatibility for resilient/basic.go)
func (mc *MetricsCollector) GaugeWithLabels(name string, value float64, labels map[string]string) {
	mc.Gauge(name, value, labels)
}
