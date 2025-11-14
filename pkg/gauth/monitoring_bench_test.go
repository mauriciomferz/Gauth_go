// Package gauth - Monitoring Service Benchmarks
// Task 8: Performance testing and optimization for monitoring service
package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkMetricsRecording measures metric recording performance
func BenchmarkMetricsRecording(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.RecordValidation("DE", true, 10*time.Millisecond, "comprehensive")
	}
}

// BenchmarkConcurrentMetricsRecording measures concurrent metric recording
func BenchmarkConcurrentMetricsRecording(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			service.RecordValidation("DE", true, 10*time.Millisecond, "comprehensive")
		}
	})
}

// BenchmarkComplianceViolationRecording measures compliance violation recording
func BenchmarkComplianceViolationRecording(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	details := map[string]interface{}{
		"rule_id": "RULE-001",
		"value":   50000.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.RecordComplianceViolation(
			AlertTypeJurisdictionViolation,
			AlertSeverityMedium,
			"DE",
			details,
		)
	}
}

// BenchmarkHealthCheckExecution measures health check execution
func BenchmarkHealthCheckExecution(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)
	ctx := context.Background()

	// Register mock health check
	service.RegisterHealthCheck("test-component", func(ctx context.Context) HealthCheckResult {
		return HealthCheckResult{
			Status:  HealthStatusHealthy,
			Message: "OK",
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = service.RunHealthChecks(ctx)
	}
}

// BenchmarkDashboardMetricsCollection measures dashboard metrics collection
func BenchmarkDashboardMetricsCollection(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	// Simulate some activity
	for i := 0; i < 100; i++ {
		service.RecordValidation("DE", true, 10*time.Millisecond, "comprehensive")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = service.GetDashboardMetrics()
	}
}

// BenchmarkSystemResourceUpdate measures resource monitoring
func BenchmarkSystemResourceUpdate(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.UpdateSystemResources(45.2, 1024.5, 150, 25, 0.85)
	}
}

// BenchmarkMultiJurisdictionMetrics measures metrics across multiple jurisdictions
func BenchmarkMultiJurisdictionMetrics(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	jurisdictions := []string{"DE", "FR", "IT", "ES", "PT", "NL", "BE", "LU", "AT"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jurisdiction := jurisdictions[i%len(jurisdictions)]
		service.RecordValidation(jurisdiction, true, 10*time.Millisecond, "comprehensive")
	}
}

// BenchmarkDashboardMetricsAccess measures dashboard metrics access
func BenchmarkDashboardMetricsAccess(b *testing.B) {
	config := MonitoringConfig{
		HealthCheckInterval:     10 * time.Second,
		ComplianceCheckInterval: 30 * time.Second,
	}
	service := NewMonitoringService(config)

	// Record various metrics
	for i := 0; i < 50; i++ {
		service.RecordValidation("DE", i%10 != 0, 100*time.Millisecond, "comprehensive")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics := service.GetDashboardMetrics()
		_ = metrics.TotalValidations
		_ = metrics.ErrorRate
		_ = metrics.ValidationLatencyP95
	}
}

// BenchmarkHistogramObservation measures histogram observation performance
func BenchmarkHistogramObservation(b *testing.B) {
	registry := prometheus.NewRegistry()
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_duration_seconds",
			Help:    "Test duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"jurisdiction", "type"},
	)
	registry.MustRegister(histogram)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		histogram.WithLabelValues("DE", "comprehensive").Observe(0.010)
	}
}

// BenchmarkCounterIncrement measures counter increment performance
func BenchmarkCounterIncrement(b *testing.B) {
	registry := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_total",
			Help: "Test total count",
		},
		[]string{"jurisdiction", "status"},
	)
	registry.MustRegister(counter)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		counter.WithLabelValues("DE", "success").Inc()
	}
}

// BenchmarkGaugeSet measures gauge set performance
func BenchmarkGaugeSet(b *testing.B) {
	registry := prometheus.NewRegistry()
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_value",
			Help: "Test gauge value",
		},
		[]string{"component"},
	)
	registry.MustRegister(gauge)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gauge.WithLabelValues("test").Set(42.5)
	}
}
