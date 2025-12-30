package tracing

import (
	"context"
	"fmt"
	"time"
)

// Span represents a simple trace span used for in-repo demos and tests.
type Span struct {
	TraceID   string                 `json:"trace_id"`
	SpanID    string                 `json:"span_id"`
	ParentID  string                 `json:"parent_id,omitempty"`
	Operation string                 `json:"operation"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Tags      map[string]interface{} `json:"tags"`
	Status    string                 `json:"status"`
}

// End marks the span as finished (compatibility for examples/resilient)
func (span *Span) End() {
	span.EndTime = time.Now()
}

// SetStatus sets the status of the span (compatibility for examples/resilient)
func (span *Span) SetStatus(code interface{}, msg string) {
	// Accepts otel/codes.Code or string, but just stores msg for demo
	if msg != "" {
		span.Status = msg
	} else if code != nil {
		span.Status = fmt.Sprintf("%v", code)
	}
}

// (Removed duplicate Attribute, AddEvent, and predefined attribute keys)

// Tracer is a tiny in-repo tracer implementation used by examples and tests.
type Tracer struct {
	serviceName string
	spans       []*Span
}

// NewTracer creates a new in-memory Tracer.
func NewTracer(serviceName string) *Tracer {
	return &Tracer{serviceName: serviceName, spans: make([]*Span, 0)}
}

// StartSpan creates and returns a new Span and a derived context that carries it.
func (t *Tracer) StartSpan(ctx context.Context, operation string) (*Span, context.Context) {
	span := &Span{
		TraceID:   fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		SpanID:    fmt.Sprintf("span-%d", time.Now().UnixNano()),
		Operation: operation,
		StartTime: time.Now(),
		Tags:      make(map[string]interface{}),
		Status:    "ok",
	}

	if parent := SpanFromContext(ctx); parent != nil {
		span.ParentID = parent.SpanID
		span.TraceID = parent.TraceID
	}

	t.spans = append(t.spans, span)
	ctx = ContextWithSpan(ctx, span)
	return span, ctx
}

// FinishSpan marks the span end time.
func (t *Tracer) FinishSpan(span *Span) { span.EndTime = time.Now() }

// SetTag sets a key/value tag on the span.
func (span *Span) SetTag(key string, value interface{}) {
	if span.Tags == nil {
		span.Tags = make(map[string]interface{})
	}
	span.Tags[key] = value
}

// GetSpans returns all captured spans.
func (t *Tracer) GetSpans() []*Span { return t.spans }

// Context key and helpers
type spanKeyType struct{}

var spanKey = spanKeyType{}

func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanKey, span)
}

func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(spanKey); v != nil {
		if s, ok := v.(*Span); ok {
			return s
		}
	}
	return nil
}

// Demo helper used by examples/tests

func Demo() error {
	fmt.Println("=== Tracing Demo ===")
	tracer := NewTracer("agentauth-service")
	ctx := context.Background()

	root, ctx := tracer.StartSpan(ctx, "root-operation")
	root.SetTag("user", "demo")
	time.Sleep(5 * time.Millisecond)
	child, _ := tracer.StartSpan(ctx, "child-op")
	child.SetTag("step", 1)
	time.Sleep(3 * time.Millisecond)
	tracer.FinishSpan(child)
	tracer.FinishSpan(root)

	for _, s := range tracer.GetSpans() {
		dur := s.EndTime.Sub(s.StartTime)
		fmt.Printf("Span %s (%s) duration=%s\n", s.Operation, s.SpanID, dur)
	}

	return nil
}

// --- Compatibility layer for examples/resilient ---

// Config represents tracer provider configuration expected by examples.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
}

// TracerProvider is a thin wrapper exposing StartSpan the way examples expect.
type TracerProvider struct{ tracer *Tracer }

// NewTracerProvider creates a TracerProvider (OTLP fields ignored in stub).
func NewTracerProvider(cfg Config) (*TracerProvider, error) {
	return &TracerProvider{tracer: NewTracer(cfg.ServiceName)}, nil
}

// Tracer returns the underlying tracer.
func (p *TracerProvider) Tracer() *Tracer {
	return p.tracer
}

// Attribute is a generic key/value used for span tagging.
type Attribute struct {
	Key   string
	Value interface{}
}

// String helper for attribute creation.
func (a Attribute) String(v string) Attribute { return Attribute{Key: a.Key, Value: v} }

// Predefined attribute keys required by examples.
var (
	SpanTransaction          = "transaction"
	AttributeTransactionType = Attribute{Key: "transaction.type"}
	AttributeResourceID      = Attribute{Key: "resource.id"}
	AttributeError           = Attribute{Key: "error"}
)

// Lightweight status codes for examples (decoupled from OpenTelemetry)
const (
	StatusOK    = "ok"
	StatusError = "error"
)

// StartSpan starts a new span and returns new context and a lightweight span wrapper.
func (p *TracerProvider) StartSpan(ctx context.Context, operation string, attrs ...Attribute) (context.Context, *Span) {
	span, ctx2 := p.tracer.StartSpan(ctx, operation)
	for _, at := range attrs {
		span.SetTag(at.Key, at.Value)
	}
	return ctx2, span
}

// AddEvent attaches an event (as tags snapshot) to span.
func AddEvent(span *Span, name string, attrs ...Attribute) {
	if span == nil {
		return
	}
	span.SetTag("event:"+name, time.Now().UnixNano())
	for _, a := range attrs {
		span.SetTag(a.Key, a.Value)
	}
}

// Spans returns the slice of recorded spans (for tests / introspection).
func (p *TracerProvider) Spans() []*Span {
	if p == nil || p.tracer == nil {
		return nil
	}
	return p.tracer.GetSpans()
}
