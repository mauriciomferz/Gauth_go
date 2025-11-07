// Package limits tests.
package limits

import (
	"testing"
	"time"
)

func TestMultiPeriodLimitTracker_SetLimit(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimit("user:alice", PeriodDaily, 1000)
	tracker.SetLimit("user:alice", PeriodWeekly, 5000)
	tracker.SetLimit("user:alice", PeriodMonthly, 20000)
	
	usage := tracker.GetUsage("user:alice")
	if len(usage) != 0 {
		t.Error("Expected no usage initially")
	}
}

func TestMultiPeriodLimitTracker_RecordUsage(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimit("user:alice", PeriodDaily, 100)
	
	// Record some usage
	err := tracker.RecordUsage("user:alice", 50)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}
	
	usage := tracker.GetUsage("user:alice")
	if len(usage) != 1 {
		t.Fatalf("Expected 1 usage record, got %d", len(usage))
	}
	
	dailyUsage, exists := usage[PeriodDaily]
	if !exists {
		t.Fatal("Expected daily usage record")
	}
	
	if dailyUsage.Current != 50 {
		t.Errorf("Expected current usage 50, got %.2f", dailyUsage.Current)
	}
	
	if dailyUsage.RemainingCapacity() != 50 {
		t.Errorf("Expected remaining 50, got %.2f", dailyUsage.RemainingCapacity())
	}
}

func TestMultiPeriodLimitTracker_ExceedLimit(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimit("user:bob", PeriodDaily, 100)
	
	// Record usage up to limit
	err := tracker.RecordUsage("user:bob", 80)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}
	
	// This should exceed the limit
	err = tracker.RecordUsage("user:bob", 30)
	if err == nil {
		t.Error("Expected error for exceeding limit")
	}
	
	// Usage should not have been recorded
	usage := tracker.GetUsage("user:bob")
	dailyUsage := usage[PeriodDaily]
	if dailyUsage.Current != 80 {
		t.Errorf("Expected usage to remain at 80, got %.2f", dailyUsage.Current)
	}
}

func TestMultiPeriodLimitTracker_MultiplePeriodsAllEnforced(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	// Set multiple limits
	tracker.SetLimit("user:charlie", PeriodDaily, 100)
	tracker.SetLimit("user:charlie", PeriodWeekly, 500)
	tracker.SetLimit("user:charlie", PeriodMonthly, 2000)
	
	// Record 90 (under daily limit)
	err := tracker.RecordUsage("user:charlie", 90)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}
	
	// Try to record 20 more (would exceed daily limit of 100)
	err = tracker.RecordUsage("user:charlie", 20)
	if err == nil {
		t.Error("Expected error for exceeding daily limit")
	}
	
	// Can record 10 more (stays within daily limit)
	err = tracker.RecordUsage("user:charlie", 10)
	if err != nil {
		t.Fatalf("RecordUsage within limit failed: %v", err)
	}
	
	// Verify all periods updated
	usage := tracker.GetUsage("user:charlie")
	
	if len(usage) != 3 {
		t.Fatalf("Expected 3 usage records, got %d", len(usage))
	}
	
	// Check each period
	for period, record := range usage {
		if record.Current != 100 {
			t.Errorf("Period %s: expected current 100, got %.2f", period, record.Current)
		}
	}
}

func TestMultiPeriodLimitTracker_CheckLimit(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimit("user:dave", PeriodDaily, 100)
	
	// Check without recording
	err := tracker.CheckLimit("user:dave", 50)
	if err != nil {
		t.Errorf("CheckLimit should allow 50: %v", err)
	}
	
	// Record actual usage
	_ = tracker.RecordUsage("user:dave", 80)
	
	// Check if we can add more
	err = tracker.CheckLimit("user:dave", 25)
	if err == nil {
		t.Error("CheckLimit should prevent exceeding limit")
	}
	
	err = tracker.CheckLimit("user:dave", 15)
	if err != nil {
		t.Errorf("CheckLimit should allow 15: %v", err)
	}
}

func TestMultiPeriodLimitTracker_Reset(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimit("user:eve", PeriodDaily, 100)
	_ = tracker.RecordUsage("user:eve", 80)
	
	// Reset daily period
	tracker.ResetPeriod("user:eve", PeriodDaily)
	
	// Should be able to use full limit again
	err := tracker.RecordUsage("user:eve", 100)
	if err != nil {
		t.Fatalf("RecordUsage after reset failed: %v", err)
	}
	
	usage := tracker.GetUsage("user:eve")
	dailyUsage := usage[PeriodDaily]
	if dailyUsage.Current != 100 {
		t.Errorf("Expected current usage 100 after reset, got %.2f", dailyUsage.Current)
	}
}

func TestMultiPeriodLimitTracker_ResetAll(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimits("user:frank", map[Period]float64{
		PeriodDaily:   100,
		PeriodWeekly:  500,
		PeriodMonthly: 2000,
	})
	
	_ = tracker.RecordUsage("user:frank", 50)
	
	// Reset all
	tracker.ResetAll("user:frank")
	
	usage := tracker.GetUsage("user:frank")
	if len(usage) != 0 {
		t.Errorf("Expected no usage records after ResetAll, got %d", len(usage))
	}
}

func TestMultiPeriodLimitTracker_GetStatistics(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	
	tracker.SetLimit("user:grace", PeriodDaily, 100)
	tracker.SetLimit("user:grace", PeriodWeekly, 500)
	
	_ = tracker.RecordUsage("user:grace", 30)
	
	stats := tracker.GetStatistics("user:grace")
	
	if stats["entity_id"] != "user:grace" {
		t.Error("Expected entity_id in statistics")
	}
	
	periods, ok := stats["periods"].(map[Period]map[string]interface{})
	if !ok {
		t.Fatal("Expected periods map in statistics")
	}
	
	if len(periods) != 2 {
		t.Errorf("Expected 2 periods in statistics, got %d", len(periods))
	}
	
	// Check daily stats
	dailyStats, exists := periods[PeriodDaily]
	if !exists {
		t.Fatal("Expected daily statistics")
	}
	
	if dailyStats["limit"].(float64) != 100 {
		t.Errorf("Expected daily limit 100, got %.2f", dailyStats["limit"].(float64))
	}
	
	if dailyStats["current"].(float64) != 30 {
		t.Errorf("Expected current 30, got %.2f", dailyStats["current"].(float64))
	}
	
	if dailyStats["remaining"].(float64) != 70 {
		t.Errorf("Expected remaining 70, got %.2f", dailyStats["remaining"].(float64))
	}
	
	utilization := dailyStats["utilization"].(float64)
	if utilization < 29.9 || utilization > 30.1 {
		t.Errorf("Expected utilization ~30%%, got %.2f%%", utilization)
	}
}

func TestUsageRecord_RemainingCapacity(t *testing.T) {
	record := &UsageRecord{
		Current: 60,
		Limit:   100,
	}
	
	if record.RemainingCapacity() != 40 {
		t.Errorf("Expected remaining 40, got %.2f", record.RemainingCapacity())
	}
	
	// Over limit
	record.Current = 110
	if record.RemainingCapacity() != 0 {
		t.Errorf("Expected remaining 0 when over limit, got %.2f", record.RemainingCapacity())
	}
}

func TestUsageRecord_CanAccommodate(t *testing.T) {
	record := &UsageRecord{
		Current: 60,
		Limit:   100,
	}
	
	if !record.CanAccommodate(30) {
		t.Error("Should accommodate 30")
	}
	
	if !record.CanAccommodate(40) {
		t.Error("Should accommodate exactly 40")
	}
	
	if record.CanAccommodate(50) {
		t.Error("Should not accommodate 50")
	}
}

func TestMultiPeriodCounter_YearlyPeriod(t *testing.T) {
	cache := NewMemoryStore()
	counter := NewMultiPeriodCounter(cache)
	now := time.Date(2025, 11, 6, 15, 30, 0, 0, time.UTC)

	// Yearly period
	expectedStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	start, end := counter.getPeriodBounds(now, PeriodYearly)

	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}

func TestCalculatePeriodBounds_Weekly(t *testing.T) {
	// Wednesday, Nov 6, 2025
	now := time.Date(2025, 11, 6, 15, 30, 0, 0, time.UTC)
	start, end := calculatePeriodBounds(now, PeriodWeekly)
	
	// Week should start on Monday, Nov 3
	expectedStart := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
	// Week should end on Monday, Nov 10
	expectedEnd := time.Date(2025, 11, 10, 0, 0, 0, 0, time.UTC)
	
	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}

func TestCalculatePeriodBounds_Monthly(t *testing.T) {
	now := time.Date(2025, 11, 6, 15, 30, 0, 0, time.UTC)
	start, end := calculatePeriodBounds(now, PeriodMonthly)
	
	expectedStart := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	
	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}

func TestCalculatePeriodBounds_Yearly(t *testing.T) {
	now := time.Date(2025, 11, 6, 15, 30, 0, 0, time.UTC)
	start, end := calculatePeriodBounds(now, PeriodYearly)
	
	expectedStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	
	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}

func TestMultiPeriodLimitTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewMultiPeriodLimitTracker()
	tracker.SetLimit("user:concurrent", PeriodDaily, 1000)
	
	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = tracker.RecordUsage("user:concurrent", 1)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify total usage
	usage := tracker.GetUsage("user:concurrent")
	dailyUsage := usage[PeriodDaily]
	if dailyUsage.Current != 100 {
		t.Errorf("Expected total usage 100, got %.2f", dailyUsage.Current)
	}
}

func BenchmarkMultiPeriodLimitTracker_RecordUsage(b *testing.B) {
	tracker := NewMultiPeriodLimitTracker()
	tracker.SetLimits("bench:user", map[Period]float64{
		PeriodDaily:   100000,
		PeriodWeekly:  500000,
		PeriodMonthly: 2000000,
	})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.RecordUsage("bench:user", 1)
	}
}

func BenchmarkMultiPeriodLimitTracker_CheckLimit(b *testing.B) {
	tracker := NewMultiPeriodLimitTracker()
	tracker.SetLimit("bench:user", PeriodDaily, 100000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.CheckLimit("bench:user", 1)
	}
}
