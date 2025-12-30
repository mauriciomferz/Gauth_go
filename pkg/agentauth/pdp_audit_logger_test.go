package agentauth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProductionPEPAuditLogger(t *testing.T) {
	t.Run("with valid configuration", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(5000, true, true)

		require.NotNil(t, logger)
		assert.Equal(t, 5000, logger.maxEntries)
		assert.True(t, logger.enableConsole)
		assert.True(t, logger.enableMetrics)
		assert.Equal(t, int64(0), logger.totalEnforcements)
		assert.Equal(t, int64(0), logger.totalViolations)
	})

	t.Run("with zero maxEntries uses default", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(0, false, false)

		require.NotNil(t, logger)
		assert.Equal(t, 10000, logger.maxEntries) // Default
	})

	t.Run("with negative maxEntries uses default", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(-100, false, false)

		require.NotNil(t, logger)
		assert.Equal(t, 10000, logger.maxEntries) // Default
	})
}

func TestProductionPEPAuditLogger_LogEnforcement(t *testing.T) {
	ctx := context.Background()

	t.Run("logs single enforcement successfully", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		entry := &EnforcementAuditEntry{
			EnforcementID:  "enf-001",
			Timestamp:      time.Now(),
			ActionType:     "transaction",
			ResourceID:     "res-123",
			Outcome:        "allowed",
			Allowed:        true,
			Reason:         "all checks passed",
			ViolationCount: 0,
		}

		err := logger.LogEnforcement(ctx, entry)

		require.NoError(t, err)
		assert.Equal(t, int64(1), logger.totalEnforcements)

		enforcements := logger.GetEnforcements(10)
		require.Len(t, enforcements, 1)
		assert.Equal(t, "enf-001", enforcements[0].EnforcementID)
		assert.Equal(t, "transaction", enforcements[0].ActionType)
		assert.True(t, enforcements[0].Allowed)
	})

	t.Run("logs multiple enforcements", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		for i := 0; i < 10; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-" + time.Now().Format("20060102150405"),
				Timestamp:      time.Now(),
				ActionType:     "action",
				ResourceID:     "res-456",
				Outcome:        "allowed",
				Allowed:        true,
				Reason:         "authorized",
				ViolationCount: 0,
			}

			err := logger.LogEnforcement(ctx, entry)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond) // Ensure unique timestamps
		}

		assert.Equal(t, int64(10), logger.totalEnforcements)

		enforcements := logger.GetEnforcements(10)
		assert.Len(t, enforcements, 10)
	})

	t.Run("rotates when maxEntries exceeded", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(5, false, false) // Small buffer

		// Log 10 entries (exceeds max of 5)
		for i := 0; i < 10; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-" + time.Now().Format("20060102150405.000000"),
				Timestamp:      time.Now(),
				ActionType:     "decision",
				ResourceID:     "res-789",
				Outcome:        "denied",
				Allowed:        false,
				Reason:         "insufficient permissions",
				ViolationCount: 1,
			}

			err := logger.LogEnforcement(ctx, entry)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond)
		}

		// Should only keep last 5 entries
		assert.Equal(t, int64(10), logger.totalEnforcements) // Total count preserved

		enforcements := logger.GetEnforcements(10)
		assert.Len(t, enforcements, 5) // Only 5 stored (FIFO rotation)
	})

	t.Run("thread-safe concurrent logging", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(1000, false, false)

		var wg sync.WaitGroup
		concurrency := 10
		entriesPerGoroutine := 10

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < entriesPerGoroutine; j++ {
					entry := &EnforcementAuditEntry{
						EnforcementID:  "concurrent-enf",
						Timestamp:      time.Now(),
						ActionType:     "action",
						ResourceID:     "res-concurrent",
						Outcome:        "allowed",
						Allowed:        true,
						Reason:         "concurrent test",
						ViolationCount: 0,
					}

					_ = logger.LogEnforcement(ctx, entry)
				}
			}(i)
		}

		wg.Wait()

		// Verify all entries logged
		assert.Equal(t, int64(concurrency*entriesPerGoroutine), logger.totalEnforcements)
	})
}

func TestProductionPEPAuditLogger_LogViolation(t *testing.T) {
	ctx := context.Background()

	t.Run("logs single violation successfully", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		entry := &ViolationAuditEntry{
			EnforcementID: "enf-violation-001",
			Timestamp:     time.Now(),
			ViolationType: "scope",
			Severity:      "high",
			Description:   "Geographic scope violation detected",
			ActionType:    "transaction",
			ResourceID:    "res-violation-123",
		}

		err := logger.LogViolation(ctx, entry)

		require.NoError(t, err)
		assert.Equal(t, int64(1), logger.totalViolations)

		violations := logger.GetViolations(10)
		require.Len(t, violations, 1)
		assert.Equal(t, "enf-violation-001", violations[0].EnforcementID)
		assert.Equal(t, "scope", violations[0].ViolationType)
		assert.Equal(t, "high", violations[0].Severity)
	})

	t.Run("logs multiple violations with different severities", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		severities := []string{"critical", "high", "medium", "low"}

		for i, severity := range severities {
			entry := &ViolationAuditEntry{
				EnforcementID: "enf-sev-" + severity,
				Timestamp:     time.Now(),
				ViolationType: "restriction",
				Severity:      severity,
				Description:   "Severity test: " + severity,
				ActionType:    "decision",
				ResourceID:    "res-severity",
			}

			err := logger.LogViolation(ctx, entry)
			require.NoError(t, err)

			assert.Equal(t, int64(i+1), logger.totalViolations)
		}

		violations := logger.GetViolations(10)
		assert.Len(t, violations, 4)
	})

	t.Run("rotates when maxEntries exceeded", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(3, false, false) // Small buffer

		// Log 5 violations (exceeds max of 3)
		for i := 0; i < 5; i++ {
			entry := &ViolationAuditEntry{
				EnforcementID: "enf-rot",
				Timestamp:     time.Now(),
				ViolationType: "compliance",
				Severity:      "medium",
				Description:   "Rotation test",
				ActionType:    "action",
				ResourceID:    "res-rotation",
			}

			err := logger.LogViolation(ctx, entry)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond)
		}

		// Should only keep last 3 entries
		assert.Equal(t, int64(5), logger.totalViolations) // Total count preserved

		violations := logger.GetViolations(10)
		assert.Len(t, violations, 3) // Only 3 stored (FIFO rotation)
	})

	t.Run("thread-safe concurrent violation logging", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(1000, false, false)

		var wg sync.WaitGroup
		concurrency := 10
		entriesPerGoroutine := 10

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < entriesPerGoroutine; j++ {
					entry := &ViolationAuditEntry{
						EnforcementID: "concurrent-viol",
						Timestamp:     time.Now(),
						ViolationType: "temporal",
						Severity:      "low",
						Description:   "Concurrent test",
						ActionType:    "action",
						ResourceID:    "res-concurrent",
					}

					_ = logger.LogViolation(ctx, entry)
				}
			}(i)
		}

		wg.Wait()

		// Verify all entries logged
		assert.Equal(t, int64(concurrency*entriesPerGoroutine), logger.totalViolations)
	})
}

func TestProductionPEPAuditLogger_GetEnforcements(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty when no enforcements", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		enforcements := logger.GetEnforcements(10)

		assert.Empty(t, enforcements)
	})

	t.Run("returns all enforcements when limit exceeds count", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		// Log 5 enforcements
		for i := 0; i < 5; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-get-" + string(rune('0'+i)),
				Timestamp:      time.Now(),
				ActionType:     "action",
				ResourceID:     "res-get",
				Outcome:        "allowed",
				Allowed:        true,
				Reason:         "test",
				ViolationCount: 0,
			}
			_ = logger.LogEnforcement(ctx, entry)
		}

		enforcements := logger.GetEnforcements(100) // Request more than available

		assert.Len(t, enforcements, 5)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		// Log 10 enforcements
		for i := 0; i < 10; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-limit",
				Timestamp:      time.Now(),
				ActionType:     "decision",
				ResourceID:     "res-limit",
				Outcome:        "denied",
				Allowed:        false,
				Reason:         "test limit",
				ViolationCount: 1,
			}
			_ = logger.LogEnforcement(ctx, entry)
		}

		enforcements := logger.GetEnforcements(3) // Request only 3

		assert.Len(t, enforcements, 3)
	})

	t.Run("returns most recent entries", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		// Log entries with identifiable IDs
		for i := 0; i < 10; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-" + string(rune('A'+i)),
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				ActionType:     "action",
				ResourceID:     "res-recent",
				Outcome:        "allowed",
				Allowed:        true,
				Reason:         "test recent",
				ViolationCount: 0,
			}
			_ = logger.LogEnforcement(ctx, entry)
		}

		enforcements := logger.GetEnforcements(3) // Get last 3

		require.Len(t, enforcements, 3)
		// Should get entries "enf-H", "enf-I", "enf-J" (last 3)
		assert.Equal(t, "enf-H", enforcements[0].EnforcementID)
		assert.Equal(t, "enf-I", enforcements[1].EnforcementID)
		assert.Equal(t, "enf-J", enforcements[2].EnforcementID)
	})
}

func TestProductionPEPAuditLogger_GetViolations(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty when no violations", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		violations := logger.GetViolations(10)

		assert.Empty(t, violations)
	})

	t.Run("returns all violations when limit exceeds count", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		// Log 5 violations
		for i := 0; i < 5; i++ {
			entry := &ViolationAuditEntry{
				EnforcementID: "enf-get-viol",
				Timestamp:     time.Now(),
				ViolationType: "geographic",
				Severity:      "medium",
				Description:   "Test get violations",
				ActionType:    "transaction",
				ResourceID:    "res-get-viol",
			}
			_ = logger.LogViolation(ctx, entry)
		}

		violations := logger.GetViolations(100) // Request more than available

		assert.Len(t, violations, 5)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		// Log 10 violations
		for i := 0; i < 10; i++ {
			entry := &ViolationAuditEntry{
				EnforcementID: "enf-limit-viol",
				Timestamp:     time.Now(),
				ViolationType: "restriction",
				Severity:      "high",
				Description:   "Test limit",
				ActionType:    "decision",
				ResourceID:    "res-limit-viol",
			}
			_ = logger.LogViolation(ctx, entry)
		}

		violations := logger.GetViolations(4) // Request only 4

		assert.Len(t, violations, 4)
	})
}

func TestProductionPEPAuditLogger_GetStatistics(t *testing.T) {
	ctx := context.Background()

	t.Run("returns correct statistics for empty logger", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(1000, false, false)

		stats := logger.GetStatistics()

		assert.Equal(t, int64(0), stats["total_enforcements"])
		assert.Equal(t, int64(0), stats["total_violations"])
		assert.Equal(t, 0, stats["stored_enforcements"])
		assert.Equal(t, 0, stats["stored_violations"])
		assert.Equal(t, 1000, stats["max_entries"])
		assert.Equal(t, 0.0, stats["enforcement_storage_usage"])
		assert.Equal(t, 0.0, stats["violation_storage_usage"])
	})

	t.Run("returns correct statistics after logging", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(100, false, false)

		// Log 30 enforcements
		for i := 0; i < 30; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-stats",
				Timestamp:      time.Now(),
				ActionType:     "action",
				ResourceID:     "res-stats",
				Outcome:        "allowed",
				Allowed:        true,
				Reason:         "test stats",
				ViolationCount: 0,
			}
			_ = logger.LogEnforcement(ctx, entry)
		}

		// Log 20 violations
		for i := 0; i < 20; i++ {
			entry := &ViolationAuditEntry{
				EnforcementID: "enf-stats-viol",
				Timestamp:     time.Now(),
				ViolationType: "compliance",
				Severity:      "low",
				Description:   "Test stats",
				ActionType:    "action",
				ResourceID:    "res-stats-viol",
			}
			_ = logger.LogViolation(ctx, entry)
		}

		stats := logger.GetStatistics()

		assert.Equal(t, int64(30), stats["total_enforcements"])
		assert.Equal(t, int64(20), stats["total_violations"])
		assert.Equal(t, 30, stats["stored_enforcements"])
		assert.Equal(t, 20, stats["stored_violations"])
		assert.Equal(t, 100, stats["max_entries"])
		assert.Equal(t, 30.0, stats["enforcement_storage_usage"]) // 30/100 * 100 = 30%
		assert.Equal(t, 20.0, stats["violation_storage_usage"])   // 20/100 * 100 = 20%
	})

	t.Run("preserves total count after rotation", func(t *testing.T) {
		logger := NewProductionPEPAuditLogger(10, false, false) // Small buffer

		// Log 50 enforcements (will rotate)
		for i := 0; i < 50; i++ {
			entry := &EnforcementAuditEntry{
				EnforcementID:  "enf-rotation-stats",
				Timestamp:      time.Now(),
				ActionType:     "decision",
				ResourceID:     "res-rotation-stats",
				Outcome:        "denied",
				Allowed:        false,
				Reason:         "test rotation",
				ViolationCount: 1,
			}
			_ = logger.LogEnforcement(ctx, entry)
		}

		stats := logger.GetStatistics()

		assert.Equal(t, int64(50), stats["total_enforcements"])    // Total count preserved
		assert.Equal(t, 10, stats["stored_enforcements"])          // Only 10 stored
		assert.Equal(t, 100.0, stats["enforcement_storage_usage"]) // 10/10 * 100 = 100%
	})
}

func TestNoopPEPAuditLogger(t *testing.T) {
	ctx := context.Background()
	logger := &noopPEPAuditLogger{}

	t.Run("logs enforcement without error", func(t *testing.T) {
		entry := &EnforcementAuditEntry{
			EnforcementID:  "noop-enf",
			Timestamp:      time.Now(),
			ActionType:     "action",
			ResourceID:     "res-noop",
			Outcome:        "allowed",
			Allowed:        true,
			Reason:         "noop test",
			ViolationCount: 0,
		}

		err := logger.LogEnforcement(ctx, entry)

		assert.NoError(t, err)
	})

	t.Run("logs violation without error", func(t *testing.T) {
		entry := &ViolationAuditEntry{
			EnforcementID: "noop-viol",
			Timestamp:     time.Now(),
			ViolationType: "scope",
			Severity:      "high",
			Description:   "Noop test",
			ActionType:    "action",
			ResourceID:    "res-noop",
		}

		err := logger.LogViolation(ctx, entry)

		assert.NoError(t, err)
	})
}
