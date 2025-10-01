// Package util provides common utility types and functions for the GAuth framework
package util

import (
	"encoding/json"
	"fmt"
	"time"
)

// TimeRange represents a time-based window with a start and end time
type TimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

// TimeRangeInput is used to parse time ranges from string inputs
type TimeRangeInput struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// NewTimeRange creates a new TimeRange from start and end times
func NewTimeRange(start, end time.Time) *TimeRange {
	return &TimeRange{
		Start: start,
		End:   end,
	}
}

// NewTimeRangeFromInput creates a TimeRange from a TimeRangeInput
func NewTimeRangeFromInput(input TimeRangeInput) (*TimeRange, error) {
	var tr TimeRange
	var err error

	if input.Start != "" {
		tr.Start, err = time.Parse(time.RFC3339, input.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
	}

	if input.End != "" {
		tr.End, err = time.Parse(time.RFC3339, input.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}
	}

	return &tr, nil
}

// Contains checks if a given time is within the time range
func (tr *TimeRange) Contains(t time.Time) bool {
	if !tr.Start.IsZero() && t.Before(tr.Start) {
		return false
	}

	if !tr.End.IsZero() && t.After(tr.End) {
		return false
	}

	return true
}

// IsAllowed checks if the given time falls within the time range
// and returns a message if not allowed
func (tr *TimeRange) IsAllowed(t time.Time) (bool, string) {
	if !tr.Start.IsZero() && t.Before(tr.Start) {
		return false, "Action not allowed before specified start time"
	}

	if !tr.End.IsZero() && t.After(tr.End) {
		return false, "Action not allowed after specified end time"
	}

	return true, ""
}

// Duration returns the duration of the time range
func (tr *TimeRange) Duration() time.Duration {
	// If either time is zero, we can't calculate a duration
	if tr.Start.IsZero() || tr.End.IsZero() {
		return 0
	}
	return tr.End.Sub(tr.Start)
}

// MarshalJSON implements custom JSON marshaling
func (tr *TimeRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Start string `json:"start,omitempty"`
		End   string `json:"end,omitempty"`
	}{
		Start: formatTimeOrEmpty(tr.Start),
		End:   formatTimeOrEmpty(tr.End),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (tr *TimeRange) UnmarshalJSON(data []byte) error {
	var input struct {
		Start string `json:"start,omitempty"`
		End   string `json:"end,omitempty"`
	}

	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	var err error
	if input.Start != "" {
		tr.Start, err = time.Parse(time.RFC3339, input.Start)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
	}

	if input.End != "" {
		tr.End, err = time.Parse(time.RFC3339, input.End)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
	}

	return nil
}

// String returns a string representation of the time range
func (tr *TimeRange) String() string {
	return fmt.Sprintf("[%s to %s]",
		formatTimeOrEmpty(tr.Start),
		formatTimeOrEmpty(tr.End))
}

// formatTimeOrEmpty formats a time or returns "unlimited" if it's zero
func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return "unlimited"
	}
	return t.Format(time.RFC3339)
}
