package monitoring

import (
	"context"
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
	otel     *OTELCollector // optional OpenTelemetry exporter
}
// MultiSignatureAdoptionGauge records the current multi-signature adoption rate
func (mc *MetricsCollector) MultiSignatureAdoptionGauge(ctx context.Context, value float64) {
	mc.Gauge("multi_signature_adoption", value, nil)
	if mc.otel != nil {
		mc.otel.RecordGauge(ctx, "multi_signature_adoption", value)
	}
}

// VerificationSuccessCounter increments the verification success counter
func (mc *MetricsCollector) VerificationSuccessCounter(ctx context.Context) {
	mc.Counter("verification_success_total", 1, nil)
	if mc.otel != nil {
		mc.otel.RecordCounter(ctx, "verification_success_total", 1)
	}
}

// VerificationFailureCounter increments the verification failure counter
func (mc *MetricsCollector) VerificationFailureCounter(ctx context.Context) {
	mc.Counter("verification_failure_total", 1, nil)
	if mc.otel != nil {
		mc.otel.RecordCounter(ctx, "verification_failure_total", 1)
	}
}

// WithTrace runs a function within a trace span if OTEL is enabled
func (mc *MetricsCollector) WithTrace(ctx context.Context, name string, fn func(ctx context.Context)) {
	if mc.otel != nil {
		ctx, span := mc.otel.StartSpan(ctx, name)
		defer mc.otel.EndSpan(span)
		fn(ctx)
	} else {
		fn(ctx)
	}
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
		otel:     NewOTELCollector(),
	}
// Example: instrument a metric update with trace
	return &MetricsCollector{
		metrics:  NewMetrics(),
		counters: make(map[string]float64),
		gauges:   make(map[string]float64),
		otel:     NewOTELCollector(),
	}
}
 
// Example: instrument a metric update with trace
func (mc *MetricsCollector) RecordWithTrace(ctx context.Context, name string, value float64) {
	mc.WithTrace(ctx, "metric_update:"+name, func(ctx context.Context) {
		mc.Counter(name, value, nil)
		if mc.otel != nil {
			mc.otel.RecordCounter(ctx, name, value)
		}
	})
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
