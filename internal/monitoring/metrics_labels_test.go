package monitoring

import "testing"

// TestMetricLabelsPresence ensures that metrics expose a non-nil Labels map with the placeholder 'type' key.
func TestMetricLabelsPresence(t *testing.T) {
	mc := NewMetricsCollector()
	mc.Counter("requests_total", 1, nil)
	mc.Gauge("inflight", 3, nil)

	metrics := mc.GetAllMetrics()
	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}
	for name, mv := range metrics {
		if mv.Labels == nil {
			t.Fatalf("metric %s has nil labels", name)
		}
		if _, ok := mv.Labels["type"]; !ok {
			t.Fatalf("metric %s missing 'type' label placeholder", name)
		}
	}
}
