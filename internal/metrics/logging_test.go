package metrics

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"
)

func TestLoggingMetrics(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	m := NewLoggingMetrics(logger)

	// Test RecordDecision
	m.RecordDecision("auth", "resource-1", "allow", 10*time.Millisecond)
	output := buf.String()
	if !strings.Contains(output, "Decision:") {
		t.Errorf("Expected log containing 'Decision:', got %q", output)
	}
	if !strings.Contains(output, "allow") {
		t.Errorf("Expected log containing 'allow', got %q", output)
	}
	buf.Reset()

	// Test IncUnauthorized
	m.IncUnauthorized()
	output = buf.String()
	if !strings.Contains(output, "Unauthorized access attempt") {
		t.Errorf("Expected unauthorized log, got %q", output)
	}
	buf.Reset()

	// Test IncSignatureVerificationFailures
	m.IncSignatureVerificationFailures()
	output = buf.String()
	if !strings.Contains(output, "Signature verification failed") {
		t.Errorf("Expected signature failure log, got %q", output)
	}
}
