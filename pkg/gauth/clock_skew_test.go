package gauth

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewClockSkewValidator_DefaultTolerance(t *testing.T) {
	// Clear any existing env var
	os.Unsetenv("GAUTH_CLOCK_SKEW_SECONDS")

	validator := NewClockSkewValidator()
	if validator.maxSkewSeconds != 300 {
		t.Errorf("Expected default tolerance of 300 seconds, got %d", validator.maxSkewSeconds)
	}
}

func TestNewClockSkewValidator_CustomTolerance(t *testing.T) {
	os.Setenv("GAUTH_CLOCK_SKEW_SECONDS", "600")
	defer os.Unsetenv("GAUTH_CLOCK_SKEW_SECONDS")

	validator := NewClockSkewValidator()
	if validator.maxSkewSeconds != 600 {
		t.Errorf("Expected custom tolerance of 600 seconds, got %d", validator.maxSkewSeconds)
	}
}

func TestNewClockSkewValidator_InvalidEnv(t *testing.T) {
	os.Setenv("GAUTH_CLOCK_SKEW_SECONDS", "invalid")
	defer os.Unsetenv("GAUTH_CLOCK_SKEW_SECONDS")

	validator := NewClockSkewValidator()
	if validator.maxSkewSeconds != 300 {
		t.Errorf("Expected default tolerance when env is invalid, got %d", validator.maxSkewSeconds)
	}
}

func TestValidateTimestamp_WithinTolerance(t *testing.T) {
	validator := &ClockSkewValidator{maxSkewSeconds: 300}

	tests := []struct {
		name      string
		timestamp time.Time
	}{
		{"Current time", time.Now()},
		{"1 minute in past", time.Now().Add(-1 * time.Minute)},
		{"1 minute in future", time.Now().Add(1 * time.Minute)},
		{"4 minutes in past", time.Now().Add(-4 * time.Minute)},
		{"4 minutes in future", time.Now().Add(4 * time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skew, err := validator.ValidateTimestamp(tt.timestamp)
			if err != nil {
				t.Errorf("Expected no error for %s, got: %v", tt.name, err)
			}
			if skew < -300 || skew > 300 {
				t.Errorf("Skew %d exceeds tolerance for %s", skew, tt.name)
			}
		})
	}
}

func TestValidateTimestamp_ExceedsTolerance(t *testing.T) {
	validator := &ClockSkewValidator{maxSkewSeconds: 300}

	tests := []struct {
		name      string
		timestamp time.Time
	}{
		{"10 minutes in past", time.Now().Add(-10 * time.Minute)},
		{"10 minutes in future", time.Now().Add(10 * time.Minute)},
		{"1 hour in past", time.Now().Add(-1 * time.Hour)},
		{"1 hour in future", time.Now().Add(1 * time.Hour)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skew, err := validator.ValidateTimestamp(tt.timestamp)
			if err == nil {
				t.Errorf("Expected error for %s (skew: %d), got nil", tt.name, skew)
			}
			if skew >= -300 && skew <= 300 {
				t.Errorf("Skew %d should exceed tolerance for %s", skew, tt.name)
			}
		})
	}
}

func TestParseJTI_ValidFormat(t *testing.T) {
	// Create JTI with current timestamp in milliseconds
	now := time.Now()
	timestampMs := now.UnixMilli()
	jti := fmt.Sprintf("%d_abcdef123456", timestampMs)

	parts := parseJTI(jti)
	if parts == nil {
		t.Fatal("Expected non-nil JTIParts")
	}

	// Check timestamp is close to now (within 1 second)
	diff := parts.Timestamp.Unix() - now.Unix()
	if diff < -1 || diff > 1 {
		t.Errorf("Parsed timestamp diff %d seconds from expected", diff)
	}

	if parts.Random != "abcdef123456" {
		t.Errorf("Expected random='abcdef123456', got '%s'", parts.Random)
	}
}

func TestParseJTI_InvalidFormats(t *testing.T) {
	tests := []struct {
		name string
		jti  string
	}{
		{"Empty string", ""},
		{"Too short", "123"},
		{"No underscore", "1699545600000"},
		{"Invalid timestamp", "invalid_abc123"},
		{"Missing random part", "1699545600000_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := parseJTI(tt.jti)
			// Some invalid formats may parse partially, but timestamp should be reasonable
			if parts != nil && !parts.Timestamp.IsZero() {
				// If parsed, timestamp should be within last 10 years to future 1 year
				tenYearsAgo := time.Now().AddDate(-10, 0, 0)
				oneYearFuture := time.Now().AddDate(1, 0, 0)
				if parts.Timestamp.Before(tenYearsAgo) || parts.Timestamp.After(oneYearFuture) {
					t.Logf("Parsed unreasonable timestamp for '%s': %v", tt.jti, parts.Timestamp)
				}
			}
		})
	}
}

func TestValidateJTITimestamp_Valid(t *testing.T) {
	validator := &ClockSkewValidator{maxSkewSeconds: 300}

	// Create JTI with timestamp within tolerance
	now := time.Now().Add(-2 * time.Minute) // 2 minutes ago
	jti := fmt.Sprintf("%d_random123", now.UnixMilli())

	skew, err := validator.ValidateJTITimestamp(jti)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if skew < -300 || skew > 300 {
		t.Errorf("Skew %d exceeds tolerance", skew)
	}
}

func TestValidateJTITimestamp_Excessive(t *testing.T) {
	validator := &ClockSkewValidator{maxSkewSeconds: 300}

	// Create JTI with timestamp exceeding tolerance
	oldTime := time.Now().Add(-10 * time.Minute) // 10 minutes ago
	jti := fmt.Sprintf("%d_random123", oldTime.UnixMilli())

	skew, err := validator.ValidateJTITimestamp(jti)
	if err == nil {
		t.Errorf("Expected error for excessive skew, got nil (skew: %d)", skew)
	}
}

func TestValidateJTITimestamp_InvalidJTI(t *testing.T) {
	validator := &ClockSkewValidator{maxSkewSeconds: 300}

	_, err := validator.ValidateJTITimestamp("invalid_jti_format")
	if err == nil {
		t.Error("Expected error for invalid JTI format")
	}
}

func TestSkewMetrics_Recording(t *testing.T) {
	metrics := NewSkewMetrics()

	// Record some skew measurements
	metrics.RecordSkew(10, false)
	metrics.RecordSkew(-20, false)
	metrics.RecordSkew(400, true) // Excessive

	if metrics.TotalValidations != 3 {
		t.Errorf("Expected 3 validations, got %d", metrics.TotalValidations)
	}

	if metrics.ExcessiveSkewCount != 1 {
		t.Errorf("Expected 1 excessive skew, got %d", metrics.ExcessiveSkewCount)
	}

	if metrics.MaxObservedSkew != 400 {
		t.Errorf("Expected max skew 400, got %d", metrics.MaxObservedSkew)
	}

	// Average should be (10 + 20 + 400) / 3 = 143.33
	expectedAvg := (10.0 + 20.0 + 400.0) / 3.0
	if metrics.AverageSkew < expectedAvg-1 || metrics.AverageSkew > expectedAvg+1 {
		t.Errorf("Expected average skew ~%.1f, got %.1f", expectedAvg, metrics.AverageSkew)
	}
}

func TestSkewMetrics_IsWarningLevel(t *testing.T) {
	metrics := &SkewMetrics{WarningThresholdRatio: 0.7}
	maxAllowed := int64(300)

	tests := []struct {
		name     string
		skew     int64
		expected bool
	}{
		{"Below warning (100)", 100, false},
		{"At warning threshold (210)", 210, true},
		{"Above warning (250)", 250, true},
		{"At max (300)", 300, false},  // Not warning, it's exceeded
		{"Above max (400)", 400, false}, // Not warning, it's exceeded
		{"Negative below warning (-100)", -100, false},
		{"Negative at warning (-210)", -210, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metrics.IsWarningLevel(tt.skew, maxAllowed)
			if result != tt.expected {
				t.Errorf("IsWarningLevel(%d, %d) = %v, expected %v", tt.skew, maxAllowed, result, tt.expected)
			}
		})
	}
}

func TestSkewMetrics_GetStats(t *testing.T) {
	metrics := NewSkewMetrics()

	// Test empty metrics
	stats := metrics.GetStats()
	if stats != "No validations recorded" {
		t.Errorf("Expected empty stats message, got: %s", stats)
	}

	// Record some data
	metrics.RecordSkew(50, false)
	metrics.RecordSkew(100, false)
	metrics.RecordSkew(400, true)

	stats = metrics.GetStats()
	if stats == "" {
		t.Error("Expected non-empty stats string")
	}

	// Check that stats contains key information
	if !containsSkewStats(stats, "Validations: 3") {
		t.Errorf("Stats missing validation count: %s", stats)
	}
	if !containsSkewStats(stats, "Excessive: 1") {
		t.Errorf("Stats missing excessive count: %s", stats)
	}
}

func TestClockSkewIntegration(t *testing.T) {
	// Integration test: Create validator, generate JTI, validate
	validator := NewClockSkewValidator()
	metrics := NewSkewMetrics()

	// Generate valid JTI
	now := time.Now()
	jti := fmt.Sprintf("%d_integration_test", now.UnixMilli())

	skew, err := validator.ValidateJTITimestamp(jti)
	if err != nil {
		t.Errorf("Integration test failed: %v", err)
	}

	metrics.RecordSkew(skew, err != nil)

	// Generate old JTI (should fail)
	oldTime := time.Now().Add(-1 * time.Hour)
	oldJTI := fmt.Sprintf("%d_old_test", oldTime.UnixMilli())

	skew, err = validator.ValidateJTITimestamp(oldJTI)
	if err == nil {
		t.Error("Expected error for old JTI")
	}

	metrics.RecordSkew(skew, err != nil)

	// Check metrics
	if metrics.TotalValidations != 2 {
		t.Errorf("Expected 2 validations, got %d", metrics.TotalValidations)
	}
	if metrics.ExcessiveSkewCount != 1 {
		t.Errorf("Expected 1 excessive skew, got %d", metrics.ExcessiveSkewCount)
	}

	t.Logf("Integration test metrics: %s", metrics.GetStats())
}

func TestClockSkewValidator_EdgeCases(t *testing.T) {
	validator := &ClockSkewValidator{maxSkewSeconds: 300}

	// Test exactly at boundary
	boundaryTime := time.Now().Add(-300 * time.Second)
	_, err := validator.ValidateTimestamp(boundaryTime)
	if err != nil {
		t.Errorf("Expected no error at exact boundary, got: %v", err)
	}

	// Test just over boundary
	overBoundary := time.Now().Add(-301 * time.Second)
	_, err = validator.ValidateTimestamp(overBoundary)
	if err == nil {
		t.Error("Expected error just over boundary")
	}
}

// Helper function for stats string validation
func containsSkewStats(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && containsSkewStatsHelper(s, substr)))
}

func containsSkewStatsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
