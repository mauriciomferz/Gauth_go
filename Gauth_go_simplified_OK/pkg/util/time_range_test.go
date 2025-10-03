package util

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeRange(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	// Create a time range
	tr := NewTimeRange(past, future)

	// Test Contains
	if !tr.Contains(now) {
		t.Error("Expected current time to be within range")
	}

	if tr.Contains(past.Add(-1 * time.Hour)) {
		t.Error("Expected time before range to not be contained")
	}

	if tr.Contains(future.Add(1 * time.Hour)) {
		t.Error("Expected time after range to not be contained")
	}

	// Test IsAllowed
	allowed, _ := tr.IsAllowed(now)
	if !allowed {
		t.Error("Expected current time to be allowed")
	}

	allowed, msg := tr.IsAllowed(past.Add(-1 * time.Hour))
	if allowed {
		t.Error("Expected time before range to not be allowed")
	}
	if msg == "" {
		t.Error("Expected error message for disallowed time")
	}

	// Test Duration
	expected := future.Sub(past)
	if tr.Duration() != expected {
		t.Errorf("Expected duration %v, got %v", expected, tr.Duration())
	}

	// Test with zero times
	zeroStart := NewTimeRange(time.Time{}, future)
	if !zeroStart.Contains(past) {
		t.Error("Expected time to be contained with zero start time")
	}

	zeroEnd := NewTimeRange(past, time.Time{})
	if !zeroEnd.Contains(future) {
		t.Error("Expected time to be contained with zero end time")
	}
}

func TestTimeRangeFromInput(t *testing.T) {
	// Valid input
	input := TimeRangeInput{
		Start: "2023-01-01T00:00:00Z",
		End:   "2023-01-02T00:00:00Z",
	}

	tr, err := NewTimeRangeFromInput(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedStart, _ := time.Parse(time.RFC3339, input.Start)
	expectedEnd, _ := time.Parse(time.RFC3339, input.End)

	if !tr.Start.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, tr.Start)
	}

	if !tr.End.Equal(expectedEnd) {
		t.Errorf("Expected end time %v, got %v", expectedEnd, tr.End)
	}

	// Invalid start time
	badInput := TimeRangeInput{
		Start: "invalid-time",
		End:   "2023-01-02T00:00:00Z",
	}

	_, err = NewTimeRangeFromInput(badInput)
	if err == nil {
		t.Error("Expected error for invalid start time")
	}

	// Invalid end time
	badInput = TimeRangeInput{
		Start: "2023-01-01T00:00:00Z",
		End:   "invalid-time",
	}

	_, err = NewTimeRangeFromInput(badInput)
	if err == nil {
		t.Error("Expected error for invalid end time")
	}
}

func TestTimeRangeJSON(t *testing.T) {
	// Create a time range
	start, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	tr := NewTimeRange(start, end)

	// Marshal to JSON
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Failed to marshal TimeRange: %v", err)
	}

	// Unmarshal back
	var decoded TimeRange
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TimeRange: %v", err)
	}

	// Check times
	if !decoded.Start.Equal(start) {
		t.Errorf("Expected start time %v, got %v", start, decoded.Start)
	}

	if !decoded.End.Equal(end) {
		t.Errorf("Expected end time %v, got %v", end, decoded.End)
	}

	// Test with zero times
	zeroTr := NewTimeRange(time.Time{}, time.Time{})
	data, err = json.Marshal(zeroTr)
	if err != nil {
		t.Fatalf("Failed to marshal TimeRange with zero times: %v", err)
	}

	// The JSON should have "unlimited" for both times
	if string(data) != `{"start":"unlimited","end":"unlimited"}` {
		t.Errorf("Unexpected JSON for zero times: %s", string(data))
	}
}
