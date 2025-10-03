package restriction

import (
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/util"
)

// TimeRange is an alias to util.TimeRange for better type safety
type TimeRange = util.TimeRange

// TimeRangeInput is an alias to util.TimeRangeInput for better type safety
type TimeRangeInput = util.TimeRangeInput

// ParseTimeRange parses a time range from structured input
// This is a thin wrapper around util.NewTimeRangeFromInput for backward compatibility
func ParseTimeRange(input TimeRangeInput) (*TimeRange, error) {
	return util.NewTimeRangeFromInput(input)
}

// ParseTimeRangeFromMap parses a time range from a map (legacy support)
func ParseTimeRangeFromMap(data map[string]interface{}) (*TimeRange, error) {
	input := util.TimeRangeInput{}

	if startStr, ok := data["start"].(string); ok {
		input.Start = startStr
	}

	if endStr, ok := data["end"].(string); ok {
		input.End = endStr
	}

	return util.NewTimeRangeFromInput(input)
}

// IsAllowedTime checks if the given time falls within the time range
// This is a utility function for backward compatibility
func IsAllowedTime(tr *TimeRange, t time.Time) (bool, string) {
	return tr.IsAllowed(t)
}
