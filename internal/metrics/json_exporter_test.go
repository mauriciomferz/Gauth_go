package metrics

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewJSONMetricsExporter(t *testing.T) {
	exporter := NewJSONMetricsExporter("gauth-test", "1.0.0", "test-host")

	if exporter == nil {
		t.Fatal("Expected non-nil exporter")
	}

	if exporter.metadata.ServiceName != "gauth-test" {
		t.Errorf("Expected service name 'gauth-test', got '%s'", exporter.metadata.ServiceName)
	}

	if exporter.metadata.ServiceVersion != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", exporter.metadata.ServiceVersion)
	}

	if exporter.metadata.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", exporter.metadata.Hostname)
	}

	if !exporter.includeReasons {
		t.Error("Expected includeReasons to be true by default")
	}

	if !exporter.includeMetadata {
		t.Error("Expected includeMetadata to be true by default")
	}
}

func TestRecordCounter(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Record counter without labels
	exporter.RecordCounter("requests_total", 5, nil)
	exporter.RecordCounter("requests_total", 3, nil)

	if exporter.counters["requests_total"] != 8 {
		t.Errorf("Expected counter value 8, got %d", exporter.counters["requests_total"])
	}

	// Record counter with labels
	labels := map[string]string{"method": "GET", "status": "200"}
	exporter.RecordCounter("http_requests", 10, labels)

	key := exporter.metricKey("http_requests", labels)
	if exporter.counters[key] != 10 {
		t.Errorf("Expected counter value 10, got %d", exporter.counters[key])
	}
}

func TestRecordGauge(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Record gauge without labels
	exporter.RecordGauge("cpu_usage", 45.5, nil)
	exporter.RecordGauge("cpu_usage", 50.2, nil)

	if exporter.gauges["cpu_usage"] != 50.2 {
		t.Errorf("Expected gauge value 50.2, got %f", exporter.gauges["cpu_usage"])
	}

	// Record gauge with labels
	labels := map[string]string{"core": "0"}
	exporter.RecordGauge("cpu_usage_per_core", 75.3, labels)

	key := exporter.metricKey("cpu_usage_per_core", labels)
	if exporter.gauges[key] != 75.3 {
		t.Errorf("Expected gauge value 75.3, got %f", exporter.gauges[key])
	}
}

func TestRecordHistogram(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Record histogram observations
	exporter.RecordHistogram("response_time", 0.05, nil)
	exporter.RecordHistogram("response_time", 0.15, nil)
	exporter.RecordHistogram("response_time", 0.25, nil)

	hist := exporter.histograms["response_time"]
	if hist == nil {
		t.Fatal("Expected histogram to be created")
	}

	if hist.Count != 3 {
		t.Errorf("Expected count 3, got %d", hist.Count)
	}

	expectedSum := 0.05 + 0.15 + 0.25
	if hist.Sum < expectedSum-0.001 || hist.Sum > expectedSum+0.001 {
		t.Errorf("Expected sum ~%.2f, got %.2f", expectedSum, hist.Sum)
	}

	if hist.Min != 0.05 {
		t.Errorf("Expected min 0.05, got %f", hist.Min)
	}

	if hist.Max != 0.25 {
		t.Errorf("Expected max 0.25, got %f", hist.Max)
	}
}

func TestExportJSON(t *testing.T) {
	exporter := NewJSONMetricsExporter("gauth", "1.0.0", "localhost")

	// Record some metrics
	exporter.RecordCounter("requests_total", 100, nil)
	exporter.RecordGauge("active_connections", 42.0, nil)
	exporter.RecordHistogram("latency", 0.1, map[string]string{"endpoint": "/auth"})
	exporter.RecordHistogram("latency", 0.2, map[string]string{"endpoint": "/auth"})

	// Export JSON
	data, err := exporter.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	// Parse JSON
	var response JSONMetricsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify metadata
	if response.Metadata.ServiceName != "gauth" {
		t.Errorf("Expected service name 'gauth', got '%s'", response.Metadata.ServiceName)
	}

	// Verify metrics count
	if len(response.Metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(response.Metrics))
	}

	// Verify reason taxonomy is included
	if len(response.Reasons) == 0 {
		t.Error("Expected reason taxonomy to be included")
	}

	// Verify JSON is well-formed (indented)
	if !strings.Contains(string(data), "\n") {
		t.Error("Expected indented JSON output")
	}
}

func TestExportJSONCompact(t *testing.T) {
	exporter := NewJSONMetricsExporter("gauth", "1.0.0", "localhost")

	exporter.RecordCounter("test_counter", 1, nil)

	data, err := exporter.ExportJSONCompact()
	if err != nil {
		t.Fatalf("ExportJSONCompact failed: %v", err)
	}

	// Verify it's valid JSON
	var response JSONMetricsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("Failed to parse compact JSON: %v", err)
	}

	// Compact JSON should have fewer newlines than indented
	if strings.Count(string(data), "\n") > 5 {
		t.Error("Expected compact JSON with minimal newlines")
	}
}

func TestSetIncludeReasons(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Disable reasons
	exporter.SetIncludeReasons(false)
	exporter.RecordCounter("test", 1, nil)

	data, err := exporter.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var response JSONMetricsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if response.Reasons != nil {
		t.Error("Expected no reason taxonomy when disabled")
	}

	// Enable reasons
	exporter.SetIncludeReasons(true)
	data, err = exporter.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(response.Reasons) == 0 {
		t.Error("Expected reason taxonomy when enabled")
	}
}

func TestReset(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Record some metrics
	exporter.RecordCounter("counter1", 10, nil)
	exporter.RecordGauge("gauge1", 20.5, nil)
	exporter.RecordHistogram("hist1", 0.5, nil)

	if exporter.GetMetricCount() != 3 {
		t.Errorf("Expected 3 metrics before reset, got %d", exporter.GetMetricCount())
	}

	// Reset
	exporter.Reset()

	if exporter.GetMetricCount() != 0 {
		t.Errorf("Expected 0 metrics after reset, got %d", exporter.GetMetricCount())
	}
}

func TestGetMetricCount(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	if exporter.GetMetricCount() != 0 {
		t.Errorf("Expected 0 metrics initially, got %d", exporter.GetMetricCount())
	}

	exporter.RecordCounter("c1", 1, nil)
	if exporter.GetMetricCount() != 1 {
		t.Errorf("Expected 1 metric, got %d", exporter.GetMetricCount())
	}

	exporter.RecordGauge("g1", 1.0, nil)
	if exporter.GetMetricCount() != 2 {
		t.Errorf("Expected 2 metrics, got %d", exporter.GetMetricCount())
	}

	exporter.RecordHistogram("h1", 1.0, nil)
	if exporter.GetMetricCount() != 3 {
		t.Errorf("Expected 3 metrics, got %d", exporter.GetMetricCount())
	}
}

func TestMetricKeyWithLabels(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	labels1 := map[string]string{"method": "GET", "status": "200"}
	labels2 := map[string]string{"method": "POST", "status": "201"}

	key1 := exporter.metricKey("http_requests", labels1)
	key2 := exporter.metricKey("http_requests", labels2)

	if key1 == key2 {
		t.Error("Expected different keys for different labels")
	}

	// Same labels should produce same key
	key1b := exporter.metricKey("http_requests", labels1)
	if key1 != key1b {
		t.Error("Expected same key for same labels")
	}
}

func TestReasonTaxonomy(t *testing.T) {
	taxonomy := initializeReasonTaxonomy()

	expectedReasons := []string{
		"policy_allow",
		"policy_deny",
		"scope_violation",
		"temporal_violation",
		"delegation_exceeded",
		"signature_invalid",
		"replay_detected",
		"revocation",
		"capability_insufficient",
		"jurisdiction_violation",
	}

	for _, reason := range expectedReasons {
		if _, exists := taxonomy[reason]; !exists {
			t.Errorf("Expected reason '%s' in taxonomy", reason)
		}
	}

	// Verify reason structure
	policyAllow := taxonomy["policy_allow"]
	if policyAllow.Category != "allow" {
		t.Errorf("Expected category 'allow', got '%s'", policyAllow.Category)
	}
	if policyAllow.Description == "" {
		t.Error("Expected non-empty description")
	}
	if len(policyAllow.Examples) == 0 {
		t.Error("Expected examples for policy_allow")
	}
}

func TestHistogramBuckets(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Record values in different buckets
	exporter.RecordHistogram("latency", 0.005, nil) // Should be in 0.01 bucket
	exporter.RecordHistogram("latency", 0.05, nil)  // Should be in 0.1 bucket
	exporter.RecordHistogram("latency", 5.0, nil)   // Should be in 10 bucket
	exporter.RecordHistogram("latency", 500.0, nil) // Should be in 1000 bucket

	hist := exporter.histograms["latency"]
	if hist == nil {
		t.Fatal("Expected histogram to be created")
	}

	// Check +Inf bucket has all observations
	if hist.Buckets["+Inf"] != 4 {
		t.Errorf("Expected +Inf bucket to have 4, got %d", hist.Buckets["+Inf"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				exporter.RecordCounter("concurrent_counter", 1, nil)
				exporter.RecordGauge("concurrent_gauge", float64(j), nil)
				exporter.RecordHistogram("concurrent_hist", float64(j)*0.01, nil)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counter accumulated correctly
	if exporter.counters["concurrent_counter"] != 1000 {
		t.Errorf("Expected counter 1000, got %d", exporter.counters["concurrent_counter"])
	}

	// Verify we can export without panic
	_, err := exporter.ExportJSON()
	if err != nil {
		t.Errorf("Concurrent export failed: %v", err)
	}
}

func TestJSONMetricsResponseStructure(t *testing.T) {
	exporter := NewJSONMetricsExporter("gauth", "1.0.0", "localhost")

	// Record various metric types
	exporter.RecordCounter("auth_decisions_total", 1000, map[string]string{"result": "allow"})
	exporter.RecordCounter("auth_decisions_total", 50, map[string]string{"result": "deny"})
	exporter.RecordGauge("active_sessions", 42.0, nil)
	exporter.RecordHistogram("decision_latency_seconds", 0.05, nil)
	exporter.RecordHistogram("decision_latency_seconds", 0.15, nil)

	data, err := exporter.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var response JSONMetricsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify metadata
	if response.Metadata.ServiceName == "" {
		t.Error("Expected non-empty service name")
	}
	if response.Metadata.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Verify metrics
	if len(response.Metrics) < 3 {
		t.Errorf("Expected at least 3 metrics, got %d", len(response.Metrics))
	}

	// Find counter metric with labels
	foundLabeledCounter := false
	for _, metric := range response.Metrics {
		if metric.Type == "counter" && metric.Labels != nil && len(metric.Labels) > 0 {
			foundLabeledCounter = true
			if metric.Labels["result"] == "" {
				t.Error("Expected 'result' label in counter")
			}
		}
	}
	if !foundLabeledCounter {
		t.Error("Expected to find labeled counter metric")
	}

	// Verify reason taxonomy
	if len(response.Reasons) < 5 {
		t.Errorf("Expected at least 5 reason categories, got %d", len(response.Reasons))
	}
}

func TestMetricsTimestamp(t *testing.T) {
	exporter := NewJSONMetricsExporter("test", "1.0", "host")

	before := time.Now()
	exporter.RecordCounter("test", 1, nil)

	data, _ := exporter.ExportJSON()
	var response JSONMetricsResponse
	_ = json.Unmarshal(data, &response)

	after := time.Now()

	// Verify timestamp is between before and after
	if response.Metadata.Timestamp.Before(before) || response.Metadata.Timestamp.After(after) {
		t.Error("Timestamp not in expected range")
	}

	// Verify all metrics have timestamps
	for _, metric := range response.Metrics {
		if metric.Timestamp.IsZero() {
			t.Errorf("Metric %s missing timestamp", metric.Name)
		}
	}
}
