package poa

import (
	"testing"
	"time"
)

// ValidateValidityPeriod checks if the given validity period is active relative to 'now',
// allowing for a specified clock skew.
// Returns nil if valid (active), error otherwise.
func ValidateValidityPeriod(vp ValidityPeriod, now time.Time, skew time.Duration) bool {
	// Sanity check
	if vp.EndTime.Before(vp.StartTime) {
		return false
	}

	// Adjusted boundaries with skew
	// ValidFrom - skew <= now <= ValidUntil + skew
	start := vp.StartTime.Add(-skew)
	end := vp.EndTime.Add(skew)

	return (now.Equal(start) || now.After(start)) && (now.Equal(end) || now.Before(end))
}

func TestValidityPeriodSkew_RFC115_C3(t *testing.T) {
	now := time.Now().UTC()
	skew := 30 * time.Second

	vp := ValidityPeriod{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
	}

	// 1. Normal active case
	if !ValidateValidityPeriod(vp, now, skew) {
		t.Error("Expected valid for active period")
	}

	// 2. Future case (not yet valid)
	future := ValidityPeriod{
		StartTime: now.Add(1 * time.Hour),
		EndTime:   now.Add(2 * time.Hour),
	}
	if ValidateValidityPeriod(future, now, skew) {
		t.Error("Expected invalid for future period")
	}

	// 3. Expired case
	expired := ValidityPeriod{
		StartTime: now.Add(-2 * time.Hour),
		EndTime:   now.Add(-1 * time.Hour),
	}
	if ValidateValidityPeriod(expired, now, skew) {
		t.Error("Expected invalid for expired period")
	}

	// 4. Skew tolerance (Early)
	// Start time is 20s in the future. Skew is 30s. Should be valid.
	nearlyActive := ValidityPeriod{
		StartTime: now.Add(20 * time.Second),
		EndTime:   now.Add(1 * time.Hour),
	}
	if !ValidateValidityPeriod(nearlyActive, now, skew) {
		t.Error("Expected valid due to skew tolerance (early)")
	}

	// 5. Skew tolerance (Late)
	// End time was 20s ago. Skew is 30s. Should be valid.
	justExpired := ValidityPeriod{
		StartTime: now.Add(-2 * time.Hour),
		EndTime:   now.Add(-20 * time.Second),
	}
	if !ValidateValidityPeriod(justExpired, now, skew) {
		t.Error("Expected valid due to skew tolerance (late)")
	}

	// 6. Outside Skew
	// Start time is 40s in the future. Skew is 30s. Should be invalid.
	tooEarly := ValidityPeriod{
		StartTime: now.Add(40 * time.Second),
		EndTime:   now.Add(1 * time.Hour),
	}
	if ValidateValidityPeriod(tooEarly, now, skew) {
		t.Error("Expected invalid outside skew tolerance")
	}
}
