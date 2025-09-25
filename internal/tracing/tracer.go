// Package tracing provides OpenTelemetry integration for GAuth
package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider manages OpenTelemetry tracing
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// Config holds configuration for tracing
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
}

// NewTracerProvider creates a new OpenTelemetry tracer provider
func NewTracerProvider(cfg Config) (*TracerProvider, error) {
	// Create OTLP exporter
	// Create stdout exporter for development/testing
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %v", err)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	// Create trace provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set as global trace provider
	otel.SetTracerProvider(provider)

	return &TracerProvider{
		provider: provider,
		tracer:   provider.Tracer(cfg.ServiceName),
	}, nil
}

// StartSpan starts a new span with the given name and attributes
func (tp *TracerProvider) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tp.tracer.Start(ctx, name,
		trace.WithAttributes(attrs...),
		trace.WithTimestamp(time.Now()),
	)
}

// AddEvent adds an event to the current span
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name,
		trace.WithAttributes(attrs...),
		trace.WithTimestamp(time.Now()),
	)
}

// SpanFromContext retrieves the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceID returns the trace ID from the span in context
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().TraceID().String()
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	return tp.provider.Shutdown(ctx)
}

// Common span names
const (
	SpanAuth            = "gauth.auth"          // #nosec G101 - This is a span name, not credentials
	SpanTokenRequest    = "gauth.token.request" // #nosec G101 - This is a span name, not credentials  
	SpanTokenValidation = "gauth.token.validate" // #nosec G101 - This is a span name, not credentials
	SpanTransaction     = "gauth.transaction"
	SpanRateLimit       = "gauth.ratelimit"
	SpanAudit           = "gauth.audit"
)

// Common attribute keys
const (
	AttributeClientID        = attribute.Key("gauth.client.id")
	AttributeResourceID      = attribute.Key("gauth.resource.id")
	AttributeTokenID         = attribute.Key("gauth.token.id")
	AttributeTransactionType = attribute.Key("gauth.transaction.type")
	AttributeStatus          = attribute.Key("gauth.status")
	AttributeError           = attribute.Key("gauth.error")
)
