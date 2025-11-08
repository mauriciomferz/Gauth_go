// Package limits provides multi-period numeric limit enforcement.
package limits

import (
	"fmt"
	"sync"
	"time"
)

// Period represents a time period for limits.
type Period string

const (
	PeriodDaily   Period = "daily"
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
)

// NumericLimit represents a limit for a specific period.
type NumericLimit struct {
	Period Period
	Limit  float64
}

// UsageRecord tracks usage within a period.
type UsageRecord struct {
	Period    Period
	StartTime time.Time
	EndTime   time.Time
	Current   float64
	Limit     float64
}

// IsExpired checks if the usage record period has expired.
func (ur *UsageRecord) IsExpired() bool {
	return time.Now().After(ur.EndTime)
}

// RemainingCapacity returns how much capacity remains.
func (ur *UsageRecord) RemainingCapacity() float64 {
	remaining := ur.Limit - ur.Current
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CanAccommodate checks if an amount can be accommodated.
func (ur *UsageRecord) CanAccommodate(amount float64) bool {
	return ur.Current+amount <= ur.Limit
}

// MultiPeriodLimitTracker tracks usage across multiple time periods.
type MultiPeriodLimitTracker struct {
	mu      sync.RWMutex
	records map[string]map[Period]*UsageRecord // entityID -> period -> record
	limits  map[string]map[Period]float64      // entityID -> period -> limit
}

// NewMultiPeriodLimitTracker creates a new tracker.
func NewMultiPeriodLimitTracker() *MultiPeriodLimitTracker {
	return &MultiPeriodLimitTracker{
		records: make(map[string]map[Period]*UsageRecord),
		limits:  make(map[string]map[Period]float64),
	}
}

// SetLimit configures a limit for an entity and period.
func (t *MultiPeriodLimitTracker) SetLimit(entityID string, period Period, limit float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.limits[entityID] == nil {
		t.limits[entityID] = make(map[Period]float64)
	}
	t.limits[entityID][period] = limit
}

// SetLimits configures multiple limits for an entity.
func (t *MultiPeriodLimitTracker) SetLimits(entityID string, limits map[Period]float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.limits[entityID] == nil {
		t.limits[entityID] = make(map[Period]float64)
	}
	for period, limit := range limits {
		t.limits[entityID][period] = limit
	}
}

// CheckLimit verifies if an amount would exceed any configured limits.
func (t *MultiPeriodLimitTracker) CheckLimit(entityID string, amount float64) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entityLimits, exists := t.limits[entityID]
	if !exists {
		return nil // No limits configured
	}

	for period, limit := range entityLimits {
		record := t.getOrCreateRecord(entityID, period)

		// Reset if period expired
		if record.IsExpired() {
			t.resetRecord(entityID, period)
			record = t.getOrCreateRecord(entityID, period)
		}

		if !record.CanAccommodate(amount) {
			return fmt.Errorf("%s limit exceeded: current %.2f + amount %.2f > limit %.2f (remaining: %.2f)",
				period, record.Current, amount, limit, record.RemainingCapacity())
		}
	}

	return nil
}

// RecordUsage records usage and checks all limits.
func (t *MultiPeriodLimitTracker) RecordUsage(entityID string, amount float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// First check all limits
	entityLimits, exists := t.limits[entityID]
	if !exists {
		return nil // No limits configured
	}

	// Check all period limits
	for period := range entityLimits {
		record := t.getOrCreateRecordUnsafe(entityID, period)

		if record.IsExpired() {
			t.resetRecordUnsafe(entityID, period)
			record = t.getOrCreateRecordUnsafe(entityID, period)
		}

		if !record.CanAccommodate(amount) {
			return fmt.Errorf("%s limit would be exceeded: %.2f + %.2f > %.2f",
				period, record.Current, amount, record.Limit)
		}
	}

	// All limits OK, record usage
	for period := range entityLimits {
		record := t.getOrCreateRecordUnsafe(entityID, period)
		record.Current += amount
	}

	return nil
}

// GetUsage returns current usage for all periods.
func (t *MultiPeriodLimitTracker) GetUsage(entityID string) map[Period]*UsageRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[Period]*UsageRecord)

	entityRecords, exists := t.records[entityID]
	if !exists {
		return result
	}

	for period, record := range entityRecords {
		// Return copy to prevent external modification
		recordCopy := *record
		result[period] = &recordCopy
	}

	return result
}

// ResetPeriod manually resets a specific period for an entity.
func (t *MultiPeriodLimitTracker) ResetPeriod(entityID string, period Period) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resetRecordUnsafe(entityID, period)
}

// ResetAll resets all usage for an entity.
func (t *MultiPeriodLimitTracker) ResetAll(entityID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, entityID)
}

// getOrCreateRecord gets or creates a usage record (with read lock).
func (t *MultiPeriodLimitTracker) getOrCreateRecord(entityID string, period Period) *UsageRecord {
	if t.records[entityID] == nil {
		t.records[entityID] = make(map[Period]*UsageRecord)
	}

	record, exists := t.records[entityID][period]
	if !exists || record.IsExpired() {
		record = t.createRecord(entityID, period)
		t.records[entityID][period] = record
	}

	return record
}

// getOrCreateRecordUnsafe gets or creates a record without locking.
func (t *MultiPeriodLimitTracker) getOrCreateRecordUnsafe(entityID string, period Period) *UsageRecord {
	if t.records[entityID] == nil {
		t.records[entityID] = make(map[Period]*UsageRecord)
	}

	record, exists := t.records[entityID][period]
	if !exists || record.IsExpired() {
		record = t.createRecord(entityID, period)
		t.records[entityID][period] = record
	}

	return record
}

// createRecord creates a new usage record for a period.
func (t *MultiPeriodLimitTracker) createRecord(entityID string, period Period) *UsageRecord {
	now := time.Now()
	startTime, endTime := calculatePeriodBounds(now, period)

	limit := float64(0)
	if t.limits[entityID] != nil {
		limit = t.limits[entityID][period]
	}

	return &UsageRecord{
		Period:    period,
		StartTime: startTime,
		EndTime:   endTime,
		Current:   0,
		Limit:     limit,
	}
}

// resetRecord resets a usage record (with write lock).
func (t *MultiPeriodLimitTracker) resetRecord(entityID string, period Period) {
	if t.records[entityID] != nil {
		delete(t.records[entityID], period)
	}
}

// resetRecordUnsafe resets a record without locking.
func (t *MultiPeriodLimitTracker) resetRecordUnsafe(entityID string, period Period) {
	if t.records[entityID] != nil {
		delete(t.records[entityID], period)
	}
}

// calculatePeriodBounds calculates start and end times for a period.
func calculatePeriodBounds(now time.Time, period Period) (time.Time, time.Time) {
	switch period {
	case PeriodDaily:
		// Start: beginning of today, End: end of today
		year, month, day := now.Date()
		start := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
		end := start.Add(24 * time.Hour)
		return start, end

	case PeriodWeekly:
		// Start: beginning of current week (Monday), End: end of week (Sunday)
		weekday := now.Weekday()
		start := time.Date(now.Year(), now.Month(), now.Day()-int(weekday)+int(time.Monday), 0, 0, 0, 0, now.Location())
		if weekday == time.Sunday {
			start = start.AddDate(0, 0, -6)
		}
		end := start.AddDate(0, 0, 7)
		return start, end

	case PeriodMonthly:
		// Start: first day of current month, End: first day of next month
		year, month, _ := now.Date()
		start := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		return start, end

	case PeriodYearly:
		// Start: first day of current year, End: first day of next year
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(1, 0, 0)
		return start, end

	default:
		// Default to daily
		year, month, day := now.Date()
		start := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
		end := start.Add(24 * time.Hour)
		return start, end
	}
}

// GetStatistics returns summary statistics for an entity.
func (t *MultiPeriodLimitTracker) GetStatistics(entityID string) map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := map[string]interface{}{
		"entity_id": entityID,
		"periods":   make(map[Period]map[string]interface{}),
	}

	entityLimits, hasLimits := t.limits[entityID]
	entityRecords, hasRecords := t.records[entityID]

	if !hasLimits && !hasRecords {
		return stats
	}

	periods := stats["periods"].(map[Period]map[string]interface{})

	for period := range entityLimits {
		periodStats := make(map[string]interface{})
		periodStats["limit"] = entityLimits[period]

		if record, exists := entityRecords[period]; exists {
			periodStats["current"] = record.Current
			periodStats["remaining"] = record.RemainingCapacity()
			periodStats["utilization"] = (record.Current / record.Limit) * 100
			periodStats["start_time"] = record.StartTime
			periodStats["end_time"] = record.EndTime
		} else {
			periodStats["current"] = 0.0
			periodStats["remaining"] = entityLimits[period]
			periodStats["utilization"] = 0.0
		}

		periods[period] = periodStats
	}

	return stats
}
