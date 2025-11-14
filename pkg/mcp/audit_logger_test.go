// Package mcp - Tests for Audit Logger
package mcp

import (
	"context"
	"testing"
	"time"
)

func TestNewInMemoryAuditLogger(t *testing.T) {
	logger := NewInMemoryAuditLogger(100)
	if logger == nil {
		t.Fatal("NewInMemoryAuditLogger() returned nil")
	}
	if logger.maxSize != 100 {
		t.Errorf("maxSize = %d, want 100", logger.maxSize)
	}

	// Test with invalid size
	logger2 := NewInMemoryAuditLogger(0)
	if logger2.maxSize != 10000 {
		t.Errorf("maxSize with 0 input = %d, want 10000 (default)", logger2.maxSize)
	}
}

func TestInMemoryAuditLogger_Log(t *testing.T) {
	logger := NewInMemoryAuditLogger(10)
	ctx := context.Background()

	entry := &AuditLogEntry{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		RequestID:  "req-123",
		Operation:  "resource_read",
		Target:     "file:///test.txt",
		Authorized: true,
		Decision:   "granted",
	}

	err := logger.Log(ctx, entry)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}

	entries := logger.GetAllEntries()
	if len(entries) != 1 {
		t.Errorf("GetAllEntries() count = %d, want 1", len(entries))
	}
}

func TestInMemoryAuditLogger_LogNil(t *testing.T) {
	logger := NewInMemoryAuditLogger(10)
	ctx := context.Background()

	err := logger.Log(ctx, nil)
	if err == nil {
		t.Error("Log(nil) should return error")
	}
}

func TestInMemoryAuditLogger_CircularBuffer(t *testing.T) {
	logger := NewInMemoryAuditLogger(3) // Small buffer
	ctx := context.Background()

	// Add more entries than max size
	for i := 0; i < 5; i++ {
		entry := &AuditLogEntry{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			RequestID:  "req-123",
			Operation:  "test",
			Target:     "target",
			Authorized: true,
		}
		logger.Log(ctx, entry)
	}

	entries := logger.GetAllEntries()
	if len(entries) != 3 {
		t.Errorf("Circular buffer size = %d, want 3", len(entries))
	}
}

func TestInMemoryAuditLogger_Query(t *testing.T) {
	logger := NewInMemoryAuditLogger(100)
	ctx := context.Background()

	// Add test entries
	now := time.Now()
	entries := []*AuditLogEntry{
		{
			Timestamp:  now.Add(-10 * time.Minute),
			AgentID:    "agent-1",
			Operation:  "resource_read",
			Authorized: true,
		},
		{
			Timestamp:  now.Add(-5 * time.Minute),
			AgentID:    "agent-2",
			Operation:  "tool_call",
			Authorized: false,
		},
		{
			Timestamp:  now,
			AgentID:    "agent-1",
			Operation:  "resource_read",
			Authorized: true,
		},
	}

	for _, entry := range entries {
		logger.Log(ctx, entry)
	}

	t.Run("query by agent ID", func(t *testing.T) {
		criteria := &AuditQueryCriteria{
			AgentID: "agent-1",
		}
		results, err := logger.Query(ctx, criteria)
		if err != nil {
			t.Errorf("Query() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Query() returned %d entries, want 2", len(results))
		}
	})

	t.Run("query by operation", func(t *testing.T) {
		criteria := &AuditQueryCriteria{
			Operation: "resource_read",
		}
		results, err := logger.Query(ctx, criteria)
		if err != nil {
			t.Errorf("Query() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Query() returned %d entries, want 2", len(results))
		}
	})

	t.Run("query by authorized status", func(t *testing.T) {
		authorized := false
		criteria := &AuditQueryCriteria{
			Authorized: &authorized,
		}
		results, err := logger.Query(ctx, criteria)
		if err != nil {
			t.Errorf("Query() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Query() returned %d entries, want 1", len(results))
		}
	})

	t.Run("query with time range", func(t *testing.T) {
		criteria := &AuditQueryCriteria{
			StartTime: now.Add(-6 * time.Minute),
			EndTime:   now.Add(1 * time.Minute),
		}
		results, err := logger.Query(ctx, criteria)
		if err != nil {
			t.Errorf("Query() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Query() returned %d entries, want 2", len(results))
		}
	})

	t.Run("query with pagination", func(t *testing.T) {
		criteria := &AuditQueryCriteria{
			Limit:  1,
			Offset: 1,
		}
		results, err := logger.Query(ctx, criteria)
		if err != nil {
			t.Errorf("Query() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Query() returned %d entries, want 1", len(results))
		}
	})

	t.Run("query nil criteria", func(t *testing.T) {
		results, err := logger.Query(ctx, nil)
		if err != nil {
			t.Errorf("Query() error = %v", err)
		}
		if len(results) == 0 {
			t.Error("Query(nil) should return all entries")
		}
	})
}

func TestInMemoryAuditLogger_Close(t *testing.T) {
	logger := NewInMemoryAuditLogger(10)
	err := logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestComputeStatistics(t *testing.T) {
	entries := []*AuditLogEntry{
		{
			AgentID:    "agent-1",
			Operation:  "resource_read",
			Authorized: true,
			Decision:   "granted",
			Duration:   100 * time.Millisecond,
		},
		{
			AgentID:    "agent-1",
			Operation:  "tool_call",
			Authorized: true,
			Decision:   "granted",
			Duration:   200 * time.Millisecond,
		},
		{
			AgentID:    "agent-2",
			Operation:  "resource_read",
			Authorized: false,
			Decision:   "denied",
			Duration:   50 * time.Millisecond,
		},
		{
			AgentID:    "agent-2",
			Operation:  "prompt_get",
			Authorized: false,
			Decision:   "error: invalid token",
			Duration:   10 * time.Millisecond,
		},
	}

	stats := ComputeStatistics(entries)

	if stats.TotalOperations != 4 {
		t.Errorf("TotalOperations = %d, want 4", stats.TotalOperations)
	}

	if stats.AuthorizedCount != 2 {
		t.Errorf("AuthorizedCount = %d, want 2", stats.AuthorizedCount)
	}

	if stats.DeniedCount != 1 {
		t.Errorf("DeniedCount = %d, want 1", stats.DeniedCount)
	}

	if stats.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", stats.ErrorCount)
	}

	expectedAvg := (100 + 200 + 50 + 10) / 4
	if stats.AverageDuration != time.Duration(expectedAvg)*time.Millisecond {
		t.Errorf("AverageDuration = %v, want %v", stats.AverageDuration, time.Duration(expectedAvg)*time.Millisecond)
	}

	if stats.OperationBreakdown["resource_read"] != 2 {
		t.Errorf("OperationBreakdown[resource_read] = %d, want 2", stats.OperationBreakdown["resource_read"])
	}

	if stats.AgentActivityCount["agent-1"] != 2 {
		t.Errorf("AgentActivityCount[agent-1] = %d, want 2", stats.AgentActivityCount["agent-1"])
	}

	if stats.AgentActivityCount["agent-2"] != 2 {
		t.Errorf("AgentActivityCount[agent-2] = %d, want 2", stats.AgentActivityCount["agent-2"])
	}
}

func TestComputeStatistics_Empty(t *testing.T) {
	stats := ComputeStatistics([]*AuditLogEntry{})

	if stats.TotalOperations != 0 {
		t.Errorf("TotalOperations = %d, want 0", stats.TotalOperations)
	}

	if stats.AverageDuration != 0 {
		t.Errorf("AverageDuration = %v, want 0", stats.AverageDuration)
	}
}

func TestNewFileAuditLogger(t *testing.T) {
	logger := NewFileAuditLogger("/tmp/test-audit.log", 50)
	if logger == nil {
		t.Fatal("NewFileAuditLogger() returned nil")
	}
	if logger.maxBatch != 50 {
		t.Errorf("maxBatch = %d, want 50", logger.maxBatch)
	}

	// Test with invalid batch size
	logger2 := NewFileAuditLogger("/tmp/test.log", 0)
	if logger2.maxBatch != 100 {
		t.Errorf("maxBatch with 0 input = %d, want 100 (default)", logger2.maxBatch)
	}
}

func TestFileAuditLogger_Log(t *testing.T) {
	logger := NewFileAuditLogger("/tmp/test-audit.log", 10)
	ctx := context.Background()

	entry := &AuditLogEntry{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		Operation:  "test",
		Authorized: true,
	}

	err := logger.Log(ctx, entry)
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}

	// Check buffer size
	if len(logger.buffer) != 1 {
		t.Errorf("buffer size = %d, want 1", len(logger.buffer))
	}
}

func TestFileAuditLogger_Close(t *testing.T) {
	logger := NewFileAuditLogger("/tmp/test-audit.log", 10)
	ctx := context.Background()

	// Add entry
	entry := &AuditLogEntry{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		Operation:  "test",
		Authorized: true,
	}
	logger.Log(ctx, entry)

	// Close should flush buffer
	err := logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Buffer should be empty after flush
	if len(logger.buffer) != 0 {
		t.Errorf("buffer size after close = %d, want 0", len(logger.buffer))
	}
}
