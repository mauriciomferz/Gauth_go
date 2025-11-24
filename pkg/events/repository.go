package events

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for events
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new events repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// EventTypeRecord represents an event type definition in the database
type EventTypeRecord struct {
	ID             string
	TenantID       string
	EventType      string
	Category       string
	Description    string
	Severity       string
	Schema         string
	RetentionDays  int
	IsSystemEvent  bool
	Count          int // Event count (computed)
	CreatedAt      time.Time
}

// EventRecord represents a system event in the database
type EventRecord struct {
	ID            string
	TenantID      string
	EventType     string
	Category      string
	Severity      string
	Timestamp     time.Time
	Source        string
	UserID        *string
	Resource      *string
	Action        *string
	Status        *string
	Message       *string
	Payload       map[string]interface{}
	IPAddress     *string
	UserAgent     *string
	RequestID     *string
	SessionID     *string
	CorrelationID *string
}

// EventHandlerRecord represents an event handler configuration in the database
type EventHandlerRecord struct {
	ID              string
	TenantID        string
	HandlerName     string
	EventType       string
	HandlerType     string
	Status          string
	EndpointURL     *string
	HTTPMethod      *string
	Headers         map[string]interface{}
	RetryConfig     map[string]interface{}
	TimeoutSeconds  int
	SuccessCount    int
	FailureCount    int
	LastSuccessAt   *time.Time
	LastFailureAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// EventStats represents aggregate statistics for event system
type EventStats struct {
	TotalEvents      int64
	EventsToday      int64
	EventsThisHour   int64
	ByCategory       map[string]int64
	BySeverity       map[string]int64
	TopEventTypes    []EventTypeCount
	HandlerStats     HandlerStats
}

// EventTypeCount represents count for a specific event type
type EventTypeCount struct {
	Type  string
	Count int64
}

// HandlerStats represents handler statistics
type HandlerStats struct {
	TotalHandlers    int
	EnabledHandlers  int
	DisabledHandlers int
	TotalInvocations int64
	AverageSuccessRate float64
}

// ListEventTypes returns all event types for a tenant with event counts
func (r *Repository) ListEventTypes(ctx context.Context, tenantID string) ([]EventTypeRecord, error) {
	query := `
		SELECT 
			et.id, et.tenant_id, et.event_type, et.category, et.description,
			et.severity, et.schema::text, et.retention_days, et.is_system_event,
			et.created_at,
			COALESCE(COUNT(e.id), 0) as event_count
		FROM event_types et
		LEFT JOIN events e ON e.tenant_id = et.tenant_id AND e.event_type = et.event_type
		WHERE et.tenant_id = $1
		GROUP BY et.id, et.tenant_id, et.event_type, et.category, et.description,
		         et.severity, et.schema, et.retention_days, et.is_system_event, et.created_at
		ORDER BY et.category, et.event_type
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query event types: %w", err)
	}
	defer rows.Close()

	var eventTypes []EventTypeRecord
	for rows.Next() {
		var et EventTypeRecord
		var schema *string
		err := rows.Scan(
			&et.ID, &et.TenantID, &et.EventType, &et.Category, &et.Description,
			&et.Severity, &schema, &et.RetentionDays, &et.IsSystemEvent,
			&et.CreatedAt, &et.Count,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event type: %w", err)
		}
		if schema != nil {
			et.Schema = *schema
		}
		eventTypes = append(eventTypes, et)
	}

	return eventTypes, nil
}

// ListEvents returns recent events with optional filtering
func (r *Repository) ListEvents(ctx context.Context, tenantID string, filters EventFilters) ([]EventRecord, error) {
	query := `
		SELECT 
			id, tenant_id, event_type, category, severity, timestamp,
			source, user_id, resource, action, status, message,
			payload, ip_address, user_agent, request_id, session_id, correlation_id
		FROM events
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argPos := 2

	if filters.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, filters.Category)
		argPos++
	}

	if filters.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argPos)
		args = append(args, filters.Severity)
		argPos++
	}

	if filters.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argPos)
		args = append(args, filters.Source)
		argPos++
	}

	if filters.EventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argPos)
		args = append(args, filters.EventType)
		argPos++
	}

	if !filters.StartTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argPos)
		args = append(args, filters.StartTime)
		argPos++
	}

	if !filters.EndTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argPos)
		args = append(args, filters.EndTime)
		argPos++
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
	} else {
		query += " LIMIT 100" // Default limit
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []EventRecord
	for rows.Next() {
		var e EventRecord
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.EventType, &e.Category, &e.Severity, &e.Timestamp,
			&e.Source, &e.UserID, &e.Resource, &e.Action, &e.Status, &e.Message,
			&e.Payload, &e.IPAddress, &e.UserAgent, &e.RequestID, &e.SessionID, &e.CorrelationID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// EventFilters represents filtering options for events
type EventFilters struct {
	Category  string
	Severity  string
	Source    string
	EventType string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// ListHandlers returns all event handlers for a tenant
func (r *Repository) ListHandlers(ctx context.Context, tenantID string) ([]EventHandlerRecord, error) {
	query := `
		SELECT 
			id, tenant_id, handler_name, event_type, handler_type, status,
			endpoint_url, http_method, headers, retry_config, timeout_seconds,
			success_count, failure_count, last_success_at, last_failure_at,
			created_at, updated_at
		FROM event_handlers
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query event handlers: %w", err)
	}
	defer rows.Close()

	var handlers []EventHandlerRecord
	for rows.Next() {
		var h EventHandlerRecord
		err := rows.Scan(
			&h.ID, &h.TenantID, &h.HandlerName, &h.EventType, &h.HandlerType, &h.Status,
			&h.EndpointURL, &h.HTTPMethod, &h.Headers, &h.RetryConfig, &h.TimeoutSeconds,
			&h.SuccessCount, &h.FailureCount, &h.LastSuccessAt, &h.LastFailureAt,
			&h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event handler: %w", err)
		}
		handlers = append(handlers, h)
	}

	return handlers, nil
}

// CreateHandler creates a new event handler
func (r *Repository) CreateHandler(ctx context.Context, h *EventHandlerRecord) error {
	query := `
		INSERT INTO event_handlers (
			tenant_id, handler_name, event_type, handler_type, status,
			endpoint_url, http_method, headers, retry_config, timeout_seconds,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		h.TenantID, h.HandlerName, h.EventType, h.HandlerType, h.Status,
		h.EndpointURL, h.HTTPMethod, h.Headers, h.RetryConfig, h.TimeoutSeconds,
	).Scan(&h.ID, &h.CreatedAt, &h.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create event handler: %w", err)
	}

	return nil
}

// UpdateHandlerStatus updates the status (enabled/disabled) of an event handler
func (r *Repository) UpdateHandlerStatus(ctx context.Context, tenantID, handlerID, status string) error {
	query := `
		UPDATE event_handlers
		SET status = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
	`

	result, err := r.db.Exec(ctx, query, status, tenantID, handlerID)
	if err != nil {
		return fmt.Errorf("failed to update handler status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("event handler not found")
	}

	return nil
}

// DeleteHandler removes an event handler
func (r *Repository) DeleteHandler(ctx context.Context, tenantID, handlerID string) error {
	query := `DELETE FROM event_handlers WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.Exec(ctx, query, tenantID, handlerID)
	if err != nil {
		return fmt.Errorf("failed to delete handler: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("event handler not found")
	}

	return nil
}

// GetEventMetrics returns aggregate metrics for the event system
func (r *Repository) GetEventMetrics(ctx context.Context, tenantID string) (*EventStats, error) {
	// Get total events and time-based counts
	timeQuery := `
		WITH stats AS (
			SELECT 
				COUNT(*) as total_events,
				COUNT(*) FILTER (WHERE timestamp >= CURRENT_DATE) as events_today,
				COUNT(*) FILTER (WHERE timestamp >= DATE_TRUNC('hour', NOW())) as events_this_hour
			FROM events
			WHERE tenant_id = $1
		)
		SELECT total_events, events_today, events_this_hour FROM stats
	`

	var stats EventStats
	err := r.db.QueryRow(ctx, timeQuery, tenantID).Scan(
		&stats.TotalEvents, &stats.EventsToday, &stats.EventsThisHour,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query time stats: %w", err)
	}

	// Get counts by category
	categoryQuery := `
		SELECT category, COUNT(*) as count
		FROM events
		WHERE tenant_id = $1
		GROUP BY category
	`
	rows, err := r.db.Query(ctx, categoryQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query category stats: %w", err)
	}
	defer rows.Close()

	stats.ByCategory = make(map[string]int64)
	for rows.Next() {
		var category string
		var count int64
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category stat: %w", err)
		}
		stats.ByCategory[category] = count
	}

	// Get counts by severity
	severityQuery := `
		SELECT severity, COUNT(*) as count
		FROM events
		WHERE tenant_id = $1
		GROUP BY severity
	`
	rows, err = r.db.Query(ctx, severityQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query severity stats: %w", err)
	}
	defer rows.Close()

	stats.BySeverity = make(map[string]int64)
	for rows.Next() {
		var severity string
		var count int64
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity stat: %w", err)
		}
		stats.BySeverity[severity] = count
	}

	// Get top event types
	topTypesQuery := `
		SELECT event_type, COUNT(*) as count
		FROM events
		WHERE tenant_id = $1
		GROUP BY event_type
		ORDER BY count DESC
		LIMIT 10
	`
	rows, err = r.db.Query(ctx, topTypesQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query top types: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tc EventTypeCount
		if err := rows.Scan(&tc.Type, &tc.Count); err != nil {
			return nil, fmt.Errorf("failed to scan top type: %w", err)
		}
		stats.TopEventTypes = append(stats.TopEventTypes, tc)
	}

	// Get handler statistics
	handlerQuery := `
		SELECT 
			COUNT(*) as total_handlers,
			COUNT(*) FILTER (WHERE status = 'active') as enabled_handlers,
			COUNT(*) FILTER (WHERE status != 'active') as disabled_handlers,
			COALESCE(SUM(success_count + failure_count), 0) as total_invocations,
			CASE 
				WHEN SUM(success_count + failure_count) > 0 
				THEN SUM(success_count)::float / SUM(success_count + failure_count) * 100
				ELSE 0 
			END as avg_success_rate
		FROM event_handlers
		WHERE tenant_id = $1
	`
	err = r.db.QueryRow(ctx, handlerQuery, tenantID).Scan(
		&stats.HandlerStats.TotalHandlers,
		&stats.HandlerStats.EnabledHandlers,
		&stats.HandlerStats.DisabledHandlers,
		&stats.HandlerStats.TotalInvocations,
		&stats.HandlerStats.AverageSuccessRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query handler stats: %w", err)
	}

	return &stats, nil
}

// UpdateHandlerStats updates success/failure counts for a handler
func (r *Repository) UpdateHandlerStats(ctx context.Context, tenantID, handlerID string, success bool) error {
	var query string
	if success {
		query = `
			UPDATE event_handlers
			SET success_count = success_count + 1,
			    last_success_at = NOW(),
			    updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`
	} else {
		query = `
			UPDATE event_handlers
			SET failure_count = failure_count + 1,
			    last_failure_at = NOW(),
			    updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`
	}

	_, err := r.db.Exec(ctx, query, tenantID, handlerID)
	if err != nil {
		return fmt.Errorf("failed to update handler stats: %w", err)
	}

	return nil
}
