// Package mcp - Audit Logger for MCP Operations
// Comprehensive logging of all MCP operations for compliance and security
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditLogEntry represents a single MCP operation audit event
type AuditLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	AgentID     string                 `json:"agent_id"`
	RequestID   string                 `json:"request_id"`
	GrantID     string                 `json:"grant_id"`
	Operation   string                 `json:"operation"` // "resource_read", "tool_call", "prompt_get"
	Target      string                 `json:"target"`    // resource URI, tool name, or prompt name
	Authorized  bool                   `json:"authorized"`
	Decision    string                 `json:"decision"` // "granted", "denied", or error reason
	Duration    time.Duration          `json:"duration,omitempty"`
	MCPServerID string                 `json:"mcp_server_id,omitempty"`
	TokenScopes []string               `json:"token_scopes,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger provides comprehensive audit logging for MCP operations
type AuditLogger interface {
	// Log records an MCP operation audit entry
	Log(ctx context.Context, entry *AuditLogEntry) error

	// Query retrieves audit entries matching criteria
	Query(ctx context.Context, criteria *AuditQueryCriteria) ([]*AuditLogEntry, error)

	// Close closes the audit logger and flushes any pending entries
	Close() error
}

// AuditQueryCriteria defines search parameters for audit log queries
type AuditQueryCriteria struct {
	AgentID    string
	Operation  string
	Authorized *bool
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

// InMemoryAuditLogger implements AuditLogger with in-memory storage
// For production, use database-backed implementation
type InMemoryAuditLogger struct {
	mu      sync.RWMutex
	entries []*AuditLogEntry
	maxSize int
}

// NewInMemoryAuditLogger creates a new in-memory audit logger
func NewInMemoryAuditLogger(maxSize int) *InMemoryAuditLogger {
	if maxSize <= 0 {
		maxSize = 10000 // Default max 10k entries
	}
	return &InMemoryAuditLogger{
		entries: make([]*AuditLogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Log records an audit entry
func (l *InMemoryAuditLogger) Log(ctx context.Context, entry *AuditLogEntry) error {
	if entry == nil {
		return fmt.Errorf("audit entry cannot be nil")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Enforce max size with circular buffer
	if len(l.entries) >= l.maxSize {
		// Remove oldest entry
		l.entries = l.entries[1:]
	}

	l.entries = append(l.entries, entry)
	return nil
}

// Query retrieves audit entries matching criteria
func (l *InMemoryAuditLogger) Query(ctx context.Context, criteria *AuditQueryCriteria) ([]*AuditLogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []*AuditLogEntry

	for _, entry := range l.entries {
		if l.matchesCriteria(entry, criteria) {
			results = append(results, entry)
		}
	}

	// Apply pagination
	if criteria != nil {
		offset := criteria.Offset
		limit := criteria.Limit
		if limit == 0 {
			limit = 100 // Default limit
		}

		if offset >= len(results) {
			return []*AuditLogEntry{}, nil
		}

		end := offset + limit
		if end > len(results) {
			end = len(results)
		}

		results = results[offset:end]
	}

	return results, nil
}

// matchesCriteria checks if entry matches query criteria
func (l *InMemoryAuditLogger) matchesCriteria(entry *AuditLogEntry, criteria *AuditQueryCriteria) bool {
	if criteria == nil {
		return true
	}

	if criteria.AgentID != "" && entry.AgentID != criteria.AgentID {
		return false
	}

	if criteria.Operation != "" && entry.Operation != criteria.Operation {
		return false
	}

	if criteria.Authorized != nil && entry.Authorized != *criteria.Authorized {
		return false
	}

	if !criteria.StartTime.IsZero() && entry.Timestamp.Before(criteria.StartTime) {
		return false
	}

	if !criteria.EndTime.IsZero() && entry.Timestamp.After(criteria.EndTime) {
		return false
	}

	return true
}

// Close closes the audit logger
func (l *InMemoryAuditLogger) Close() error {
	// In-memory logger doesn't need cleanup
	return nil
}

// GetAllEntries returns all audit entries (for testing/debugging)
func (l *InMemoryAuditLogger) GetAllEntries() []*AuditLogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]*AuditLogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// FileAuditLogger implements AuditLogger with file-based storage
type FileAuditLogger struct {
	filePath string
	mu       sync.Mutex
	buffer   []*AuditLogEntry
	maxBatch int
}

// NewFileAuditLogger creates a new file-based audit logger
func NewFileAuditLogger(filePath string, maxBatch int) *FileAuditLogger {
	if maxBatch <= 0 {
		maxBatch = 100
	}
	return &FileAuditLogger{
		filePath: filePath,
		buffer:   make([]*AuditLogEntry, 0, maxBatch),
		maxBatch: maxBatch,
	}
}

// Log records an audit entry (buffered)
func (l *FileAuditLogger) Log(ctx context.Context, entry *AuditLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buffer = append(l.buffer, entry)

	// Flush if buffer full
	if len(l.buffer) >= l.maxBatch {
		return l.flush()
	}

	return nil
}

// flush writes buffered entries to file
func (l *FileAuditLogger) flush() error {
	if len(l.buffer) == 0 {
		return nil
	}

	// Convert to JSON Lines format (one JSON object per line)
	data := make([]byte, 0)
	for _, entry := range l.buffer {
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal audit entry: %w", err)
		}
		data = append(data, jsonBytes...)
		data = append(data, '\n')
	}

	// In production, append to file atomically
	// For now, this is a stub implementation
	_ = data

	l.buffer = l.buffer[:0]
	return nil
}

// Query retrieves audit entries (stub - requires file parsing)
func (l *FileAuditLogger) Query(ctx context.Context, criteria *AuditQueryCriteria) ([]*AuditLogEntry, error) {
	return nil, fmt.Errorf("file-based query not implemented")
}

// Close flushes pending entries and closes logger
func (l *FileAuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flush()
}

// AuditStatistics provides summary statistics for audit logs
type AuditStatistics struct {
	TotalOperations    int64
	AuthorizedCount    int64
	DeniedCount        int64
	ErrorCount         int64
	AverageDuration    time.Duration
	OperationBreakdown map[string]int64
	AgentActivityCount map[string]int64
}

// ComputeStatistics calculates audit statistics from entries
func ComputeStatistics(entries []*AuditLogEntry) *AuditStatistics {
	stats := &AuditStatistics{
		OperationBreakdown: make(map[string]int64),
		AgentActivityCount: make(map[string]int64),
	}

	var totalDuration time.Duration

	for _, entry := range entries {
		stats.TotalOperations++

		if entry.Authorized {
			stats.AuthorizedCount++
		} else {
			if entry.Decision == "denied" {
				stats.DeniedCount++
			} else {
				stats.ErrorCount++
			}
		}

		stats.OperationBreakdown[entry.Operation]++
		stats.AgentActivityCount[entry.AgentID]++
		totalDuration += entry.Duration
	}

	if stats.TotalOperations > 0 {
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalOperations)
	}

	return stats
}
