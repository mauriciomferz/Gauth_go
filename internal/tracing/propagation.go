package tracing

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// W3C Trace Context propagation for distributed tracing (sec7.item4)
// Implements basic traceparent header injection/extraction per https://www.w3.org/TR/trace-context/

const (
	// TraceparentHeader is the W3C Trace Context header name
	TraceparentHeader = "traceparent"
	// TracestateHeader is the W3C Trace Context state header
	TracestateHeader = "tracestate"
)

// InjectW3C injects the current span's trace context into HTTP headers
// Format: traceparent: 00-{trace-id}-{span-id}-{flags}
func InjectW3C(ctx context.Context, headers http.Header) {
	span := SpanFromContext(ctx)
	if span == nil {
		return
	}

	// W3C traceparent format: version-traceid-spanid-flags
	// version: 00 (current spec)
	// traceid: 32 hex chars (we'll pad/truncate our trace IDs)
	// spanid: 16 hex chars (we'll pad/truncate our span IDs)
	// flags: 01 for sampled, 00 for not sampled
	traceparent := fmt.Sprintf("00-%s-%s-01",
		padOrTruncateTraceID(span.TraceID),
		padOrTruncateSpanID(span.SpanID))

	headers.Set(TraceparentHeader, traceparent)
}

// ExtractW3C extracts trace context from HTTP headers and returns a parent span ID and trace ID
// Returns empty strings if no valid traceparent header found
func ExtractW3C(headers http.Header) (traceID, spanID string) {
	traceparent := headers.Get(TraceparentHeader)
	if traceparent == "" {
		return "", ""
	}

	// Parse format: 00-{trace-id}-{span-id}-{flags}
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return "", ""
	}

	version := parts[0]
	if version != "00" {
		// Unknown version, skip
		return "", ""
	}

	return parts[1], parts[2]
}

// StartSpanFromW3C creates a new span using trace context from HTTP headers
func (t *Tracer) StartSpanFromW3C(ctx context.Context, operation string, headers http.Header) (*Span, context.Context) {
	traceID, parentSpanID := ExtractW3C(headers)

	span := &Span{
		TraceID:   traceID,
		SpanID:    fmt.Sprintf("span-%d", time.Now().UnixNano()),
		ParentID:  parentSpanID,
		Operation: operation,
		StartTime: time.Now(),
		Tags:      make(map[string]interface{}),
		Status:    "ok",
	}

	// If no trace context in headers, generate new trace ID
	if span.TraceID == "" {
		span.TraceID = fmt.Sprintf("trace-%d", time.Now().UnixNano())
		span.ParentID = ""
	}

	t.spans = append(t.spans, span)
	ctx = ContextWithSpan(ctx, span)
	return span, ctx
}

// Helper functions to pad/truncate IDs to W3C format

func padOrTruncateTraceID(id string) string {
	// W3C requires 32 hex chars for trace-id
	cleaned := strings.TrimPrefix(id, "trace-")
	if len(cleaned) > 32 {
		return cleaned[:32]
	}
	return fmt.Sprintf("%032s", cleaned)
}

func padOrTruncateSpanID(id string) string {
	// W3C requires 16 hex chars for span-id
	cleaned := strings.TrimPrefix(id, "span-")
	if len(cleaned) > 16 {
		return cleaned[:16]
	}
	return fmt.Sprintf("%016s", cleaned)
}

// PropagationMiddleware is HTTP middleware that extracts incoming trace context
// and injects it into outgoing responses
func PropagationMiddleware(next http.Handler, tracer *Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming request
		span, ctx := tracer.StartSpanFromW3C(r.Context(), r.URL.Path, r.Header)
		span.SetTag("http.method", r.Method)
		span.SetTag("http.url", r.URL.String())
		span.SetTag("http.remote_addr", r.RemoteAddr)

		// Create response writer wrapper to inject headers
		wrapped := &responseWriter{ResponseWriter: w, headers: w.Header()}

		// Inject trace context into outgoing response
		InjectW3C(ctx, wrapped.headers)

		// Call next handler with context containing span
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Finish span after handler completes
		tracer.FinishSpan(span)
		span.SetTag("http.status_code", wrapped.statusCode)
	})
}

// GinTracingMiddleware is Gin-compatible middleware for trace propagation
func GinTracingMiddleware(tracer *Tracer, sampleRatio float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Probabilistic sampling
		if sampleRatio > 0 && sampleRatio < 1.0 {
			// #nosec G404
			if rand.Float64() > sampleRatio {
				c.Next()
				return
			}
		}

		// Extract trace context from incoming request headers
		span, ctx := tracer.StartSpanFromW3C(c.Request.Context(), c.Request.URL.Path, c.Request.Header)
		span.SetTag("http.method", c.Request.Method)
		span.SetTag("http.url", c.Request.URL.String())
		span.SetTag("http.remote_addr", c.ClientIP())

		// Update request context with span
		c.Request = c.Request.WithContext(ctx)

		// Inject trace context into outgoing response headers (W3C)
		InjectW3C(ctx, c.Writer.Header())

		// Process request
		c.Next()

		// Finalize span
		tracer.FinishSpan(span)
		span.SetTag("http.status_code", c.Writer.Status())
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	headers    http.Header
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = 200
	}
	return rw.ResponseWriter.Write(b)
}
