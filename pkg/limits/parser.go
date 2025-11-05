package limits

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RateLimit represents a parsed rate limit with a numeric limit and time period.
type RateLimit struct {
	Limit  int
	Period time.Duration
}

var (
	// rateLimitPattern matches "1000/hour", "5K/day", "1M/month", etc.
	// Captures: (1) numeric value with optional K/M suffix, (2) period name
	rateLimitPattern = regexp.MustCompile(`^(\d+(?:\.\d+)?[KkMm]?)\s*/\s*(\w+)$`)
)

// ParseRateLimit parses a rate limit string like "1000/hour" or "5K/day" into a RateLimit struct.
// Supported formats:
//   - "1000/minute" -> {1000, 1*time.Minute}
//   - "5000/hour"   -> {5000, 1*time.Hour}
//   - "50K/day"     -> {50000, 24*time.Hour}
//   - "1M/month"    -> {1000000, 30*24*time.Hour}
//   - "10K/week"    -> {10000, 7*24*time.Hour}
//
// Supported period names: minute, hour, day, week, month (case-insensitive, plural forms accepted).
// Supported multipliers: K/k (1000), M/m (1000000).
func ParseRateLimit(s string) (RateLimit, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return RateLimit{}, fmt.Errorf("empty rate limit string")
	}

	matches := rateLimitPattern.FindStringSubmatch(s)
	if matches == nil {
		return RateLimit{}, fmt.Errorf("invalid rate limit format: %q (expected format: \"1000/hour\")", s)
	}

	// Parse numeric value with multiplier
	valueStr := matches[1]
	value, err := parseNumericWithMultiplier(valueStr)
	if err != nil {
		return RateLimit{}, fmt.Errorf("invalid numeric value %q: %w", valueStr, err)
	}

	// Parse period name
	periodStr := strings.ToLower(strings.TrimSpace(matches[2]))
	period, err := parsePeriod(periodStr)
	if err != nil {
		return RateLimit{}, fmt.Errorf("invalid period %q: %w", periodStr, err)
	}

	return RateLimit{Limit: value, Period: period}, nil
}

// parseNumericWithMultiplier parses a numeric string with optional K/M multiplier.
// Examples: "1000" -> 1000, "5K" -> 5000, "1.5M" -> 1500000
func parseNumericWithMultiplier(s string) (int, error) {
	s = strings.TrimSpace(s)
	multiplier := 1

	// Extract multiplier suffix
	if len(s) > 0 {
		lastChar := s[len(s)-1]
		switch lastChar {
		case 'K', 'k':
			multiplier = 1000
			s = s[:len(s)-1]
		case 'M', 'm':
			multiplier = 1000000
			s = s[:len(s)-1]
		}
	}

	// Parse base numeric value (support decimal for K/M)
	if strings.Contains(s, ".") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %w", err)
		}
		return int(f * float64(multiplier)), nil
	}

	base, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}
	if base < 0 {
		return 0, fmt.Errorf("negative numbers not allowed: %d", base)
	}
	return base * multiplier, nil
}

// parsePeriod converts a period name to time.Duration.
// Accepts singular/plural forms (case-insensitive): minute/minutes, hour/hours, day/days, week/weeks, month/months.
func parsePeriod(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	// Normalize plural forms
	s = strings.TrimSuffix(s, "s")

	switch s {
	case "minute", "min":
		return time.Minute, nil
	case "hour", "hr", "h":
		return time.Hour, nil
	case "day", "d":
		return 24 * time.Hour, nil
	case "week", "wk", "w":
		return 7 * 24 * time.Hour, nil
	case "month", "mo":
		// Approximate month as 30 days (sufficient for rate limiting)
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown period: %q (supported: minute, hour, day, week, month)", s)
	}
}

// FormatPeriod returns a human-readable period name for a duration.
func FormatPeriod(d time.Duration) string {
	switch d {
	case time.Minute:
		return "minute"
	case time.Hour:
		return "hour"
	case 24 * time.Hour:
		return "day"
	case 7 * 24 * time.Hour:
		return "week"
	case 30 * 24 * time.Hour:
		return "month"
	default:
		return d.String()
	}
}
