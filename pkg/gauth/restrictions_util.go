package gauth

import (
	"time"
)

// RestrictionsUtil provides helper functions for working with restrictions
// and helps transition from map[string]interface{} to typed Properties

// CreateTimeRangeRestriction creates a new time-based restriction
func CreateTimeRangeRestriction(start, end time.Time) Restriction {
	props := NewProperties()
	props.SetTime("start", start)
	props.SetTime("end", end)

	return Restriction{
		Type:        "time",
		Description: "Time-based access restriction",
		Enforced:    true,
		Properties:  props,
	}
}

// CreateIPRangeRestriction creates a new IP-based restriction
func CreateIPRangeRestriction(allowedRanges []string) Restriction {
	props := NewProperties()
	for i, ipRange := range allowedRanges {
		props.SetString("range_"+string(i), ipRange)
	}

	return Restriction{
		Type:        "ip",
		Description: "IP-based access restriction",
		Enforced:    true,
		Properties:  props,
	}
}

// CreateRateLimitRestriction creates a new rate limit restriction
func CreateRateLimitRestriction(limit int, duration time.Duration) Restriction {
	props := NewProperties()
	props.SetInt("limit", limit)
	props.SetInt64("duration_ms", duration.Milliseconds())

	return Restriction{
		Type:        "rate",
		Description: "Rate limit restriction",
		Enforced:    true,
		Properties:  props,
	}
}

// GetTimeRange extracts a time range from a restriction
func GetTimeRange(r Restriction) (start, end time.Time, ok bool) {
	if r.Type != "time" || r.Properties == nil {
		return time.Time{}, time.Time{}, false
	}

	start, startOk := r.Properties.GetTime("start")
	end, endOk := r.Properties.GetTime("end")

	return start, end, startOk && endOk
}

// GetIPRanges extracts IP ranges from a restriction
func GetIPRanges(r Restriction) ([]string, bool) {
	if r.Type != "ip" || r.Properties == nil {
		return nil, false
	}

	var ranges []string
	for _, key := range r.Properties.Keys() {
		if ipRange, ok := r.Properties.GetString(key); ok {
			ranges = append(ranges, ipRange)
		}
	}

	return ranges, len(ranges) > 0
}

// GetRateLimit extracts rate limit from a restriction
func GetRateLimit(r Restriction) (limit int, duration time.Duration, ok bool) {
	if r.Type != "rate" || r.Properties == nil {
		return 0, 0, false
	}

	limit, limitOk := r.Properties.GetInt("limit")
	durationMs, durationOk := r.Properties.GetInt64("duration_ms")

	return limit, time.Duration(durationMs) * time.Millisecond, limitOk && durationOk
}

// LegacyCompatGetPropertiesMap gets a map representation for backward compatibility
// Deprecated: LegacyCompatGetPropertiesMap gets a map representation for backward compatibility.
// This function exists only for migration from legacy code using map[string]interface{}.
// Use strongly-typed Properties methods instead.
func LegacyCompatGetPropertiesMap(r Restriction) map[string]interface{} {
	if r.Properties == nil {
		return nil
	}

	return r.Properties.ToMap()
}
