package tracing

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestW3CTracePropagation(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	// Start a span
	span, ctx := tracer.StartSpan(ctx, "test-operation")
	span.TraceID = "1234567890abcdef1234567890abcdef"
	span.SpanID = "1234567890abcdef"

	// Inject into headers
	headers := http.Header{}
	InjectW3C(ctx, headers)

	// Verify traceparent format
	traceparent := headers.Get(TraceparentHeader)
	if traceparent == "" {
		t.Fatal("Expected traceparent header to be set")
	}

	t.Logf("Traceparent: %s", traceparent)

	// Extract from headers
	traceID, spanID := ExtractW3C(headers)
	if traceID == "" || spanID == "" {
		t.Errorf("Failed to extract trace context: traceID=%s, spanID=%s", traceID, spanID)
	}

	// Verify trace ID is preserved (may be padded)
	if !contains(traceID, "1234567890abcdef") {
		t.Errorf("Trace ID not preserved: got %s", traceID)
	}
}

func TestStartSpanFromW3C(t *testing.T) {
	tracer := NewTracer("test-service")

	// Create headers with trace context
	headers := http.Header{}
	headers.Set(TraceparentHeader, "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")

	// Start span from W3C context
	span, _ := tracer.StartSpanFromW3C(context.Background(), "incoming-request", headers)

	if span.TraceID != "0123456789abcdef0123456789abcdef" {
		t.Errorf("Expected trace ID to be extracted, got %s", span.TraceID)
	}

	if span.ParentID != "0123456789abcdef" {
		t.Errorf("Expected parent span ID to be extracted, got %s", span.ParentID)
	}
}

func TestExtractW3C_InvalidFormat(t *testing.T) {
	tests := []struct {
		name        string
		traceparent string
	}{
		{"empty", ""},
		{"wrong parts", "00-abc-def"},
		{"wrong version", "01-0123456789abcdef0123456789abcdef-0123456789abcdef-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			if tt.traceparent != "" {
				headers.Set(TraceparentHeader, tt.traceparent)
			}

			traceID, spanID := ExtractW3C(headers)
			if traceID != "" || spanID != "" {
				t.Errorf("Expected empty extraction for invalid format, got traceID=%s, spanID=%s", traceID, spanID)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && strings.Contains(s, substr)
}
