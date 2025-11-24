package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles audit trail database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new audit repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// AuditEvent represents an audit event from the database
type AuditEvent struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenantId"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     string                 `json:"eventType"`
	Category      string                 `json:"category"`
	Severity      string                 `json:"severity"`
	UserID        string                 `json:"userId"`
	UserName      string                 `json:"userName"`
	UserRole      string                 `json:"userRole"`
	Action        string                 `json:"action"`
	ResourceType  string                 `json:"resourceType"`
	ResourceID    string                 `json:"resourceId"`
	ResourceName  string                 `json:"resourceName"`
	Status        string                 `json:"status"`
	StatusCode    *int                   `json:"statusCode"`
	ErrorMessage  *string                `json:"errorMessage"`
	IPAddress     string                 `json:"ipAddress"`
	UserAgent     *string                `json:"userAgent"`
	RequestID     *string                `json:"requestId"`
	SessionID     *string                `json:"sessionId"`
	CorrelationID *string                `json:"correlationId"`
	BeforeState   map[string]interface{} `json:"beforeState"`
	AfterState    map[string]interface{} `json:"afterState"`
	Changes       map[string]interface{} `json:"changes"`
	Framework     *string                `json:"complianceFramework"`
	RiskLevel     *string                `json:"riskLevel"`
	RequiresReview bool                  `json:"requiresReview"`
	ReviewedAt    *time.Time             `json:"reviewedAt"`
	ReviewedBy    *string                `json:"reviewedBy"`
	Hash          string                 `json:"hash"`
	PreviousHash  *string                `json:"previousHash"`
}

// ComplianceReport represents a compliance framework report
type ComplianceReport struct {
	ID                  string                 `json:"id"`
	TenantID            string                 `json:"tenantId"`
	ReportName          string                 `json:"reportName"`
	Framework           string                 `json:"framework"`
	PeriodStart         time.Time              `json:"periodStart"`
	PeriodEnd           time.Time              `json:"periodEnd"`
	TotalEvents         int                    `json:"totalEvents"`
	CompliantEvents     int                    `json:"compliantEvents"`
	NonCompliantEvents  int                    `json:"nonCompliantEvents"`
	CriticalViolations  int                    `json:"criticalViolations"`
	Status              string                 `json:"status"`
	GeneratedAt         time.Time              `json:"generatedAt"`
	GeneratedBy         string                 `json:"generatedBy"`
	Summary             *string                `json:"summary"`
	Recommendations     []string               `json:"recommendations"`
	Violations          map[string]interface{} `json:"violations"`
	ReportData          map[string]interface{} `json:"reportData"`
}

// EventCorrelationPattern represents a correlation pattern
type EventCorrelationPattern struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenantId"`
	PatternName     string                 `json:"patternName"`
	PatternType     string                 `json:"patternType"`
	Description     *string                `json:"description"`
	EventSequence   []string               `json:"eventSequence"`
	TimeWindowMin   int                    `json:"timeWindowMinutes"`
	MinOccurrences  int                    `json:"minOccurrences"`
	Conditions      map[string]interface{} `json:"conditions"`
	Severity        string                 `json:"severity"`
	AlertEnabled    bool                   `json:"alertEnabled"`
	AlertRecipients []string               `json:"alertRecipients"`
	MatchesCount    int                    `json:"matchesCount"`
	LastMatchAt     *time.Time             `json:"lastMatchAt"`
	CreatedAt       time.Time              `json:"createdAt"`
}

// SIEMIntegration represents a SIEM integration
type SIEMIntegration struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenantId"`
	IntegrationName   string     `json:"integrationName"`
	SIEMType          string     `json:"siemType"`
	Status            string     `json:"status"`
	EndpointURL       string     `json:"endpointUrl"`
	AuthType          *string    `json:"authType"`
	APIKey            *string    `json:"apiKey"`
	Format            string     `json:"format"`
	BatchSize         int        `json:"batchSize"`
	FlushIntervalSec  int        `json:"flushIntervalSeconds"`
	EventTypes        []string   `json:"eventTypes"`
	MinSeverity       *string    `json:"minSeverity"`
	EventsSent        int64      `json:"eventsSent"`
	LastSyncAt        *time.Time `json:"lastSyncAt"`
	LastError         *string    `json:"lastError"`
	LastErrorAt       *time.Time `json:"lastErrorAt"`
	CreatedAt         time.Time  `json:"createdAt"`
}

// EventFilters represents query filters for audit events
type EventFilters struct {
	TenantID      string
	Category      string
	Severity      string
	UserID        string
	Action        string
	Status        string
	ResourceType  string
	StartTime     *time.Time
	EndTime       *time.Time
	Limit         int
	Offset        int
}

// ListEvents retrieves audit events with filtering
func (r *Repository) ListEvents(ctx context.Context, filters EventFilters) ([]AuditEvent, int, error) {
	query := `
		SELECT 
			id, tenant_id, timestamp, event_type, category, severity,
			user_id, user_name, user_role, action, resource_type, resource_id, resource_name,
			status, status_code, error_message, ip_address, user_agent,
			request_id, session_id, correlation_id,
			before_state, after_state, changes,
			compliance_framework, risk_level, requires_review,
			reviewed_at, reviewed_by, hash, previous_hash
		FROM audit_events
		WHERE tenant_id = $1
	`
	args := []interface{}{filters.TenantID}
	argIndex := 2

	if filters.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, filters.Category)
		argIndex++
	}
	if filters.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argIndex)
		args = append(args, filters.Severity)
		argIndex++
	}
	if filters.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, filters.UserID)
		argIndex++
	}
	if filters.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, filters.Action)
		argIndex++
	}
	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}
	if filters.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		args = append(args, filters.ResourceType)
		argIndex++
	}
	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filters.StartTime)
		argIndex++
	}
	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filters.EndTime)
		argIndex++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS filtered"
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY timestamp DESC"
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	events := []AuditEvent{}
	for rows.Next() {
		var evt AuditEvent
		var beforeState, afterState, changes []byte

		err := rows.Scan(
			&evt.ID, &evt.TenantID, &evt.Timestamp, &evt.EventType, &evt.Category, &evt.Severity,
			&evt.UserID, &evt.UserName, &evt.UserRole, &evt.Action, &evt.ResourceType, &evt.ResourceID, &evt.ResourceName,
			&evt.Status, &evt.StatusCode, &evt.ErrorMessage, &evt.IPAddress, &evt.UserAgent,
			&evt.RequestID, &evt.SessionID, &evt.CorrelationID,
			&beforeState, &afterState, &changes,
			&evt.Framework, &evt.RiskLevel, &evt.RequiresReview,
			&evt.ReviewedAt, &evt.ReviewedBy, &evt.Hash, &evt.PreviousHash,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}

		// Unmarshal JSONB fields
		if beforeState != nil {
			json.Unmarshal(beforeState, &evt.BeforeState)
		}
		if afterState != nil {
			json.Unmarshal(afterState, &evt.AfterState)
		}
		if changes != nil {
			json.Unmarshal(changes, &evt.Changes)
		}

		events = append(events, evt)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating events: %w", err)
	}

	return events, total, nil
}

// CreateEvent inserts a new audit event
func (r *Repository) CreateEvent(ctx context.Context, evt *AuditEvent) error {
	// Get previous hash for chain
	var previousHash *string
	err := r.db.QueryRow(ctx,
		"SELECT hash FROM audit_events WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT 1",
		evt.TenantID,
	).Scan(&previousHash)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to get previous hash: %w", err)
	}

	// Calculate current hash
	hashData := fmt.Sprintf("%s|%s|%s|%s|%s|%v",
		evt.TenantID, evt.Timestamp.Format(time.RFC3339Nano), evt.UserID, evt.Action, evt.ResourceID, previousHash)
	hash := sha256.Sum256([]byte(hashData))
	evt.Hash = hex.EncodeToString(hash[:])
	evt.PreviousHash = previousHash

	// Marshal JSONB fields
	beforeState, _ := json.Marshal(evt.BeforeState)
	afterState, _ := json.Marshal(evt.AfterState)
	changes, _ := json.Marshal(evt.Changes)

	query := `
		INSERT INTO audit_events (
			tenant_id, timestamp, event_type, category, severity,
			user_id, user_name, user_role, action, resource_type, resource_id, resource_name,
			status, status_code, error_message, ip_address, user_agent,
			request_id, session_id, correlation_id,
			before_state, after_state, changes,
			compliance_framework, risk_level, requires_review,
			hash, previous_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		) RETURNING id
	`

	err = r.db.QueryRow(ctx, query,
		evt.TenantID, evt.Timestamp, evt.EventType, evt.Category, evt.Severity,
		evt.UserID, evt.UserName, evt.UserRole, evt.Action, evt.ResourceType, evt.ResourceID, evt.ResourceName,
		evt.Status, evt.StatusCode, evt.ErrorMessage, evt.IPAddress, evt.UserAgent,
		evt.RequestID, evt.SessionID, evt.CorrelationID,
		beforeState, afterState, changes,
		evt.Framework, evt.RiskLevel, evt.RequiresReview,
		evt.Hash, evt.PreviousHash,
	).Scan(&evt.ID)

	if err != nil {
		return fmt.Errorf("failed to create audit event: %w", err)
	}

	return nil
}

// CreateEventsBulk inserts multiple audit events in a batch
func (r *Repository) CreateEventsBulk(ctx context.Context, events []*AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get previous hash
	var previousHash *string
	err = tx.QueryRow(ctx,
		"SELECT hash FROM audit_events WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT 1",
		events[0].TenantID,
	).Scan(&previousHash)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to get previous hash: %w", err)
	}

	for _, evt := range events {
		// Calculate hash chain
		hashData := fmt.Sprintf("%s|%s|%s|%s|%s|%v",
			evt.TenantID, evt.Timestamp.Format(time.RFC3339Nano), evt.UserID, evt.Action, evt.ResourceID, previousHash)
		hash := sha256.Sum256([]byte(hashData))
		evt.Hash = hex.EncodeToString(hash[:])
		evt.PreviousHash = previousHash

		// Marshal JSONB fields
		beforeState, _ := json.Marshal(evt.BeforeState)
		afterState, _ := json.Marshal(evt.AfterState)
		changes, _ := json.Marshal(evt.Changes)

		query := `
			INSERT INTO audit_events (
				tenant_id, timestamp, event_type, category, severity,
				user_id, user_name, user_role, action, resource_type, resource_id, resource_name,
				status, status_code, error_message, ip_address, user_agent,
				request_id, session_id, correlation_id,
				before_state, after_state, changes,
				compliance_framework, risk_level, requires_review,
				hash, previous_hash
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
				$18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
			) RETURNING id
		`

		err = tx.QueryRow(ctx, query,
			evt.TenantID, evt.Timestamp, evt.EventType, evt.Category, evt.Severity,
			evt.UserID, evt.UserName, evt.UserRole, evt.Action, evt.ResourceType, evt.ResourceID, evt.ResourceName,
			evt.Status, evt.StatusCode, evt.ErrorMessage, evt.IPAddress, evt.UserAgent,
			evt.RequestID, evt.SessionID, evt.CorrelationID,
			beforeState, afterState, changes,
			evt.Framework, evt.RiskLevel, evt.RequiresReview,
			evt.Hash, evt.PreviousHash,
		).Scan(&evt.ID)

		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}

		// Update previous hash for next event
		previousHash = &evt.Hash
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// VerifyHashChain verifies the integrity of the audit event hash chain
func (r *Repository) VerifyHashChain(ctx context.Context, tenantID string, eventID string) (bool, error) {
	query := `
		SELECT id, tenant_id, timestamp, user_id, action, resource_id, hash, previous_hash
		FROM audit_events
		WHERE tenant_id = $1 AND id = $2
	`

	var evt struct {
		ID           string
		TenantID     string
		Timestamp    time.Time
		UserID       string
		Action       string
		ResourceID   string
		Hash         string
		PreviousHash *string
	}

	err := r.db.QueryRow(ctx, query, tenantID, eventID).Scan(
		&evt.ID, &evt.TenantID, &evt.Timestamp, &evt.UserID, &evt.Action, &evt.ResourceID, &evt.Hash, &evt.PreviousHash,
	)
	if err != nil {
		return false, fmt.Errorf("failed to query event: %w", err)
	}

	// Recalculate hash
	hashData := fmt.Sprintf("%s|%s|%s|%s|%s|%v",
		evt.TenantID, evt.Timestamp.Format(time.RFC3339Nano), evt.UserID, evt.Action, evt.ResourceID, evt.PreviousHash)
	hash := sha256.Sum256([]byte(hashData))
	calculatedHash := hex.EncodeToString(hash[:])

	return calculatedHash == evt.Hash, nil
}

// ListComplianceReports retrieves compliance reports
func (r *Repository) ListComplianceReports(ctx context.Context, tenantID string) ([]ComplianceReport, error) {
	query := `
		SELECT 
			id, tenant_id, report_name, framework, period_start, period_end,
			total_events, compliant_events, non_compliant_events, critical_violations,
			status, generated_at, generated_by, summary, recommendations, violations, report_data
		FROM compliance_reports
		WHERE tenant_id = $1
		ORDER BY generated_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query compliance reports: %w", err)
	}
	defer rows.Close()

	reports := []ComplianceReport{}
	for rows.Next() {
		var report ComplianceReport
		var recommendations, violations, reportData []byte

		err := rows.Scan(
			&report.ID, &report.TenantID, &report.ReportName, &report.Framework, &report.PeriodStart, &report.PeriodEnd,
			&report.TotalEvents, &report.CompliantEvents, &report.NonCompliantEvents, &report.CriticalViolations,
			&report.Status, &report.GeneratedAt, &report.GeneratedBy, &report.Summary, &recommendations, &violations, &reportData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}

		// Unmarshal JSONB fields
		if recommendations != nil {
			json.Unmarshal(recommendations, &report.Recommendations)
		}
		if violations != nil {
			json.Unmarshal(violations, &report.Violations)
		}
		if reportData != nil {
			json.Unmarshal(reportData, &report.ReportData)
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// GenerateComplianceReport generates a new compliance report
func (r *Repository) GenerateComplianceReport(ctx context.Context, tenantID, framework string, periodStart, periodEnd time.Time, generatedBy string) (*ComplianceReport, error) {
	// Query events for the period
	query := `
		SELECT COUNT(*) as total,
		       COUNT(*) FILTER (WHERE compliance_framework = $2) as compliant,
		       COUNT(*) FILTER (WHERE compliance_framework IS NULL OR compliance_framework != $2) as non_compliant,
		       COUNT(*) FILTER (WHERE severity = 'critical' AND requires_review = true) as critical_violations
		FROM audit_events
		WHERE tenant_id = $1 AND timestamp BETWEEN $3 AND $4
	`

	var total, compliant, nonCompliant, criticalViolations int
	err := r.db.QueryRow(ctx, query, tenantID, framework, periodStart, periodEnd).Scan(
		&total, &compliant, &nonCompliant, &criticalViolations,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate compliance metrics: %w", err)
	}

	// Determine status
	status := "compliant"
	if criticalViolations > 0 {
		status = "non-compliant"
	} else if nonCompliant > total/10 {
		status = "partial"
	}

	// Create report
	report := &ComplianceReport{
		TenantID:           tenantID,
		ReportName:         fmt.Sprintf("%s Compliance Report - %s", framework, periodEnd.Format("2006-01-02")),
		Framework:          framework,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		TotalEvents:        total,
		CompliantEvents:    compliant,
		NonCompliantEvents: nonCompliant,
		CriticalViolations: criticalViolations,
		Status:             status,
		GeneratedAt:        time.Now(),
		GeneratedBy:        generatedBy,
		Recommendations:    []string{},
		Violations:         map[string]interface{}{},
		ReportData:         map[string]interface{}{},
	}

	// Marshal JSONB fields
	recommendations, _ := json.Marshal(report.Recommendations)
	violations, _ := json.Marshal(report.Violations)
	reportData, _ := json.Marshal(report.ReportData)

	insertQuery := `
		INSERT INTO compliance_reports (
			tenant_id, report_name, framework, period_start, period_end,
			total_events, compliant_events, non_compliant_events, critical_violations,
			status, generated_at, generated_by, recommendations, violations, report_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	err = r.db.QueryRow(ctx, insertQuery,
		report.TenantID, report.ReportName, report.Framework, report.PeriodStart, report.PeriodEnd,
		report.TotalEvents, report.CompliantEvents, report.NonCompliantEvents, report.CriticalViolations,
		report.Status, report.GeneratedAt, report.GeneratedBy, recommendations, violations, reportData,
	).Scan(&report.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create compliance report: %w", err)
	}

	return report, nil
}

// ListCorrelationPatterns retrieves event correlation patterns
func (r *Repository) ListCorrelationPatterns(ctx context.Context, tenantID string) ([]EventCorrelationPattern, error) {
	query := `
		SELECT 
			id, tenant_id, pattern_name, pattern_type, description,
			event_sequence, time_window_minutes, min_occurrences, conditions,
			severity, alert_enabled, alert_recipients,
			matches_count, last_match_at, created_at
		FROM event_correlation_patterns
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query correlation patterns: %w", err)
	}
	defer rows.Close()

	patterns := []EventCorrelationPattern{}
	for rows.Next() {
		var pattern EventCorrelationPattern
		var conditions []byte

		err := rows.Scan(
			&pattern.ID, &pattern.TenantID, &pattern.PatternName, &pattern.PatternType, &pattern.Description,
			&pattern.EventSequence, &pattern.TimeWindowMin, &pattern.MinOccurrences, &conditions,
			&pattern.Severity, &pattern.AlertEnabled, &pattern.AlertRecipients,
			&pattern.MatchesCount, &pattern.LastMatchAt, &pattern.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pattern: %w", err)
		}

		// Unmarshal JSONB
		if conditions != nil {
			json.Unmarshal(conditions, &pattern.Conditions)
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// CreateCorrelationPattern creates a new correlation pattern
func (r *Repository) CreateCorrelationPattern(ctx context.Context, pattern *EventCorrelationPattern) error {
	conditions, _ := json.Marshal(pattern.Conditions)

	query := `
		INSERT INTO event_correlation_patterns (
			tenant_id, pattern_name, pattern_type, description,
			event_sequence, time_window_minutes, min_occurrences, conditions,
			severity, alert_enabled, alert_recipients
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		pattern.TenantID, pattern.PatternName, pattern.PatternType, pattern.Description,
		pattern.EventSequence, pattern.TimeWindowMin, pattern.MinOccurrences, conditions,
		pattern.Severity, pattern.AlertEnabled, pattern.AlertRecipients,
	).Scan(&pattern.ID, &pattern.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create correlation pattern: %w", err)
	}

	return nil
}

// ListSIEMIntegrations retrieves SIEM integrations
func (r *Repository) ListSIEMIntegrations(ctx context.Context, tenantID string) ([]SIEMIntegration, error) {
	query := `
		SELECT 
			id, tenant_id, integration_name, siem_type, status,
			endpoint_url, auth_type, api_key, format,
			batch_size, flush_interval_seconds, event_types, min_severity,
			events_sent, last_sync_at, last_error, last_error_at, created_at
		FROM siem_integrations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query SIEM integrations: %w", err)
	}
	defer rows.Close()

	integrations := []SIEMIntegration{}
	for rows.Next() {
		var integration SIEMIntegration

		err := rows.Scan(
			&integration.ID, &integration.TenantID, &integration.IntegrationName, &integration.SIEMType, &integration.Status,
			&integration.EndpointURL, &integration.AuthType, &integration.APIKey, &integration.Format,
			&integration.BatchSize, &integration.FlushIntervalSec, &integration.EventTypes, &integration.MinSeverity,
			&integration.EventsSent, &integration.LastSyncAt, &integration.LastError, &integration.LastErrorAt, &integration.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan SIEM integration: %w", err)
		}

		integrations = append(integrations, integration)
	}

	return integrations, nil
}

// CreateSIEMIntegration creates a new SIEM integration
func (r *Repository) CreateSIEMIntegration(ctx context.Context, integration *SIEMIntegration) error {
	query := `
		INSERT INTO siem_integrations (
			tenant_id, integration_name, siem_type, status,
			endpoint_url, auth_type, api_key, format,
			batch_size, flush_interval_seconds, event_types, min_severity
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		integration.TenantID, integration.IntegrationName, integration.SIEMType, integration.Status,
		integration.EndpointURL, integration.AuthType, integration.APIKey, integration.Format,
		integration.BatchSize, integration.FlushIntervalSec, integration.EventTypes, integration.MinSeverity,
	).Scan(&integration.ID, &integration.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create SIEM integration: %w", err)
	}

	return nil
}

// UpdateSIEMIntegration updates a SIEM integration
func (r *Repository) UpdateSIEMIntegration(ctx context.Context, integration *SIEMIntegration) error {
	query := `
		UPDATE siem_integrations SET
			integration_name = $3, status = $4, endpoint_url = $5,
			auth_type = $6, api_key = $7, format = $8,
			batch_size = $9, flush_interval_seconds = $10,
			event_types = $11, min_severity = $12
		WHERE tenant_id = $1 AND id = $2
	`

	_, err := r.db.Exec(ctx, query,
		integration.TenantID, integration.ID, integration.IntegrationName, integration.Status, integration.EndpointURL,
		integration.AuthType, integration.APIKey, integration.Format,
		integration.BatchSize, integration.FlushIntervalSec, integration.EventTypes, integration.MinSeverity,
	)

	if err != nil {
		return fmt.Errorf("failed to update SIEM integration: %w", err)
	}

	return nil
}

// DeleteSIEMIntegration deletes a SIEM integration
func (r *Repository) DeleteSIEMIntegration(ctx context.Context, tenantID, integrationID string) error {
	query := "DELETE FROM siem_integrations WHERE tenant_id = $1 AND id = $2"

	result, err := r.db.Exec(ctx, query, tenantID, integrationID)
	if err != nil {
		return fmt.Errorf("failed to delete SIEM integration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetAuditMetrics retrieves audit trail metrics
func (r *Repository) GetAuditMetrics(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_events,
			COUNT(*) FILTER (WHERE category = 'auth') as auth_events,
			COUNT(*) FILTER (WHERE category = 'authz') as authz_events,
			COUNT(*) FILTER (WHERE category = 'token') as token_events,
			COUNT(*) FILTER (WHERE category = 'admin') as admin_events,
			COUNT(*) FILTER (WHERE category = 'system') as system_events,
			COUNT(*) FILTER (WHERE severity = 'critical') as critical_events,
			COUNT(*) FILTER (WHERE severity = 'high') as high_events,
			COUNT(*) FILTER (WHERE severity = 'medium') as medium_events,
			COUNT(*) FILTER (WHERE severity = 'low') as low_events,
			COUNT(*) FILTER (WHERE severity = 'info') as info_events,
			COUNT(*) FILTER (WHERE status = 'success') as success_events,
			COUNT(*) FILTER (WHERE status = 'failure') as failure_events,
			COUNT(*) FILTER (WHERE status = 'error') as error_events
		FROM audit_events
		WHERE tenant_id = $1
	`

	var metrics struct {
		Total, Auth, Authz, Token, Admin, System                       int
		Critical, High, Medium, Low, Info                              int
		Success, Failure, Error                                        int
	}

	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&metrics.Total, &metrics.Auth, &metrics.Authz, &metrics.Token, &metrics.Admin, &metrics.System,
		&metrics.Critical, &metrics.High, &metrics.Medium, &metrics.Low, &metrics.Info,
		&metrics.Success, &metrics.Failure, &metrics.Error,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit metrics: %w", err)
	}

	return map[string]interface{}{
		"total_events": metrics.Total,
		"events_by_category": map[string]int{
			"auth":   metrics.Auth,
			"authz":  metrics.Authz,
			"token":  metrics.Token,
			"admin":  metrics.Admin,
			"system": metrics.System,
		},
		"events_by_severity": map[string]int{
			"critical": metrics.Critical,
			"high":     metrics.High,
			"medium":   metrics.Medium,
			"low":      metrics.Low,
			"info":     metrics.Info,
		},
		"events_by_status": map[string]int{
			"success": metrics.Success,
			"failure": metrics.Failure,
			"error":   metrics.Error,
		},
	}, nil
}
