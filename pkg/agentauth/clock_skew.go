package agentauth

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ClockSkewValidator validates token timestamps against clock skew tolerance
type ClockSkewValidator struct {
	maxSkewSeconds int64
}

// NewClockSkewValidator creates a new clock skew validator
// Reads tolerance from AGENTAUTH_CLOCK_SKEW_SECONDS environment variable (default: 300 seconds = 5 minutes)
func NewClockSkewValidator() *ClockSkewValidator {
	maxSkew := int64(300) // 5 minutes default
	if envSkew := os.Getenv("AGENTAUTH_CLOCK_SKEW_SECONDS"); envSkew != "" {
		if parsed, err := strconv.ParseInt(envSkew, 10, 64); err == nil && parsed > 0 {
			maxSkew = parsed
		}
	}
	return &ClockSkewValidator{
		maxSkewSeconds: maxSkew,
	}
}

// ValidateTimestamp checks if the given timestamp is within acceptable skew tolerance
// Returns skew in seconds (positive = future, negative = past) and error if excessive
func (v *ClockSkewValidator) ValidateTimestamp(ts time.Time) (int64, error) {
	now := time.Now()
	skew := ts.Unix() - now.Unix()
	absSkew := skew
	if absSkew < 0 {
		absSkew = -absSkew
	}

	if absSkew > v.maxSkewSeconds {
		return skew, fmt.Errorf("clock skew exceeds tolerance: %d seconds (max: %d)", absSkew, v.maxSkewSeconds)
	}

	return skew, nil
}

// ValidateJTITimestamp validates a JTI's embedded timestamp component
// JTI format: {base64(timestamp)}_{random_bytes}
// Returns skew in seconds and any validation error
func (v *ClockSkewValidator) ValidateJTITimestamp(jti string) (int64, error) {
	// Parse JTI to extract timestamp
	// Expected format: base64url(unix_timestamp) + "_" + random_suffix
	parts := parseJTI(jti)
	if parts == nil || parts.Timestamp.IsZero() {
		return 0, fmt.Errorf("invalid JTI format: cannot extract timestamp")
	}

	return v.ValidateTimestamp(parts.Timestamp)
}

// JTIParts represents parsed components of a JTI
type JTIParts struct {
	Timestamp time.Time
	Random    string
}

// parseJTI extracts timestamp and random components from JTI
// Expected format: {unix_timestamp_ms}_{random_hex}
func parseJTI(jti string) *JTIParts {
	// Simple implementation - looks for underscore separator
	// Format: timestamp_random (e.g., "1699545600000_a1b2c3d4e5f6")
	if len(jti) < 10 {
		return nil
	}

	// Find the underscore separator
	var sepIdx int = -1
	for i := 0; i < len(jti); i++ {
		if jti[i] == '_' {
			sepIdx = i
			break
		}
	}

	if sepIdx == -1 || sepIdx == 0 {
		return nil
	}

	// Parse timestamp (milliseconds since epoch)
	timestampStr := jti[:sepIdx]
	timestampMs, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil
	}

	// Convert milliseconds to time.Time
	timestamp := time.Unix(timestampMs/1000, (timestampMs%1000)*1000000)

	// Extract random suffix
	random := ""
	if sepIdx+1 < len(jti) {
		random = jti[sepIdx+1:]
	}

	return &JTIParts{
		Timestamp: timestamp,
		Random:    random,
	}
}

// SkewMetrics tracks clock skew detection metrics
type SkewMetrics struct {
	TotalValidations      int64
	ExcessiveSkewCount    int64
	WarningSkewCount      int64
	MaxObservedSkew       int64
	AverageSkew           float64
	skewSum               int64
	WarningThresholdRatio float64 // Trigger warning at this ratio of max skew (default: 0.7)
}

// NewSkewMetrics creates a new metrics tracker
func NewSkewMetrics() *SkewMetrics {
	return &SkewMetrics{
		WarningThresholdRatio: 0.7,
	}
}

// RecordSkew records a clock skew measurement
func (m *SkewMetrics) RecordSkew(skewSeconds int64, exceeded bool) {
	m.TotalValidations++
	if exceeded {
		m.ExcessiveSkewCount++
	}

	// Track absolute skew
	absSkew := skewSeconds
	if absSkew < 0 {
		absSkew = -absSkew
	}

	// Update max observed
	if absSkew > m.MaxObservedSkew {
		m.MaxObservedSkew = absSkew
	}

	// Update running average
	m.skewSum += absSkew
	m.AverageSkew = float64(m.skewSum) / float64(m.TotalValidations)
}

// IsWarningLevel checks if skew is approaching threshold (but not exceeded)
func (m *SkewMetrics) IsWarningLevel(skewSeconds int64, maxAllowed int64) bool {
	absSkew := skewSeconds
	if absSkew < 0 {
		absSkew = -absSkew
	}
	warningThreshold := int64(float64(maxAllowed) * m.WarningThresholdRatio)
	return absSkew >= warningThreshold && absSkew < maxAllowed
}

// GetStats returns formatted statistics string
func (m *SkewMetrics) GetStats() string {
	if m.TotalValidations == 0 {
		return "No validations recorded"
	}
	return fmt.Sprintf(
		"Validations: %d, Excessive: %d (%.1f%%), Max: %ds, Avg: %.1fs",
		m.TotalValidations,
		m.ExcessiveSkewCount,
		100.0*float64(m.ExcessiveSkewCount)/float64(m.TotalValidations),
		m.MaxObservedSkew,
		m.AverageSkew,
	)
}
