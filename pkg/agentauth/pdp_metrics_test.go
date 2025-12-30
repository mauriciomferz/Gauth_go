// Package agentauth - PDP Metrics Integration Tests
package agentauth

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
)

// TestPDPMetricsIntegration_EnforcementMetrics verifies enforcement metrics are exported correctly
func TestPDPMetricsIntegration_EnforcementMetrics(t *testing.T) {
	// Create a dedicated registry for testing
	registry := prom.NewRegistry()
	promMetrics := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{
		Namespace: "test",
		Subsystem: "pdp",
		Registry:  registry,
	})

	// Create audit logger with metrics
	logger := NewProductionPEPAuditLogger(100, false, true)
	logger.SetMetrics(promMetrics)

	ctx := context.Background()

	// Log enforcement with allowed=true, action="read"
	err := logger.LogEnforcement(ctx, &EnforcementAuditEntry{
		EnforcementID:  "enf-001",
		Timestamp:      time.Now(),
		ActionType:     "read",
		ResourceID:     "resource-001",
		Allowed:        true,
		Outcome:        "success",
		Reason:         "policy_allows",
		ViolationCount: 0,
	})
	if err != nil {
		t.Fatalf("Failed to log enforcement: %v", err)
	}

	// Log enforcement with allowed=false, action="write"
	err = logger.LogEnforcement(ctx, &EnforcementAuditEntry{
		EnforcementID:  "enf-002",
		Timestamp:      time.Now(),
		ActionType:     "write",
		ResourceID:     "resource-002",
		Allowed:        false,
		Outcome:        "denied",
		Reason:         "policy_denies",
		ViolationCount: 1,
	})
	if err != nil {
		t.Fatalf("Failed to log enforcement: %v", err)
	}

	// Gather metrics and verify counters
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the pep_enforcements_total metric
	var enforcementsFound bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_pdp_pep_enforcements_total" {
			enforcementsFound = true
			metrics := mf.GetMetric()

			// We should have 2 label combinations: {allowed="true", action_type="read"} and {allowed="false", action_type="write"}
			if len(metrics) < 2 {
				t.Errorf("Expected at least 2 enforcement metrics, got %d", len(metrics))
			}

			for _, m := range metrics {
				labels := m.GetLabel()
				var allowed, actionType string
				for _, label := range labels {
					if label.GetName() == "allowed" {
						allowed = label.GetValue()
					}
					if label.GetName() == "action_type" {
						actionType = label.GetValue()
					}
				}

				value := m.GetCounter().GetValue()
				if allowed == "true" && actionType == "read" {
					if value != 1 {
						t.Errorf("Expected 1 enforcement for allowed=true, action_type=read, got %v", value)
					}
				} else if allowed == "false" && actionType == "write" {
					if value != 1 {
						t.Errorf("Expected 1 enforcement for allowed=false, action_type=write, got %v", value)
					}
				}
			}
		}
	}

	if !enforcementsFound {
		t.Error("pep_enforcements_total metric not found in registry")
	}
}

// TestPDPMetricsIntegration_ViolationMetrics verifies violation metrics are exported with severity labels
func TestPDPMetricsIntegration_ViolationMetrics(t *testing.T) {
	registry := prom.NewRegistry()
	promMetrics := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{
		Namespace: "test",
		Subsystem: "pdp",
		Registry:  registry,
	})

	logger := NewProductionPEPAuditLogger(100, false, true)
	logger.SetMetrics(promMetrics)

	ctx := context.Background()

	// Log critical violation
	err := logger.LogViolation(ctx, &ViolationAuditEntry{
		EnforcementID: "enf-001",
		Timestamp:     time.Now(),
		ViolationType: "policy_violation",
		Severity:      "critical",
		ActionType:    "delete",
		ResourceID:    "resource-001",
		Description:   "Unauthorized deletion attempt",
	})
	if err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Log high severity violation
	err = logger.LogViolation(ctx, &ViolationAuditEntry{
		EnforcementID: "enf-002",
		Timestamp:     time.Now(),
		ViolationType: "resource_access_denied",
		Severity:      "high",
		ActionType:    "read",
		ResourceID:    "resource-002",
		Description:   "Access to restricted resource",
	})
	if err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Log medium severity violation
	err = logger.LogViolation(ctx, &ViolationAuditEntry{
		EnforcementID: "enf-003",
		Timestamp:     time.Now(),
		ViolationType: "rate_limit_exceeded",
		Severity:      "medium",
		ActionType:    "write",
		ResourceID:    "resource-003",
		Description:   "Rate limit exceeded",
	})
	if err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Gather and verify metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var violationsFound bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_pdp_pep_violations_total" {
			violationsFound = true
			metrics := mf.GetMetric()

			if len(metrics) != 3 {
				t.Errorf("Expected 3 violation metrics, got %d", len(metrics))
			}

			for _, m := range metrics {
				labels := m.GetLabel()
				var violationType, severity string
				for _, label := range labels {
					if label.GetName() == "violation_type" {
						violationType = label.GetValue()
					}
					if label.GetName() == "severity" {
						severity = label.GetValue()
					}
				}

				value := m.GetCounter().GetValue()
				if value != 1 {
					t.Errorf("Expected 1 violation for %s/%s, got %v", violationType, severity, value)
				}

				// Verify severity labels are present
				if severity != "critical" && severity != "high" && severity != "medium" {
					t.Errorf("Unexpected severity label: %s", severity)
				}
			}
		}
	}

	if !violationsFound {
		t.Error("pep_violations_total metric not found in registry")
	}
}

// TestPDPMetricsIntegration_BufferSizeGauges verifies audit buffer size gauges are updated
func TestPDPMetricsIntegration_BufferSizeGauges(t *testing.T) {
	registry := prom.NewRegistry()
	promMetrics := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{
		Namespace: "test",
		Subsystem: "pdp",
		Registry:  registry,
	})

	logger := NewProductionPEPAuditLogger(100, false, true)
	logger.SetMetrics(promMetrics)

	ctx := context.Background()

	// Log 3 enforcements
	for i := 0; i < 3; i++ {
		err := logger.LogEnforcement(ctx, &EnforcementAuditEntry{
			EnforcementID:  "enf-" + string(rune(i)),
			Timestamp:      time.Now(),
			ActionType:     "read",
			ResourceID:     "resource-001",
			Allowed:        true,
			Outcome:        "success",
			ViolationCount: 0,
		})
		if err != nil {
			t.Fatalf("Failed to log enforcement: %v", err)
		}
	}

	// Log 2 violations
	for i := 0; i < 2; i++ {
		err := logger.LogViolation(ctx, &ViolationAuditEntry{
			EnforcementID: "enf-" + string(rune(i)),
			Timestamp:     time.Now(),
			ViolationType: "policy_violation",
			Severity:      "medium",
			ActionType:    "write",
			ResourceID:    "resource-001",
			Description:   "Test violation",
		})
		if err != nil {
			t.Fatalf("Failed to log violation: %v", err)
		}
	}

	// Gather and verify buffer size gauges
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var enforcementBufferFound, violationBufferFound bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_pdp_pep_audit_buffer_enforcements" {
			enforcementBufferFound = true
			value := mf.GetMetric()[0].GetGauge().GetValue()
			if value != 3 {
				t.Errorf("Expected enforcement buffer size 3, got %v", value)
			}
		}
		if mf.GetName() == "test_pdp_pep_audit_buffer_violations" {
			violationBufferFound = true
			value := mf.GetMetric()[0].GetGauge().GetValue()
			if value != 2 {
				t.Errorf("Expected violation buffer size 2, got %v", value)
			}
		}
	}

	if !enforcementBufferFound {
		t.Error("pep_audit_buffer_enforcements gauge not found")
	}
	if !violationBufferFound {
		t.Error("pep_audit_buffer_violations gauge not found")
	}
}

// TestPDPMetricsIntegration_NoopMetrics verifies noop metrics don't cause errors
func TestPDPMetricsIntegration_NoopMetrics(t *testing.T) {
	// Logger with metrics enabled but using noop metrics (default)
	logger := NewProductionPEPAuditLogger(100, false, true)

	ctx := context.Background()

	// Should not panic or error with noop metrics
	err := logger.LogEnforcement(ctx, &EnforcementAuditEntry{
		EnforcementID:  "enf-001",
		Timestamp:      time.Now(),
		ActionType:     "read",
		ResourceID:     "resource-001",
		Allowed:        true,
		Outcome:        "success",
		ViolationCount: 0,
	})
	if err != nil {
		t.Fatalf("Failed with noop metrics: %v", err)
	}

	err = logger.LogViolation(ctx, &ViolationAuditEntry{
		EnforcementID: "enf-001",
		Timestamp:     time.Now(),
		ViolationType: "policy_violation",
		Severity:      "low",
		ActionType:    "read",
		ResourceID:    "resource-001",
		Description:   "Test",
	})
	if err != nil {
		t.Fatalf("Failed with noop metrics: %v", err)
	}
}

// TestPDPMetricsIntegration_MetricsDisabled verifies metrics are not collected when disabled
func TestPDPMetricsIntegration_MetricsDisabled(t *testing.T) {
	registry := prom.NewRegistry()
	promMetrics := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{
		Namespace: "test",
		Subsystem: "pdp",
		Registry:  registry,
	})

	// Logger with metrics disabled
	logger := NewProductionPEPAuditLogger(100, false, false)
	logger.SetMetrics(promMetrics)

	ctx := context.Background()

	err := logger.LogEnforcement(ctx, &EnforcementAuditEntry{
		EnforcementID:  "enf-001",
		Timestamp:      time.Now(),
		ActionType:     "read",
		ResourceID:     "resource-001",
		Allowed:        true,
		Outcome:        "success",
		ViolationCount: 0,
	})
	if err != nil {
		t.Fatalf("Failed to log enforcement: %v", err)
	}

	// Metrics should not be incremented when metrics are disabled
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// The pep_enforcements_total counter should not exist or have value 0
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_pdp_pep_enforcements_total" {
			metrics := mf.GetMetric()
			for _, m := range metrics {
				value := m.GetCounter().GetValue()
				if value != 0 {
					t.Errorf("Expected 0 enforcements when metrics disabled, got %v", value)
				}
			}
		}
	}
}
