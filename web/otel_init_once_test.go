package web

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	metrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestOTELInitOnce ensures that multiple BetaServer constructions only initialize the OTEL exporter once.
func TestOTELInitOnce(t *testing.T) {
	t.Setenv("GAUTH_OTEL_METRICS_ENABLE", "1")

	// Capture stderr
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	// Construct two servers (minimal) then shutdown
	s1 := NewBetaServerWithMetrics(":0", metrics.NewMemory())
	s2 := NewBetaServerWithMetrics(":0", metrics.NewMemory())

	s1.Shutdown()
	s2.Shutdown()

	// Restore stderr
	w.Close()
	os.Stderr = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Count init lines
	wantMarker := "[otel-metrics] stdout exporter initialized"
	count := strings.Count(output, wantMarker)
	if count == 0 {
		t.Fatalf("expected at least 1 OTEL init line, got 0. stderr=\n%s", output)
	}
	if count > 1 {
		t.Fatalf("expected OTEL init guarded by sync.Once; saw %d init lines. stderr=\n%s", count, output)
	}
}
