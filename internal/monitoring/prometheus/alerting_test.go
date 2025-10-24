package prometheus

import (
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/monitoring"
)

func TestPrometheusExporter_Alerting(t *testing.T) {
	mc := monitoring.NewDefaultMetricsCollector()
	mc.Counter("errors_total", 1, nil)
	mc.Gauge("active_sessions", 5, nil)
	mc.Counter("requests_total", 10, nil)
	// Simulate error spike
	for i := 0; i < 10; i++ {
		mc.Counter("errors_total", 1, nil)
	}
	metrics := mc.GetAllMetrics()
	if metrics["errors_total"].Value < 10 {
		t.Fatalf("expected error count >= 10, got %f", metrics["errors_total"].Value)
	}
	// Simulate alerting rule: error rate > threshold
	if metrics["errors_total"].Value/metrics["requests_total"].Value > 0.5 {
		// In real system, trigger Prometheus alert
		t.Logf("ALERT: error rate exceeds threshold")
	}
	// Simulate session drop
	mc.Gauge("active_sessions", 0, nil)
	if metrics["active_sessions"].Value == 0 {
		t.Logf("ALERT: all sessions dropped")
	}
	// Simulate latency spike
	mc.Gauge("response_time_ms", 5000, nil)
	if metrics["response_time_ms"].Value > 1000 {
		t.Logf("ALERT: response time exceeds threshold")
	}
}
