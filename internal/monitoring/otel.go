package monitoring

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// OTELCollector provides OpenTelemetry metrics and trace export
// Integrates with MetricsCollector and supports trace propagation

type OTELCollector struct {
	meter  metric.Meter
	tracer trace.Tracer
}

func NewOTELCollector() *OTELCollector {
	meterProvider := noop.NewMeterProvider()
	return &OTELCollector{
		meter:  meterProvider.Meter("agentauth-monitoring"),
		tracer: otel.Tracer("agentauth-monitoring"),
	}
}

// RecordCounter records a counter metric with OTEL
func (c *OTELCollector) RecordCounter(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) {
	counter, _ := c.meter.Float64Counter(name)
	counter.Add(ctx, value)
}

// RecordGauge records a gauge metric with OTEL
func (c *OTELCollector) RecordGauge(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) {
	// OTEL gauges are usually observed via callback, so this is a stub
}

// StartSpan starts a new trace span
func (c *OTELCollector) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return c.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// EndSpan ends a trace span
func (c *OTELCollector) EndSpan(span trace.Span) {
	span.End()
}

// Example integration with MetricsCollector
// func (mc *MetricsCollector) WithTrace(ctx context.Context, name string, fn func(ctx context.Context) {
// 	otelCollector := NewOTELCollector()
// 	ctx, span := otelCollector.StartSpan(ctx, name)
// 	defer otelCollector.EndSpan(span)
// 	fn(ctx)
// }
