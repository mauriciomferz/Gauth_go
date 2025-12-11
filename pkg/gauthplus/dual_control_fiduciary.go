// Package gauthplus - Dual Control and Fiduciary Duty Services
package gauthplus

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	statusPending = "pending"
)

// PostgreSQLDualControlService implements DualControlService using PostgreSQL
type PostgreSQLDualControlService struct {
	db *sql.DB
}

// NewPostgreSQLDualControlService creates a new dual control service
func NewPostgreSQLDualControlService(db *sql.DB) *PostgreSQLDualControlService {
	return &PostgreSQLDualControlService{db: db}
}

// RequestApproval initiates approval workflow
func (s *PostgreSQLDualControlService) RequestApproval(
	ctx context.Context,
	approval *DualControlApproval,
) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not available")
	}

	if approval.ID == "" {
		approval.ID = uuid.New().String()
	}

	approval.CreatedAt = time.Now().UTC()
	approval.UpdatedAt = time.Now().UTC()
	approval.Status = statusPending

	// Set expiration if not provided (default 24 hours)
	if approval.ExpiresAt == nil {
		expiresAt := time.Now().Add(24 * time.Hour).UTC()
		approval.ExpiresAt = &expiresAt
	}

	approvedByJSON, err := json.Marshal(approval.ApprovedBy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal approved_by: %w", err)
	}

	rejectedByJSON, err := json.Marshal(approval.RejectedBy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal rejected_by: %w", err)
	}

	metadataJSON, err := json.Marshal(approval.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dual_control_approvals (
			id, poa_id, action_type, action_description,
			requested_by, requested_at, required_approvers,
			approval_threshold, status, approved_by, rejected_by,
			expires_at, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, approval.ID, approval.POAID, approval.ActionType, approval.ActionDescription,
		approval.RequestedBy, approval.RequestedAt, approval.RequiredApprovers,
		approval.ApprovalThreshold, approval.Status, approvedByJSON, rejectedByJSON,
		approval.ExpiresAt, metadataJSON, approval.CreatedAt, approval.UpdatedAt)

	if err != nil {
		return "", fmt.Errorf("failed to create approval request: %w", err)
	}

	return approval.ID, nil
}

// ApproveAction records approver's approval
func (s *PostgreSQLDualControlService) ApproveAction(
	ctx context.Context,
	approvalID, approverID, comments string,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// Get current approval
	approval, err := s.getApproval(ctx, approvalID)
	if err != nil {
		return err
	}

	if approval.Status != statusPending {
		return fmt.Errorf("approval already finalized with status: %s", approval.Status)
	}

	// Check expiration
	if approval.ExpiresAt != nil && time.Now().UTC().After(*approval.ExpiresAt) {
		return fmt.Errorf("approval request expired")
	}

	// Add approval record
	record := ApprovalRecord{
		ApproverID: approverID,
		ApprovedAt: time.Now().UTC(),
		Comments:   comments,
		Weight:     1, // Default weight
	}

	approval.ApprovedBy = append(approval.ApprovedBy, record)

	// Check if threshold met
	newStatus := s.calculateApprovalStatus(approval)

	approvedByJSON, err := json.Marshal(approval.ApprovedBy)
	if err != nil {
		return fmt.Errorf("failed to marshal approved_by: %w", err)
	}

	now := time.Now().UTC()
	var decisionFinalizedAt *time.Time
	if newStatus != statusPending {
		decisionFinalizedAt = &now
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE dual_control_approvals
		SET approved_by = $1,
		    status = $2,
		    decision_finalized_at = $3,
		    updated_at = $4
		WHERE id = $5
	`, approvedByJSON, newStatus, decisionFinalizedAt, now, approvalID)

	if err != nil {
		return fmt.Errorf("failed to record approval: %w", err)
	}

	return nil
}

// RejectAction records approver's rejection
func (s *PostgreSQLDualControlService) RejectAction(
	ctx context.Context,
	approvalID, approverID, comments string,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// Get current approval
	approval, err := s.getApproval(ctx, approvalID)
	if err != nil {
		return err
	}

	if approval.Status != statusPending {
		return fmt.Errorf("approval already finalized with status: %s", approval.Status)
	}

	// Add rejection record
	record := ApprovalRecord{
		ApproverID: approverID,
		ApprovedAt: time.Now().UTC(),
		Comments:   comments,
		Weight:     1,
	}

	approval.RejectedBy = append(approval.RejectedBy, record)

	// Check if rejection threshold met (any rejection can reject for "all" threshold)
	newStatus := statusPending
	if approval.ApprovalThreshold == "all" {
		newStatus = "rejected"
	} else if len(approval.RejectedBy) > approval.RequiredApprovers/2 {
		newStatus = "rejected"
	}

	rejectedByJSON, err := json.Marshal(approval.RejectedBy)
	if err != nil {
		return fmt.Errorf("failed to marshal rejected_by: %w", err)
	}

	now := time.Now().UTC()
	var decisionFinalizedAt *time.Time
	if newStatus != statusPending {
		decisionFinalizedAt = &now
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE dual_control_approvals
		SET rejected_by = $1,
		    status = $2,
		    decision_finalized_at = $3,
		    updated_at = $4
		WHERE id = $5
	`, rejectedByJSON, newStatus, decisionFinalizedAt, now, approvalID)

	if err != nil {
		return fmt.Errorf("failed to record rejection: %w", err)
	}

	return nil
}

// CheckApprovalStatus checks if threshold met
func (s *PostgreSQLDualControlService) CheckApprovalStatus(
	ctx context.Context,
	approvalID string,
) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not available")
	}

	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT status FROM dual_control_approvals WHERE id = $1
	`, approvalID).Scan(&status)

	if err != nil {
		return "", fmt.Errorf("failed to check approval status: %w", err)
	}

	return status, nil
}

// FindApprovalsByPoAAndAction queries approvals matching PoA and action type
func (s *PostgreSQLDualControlService) FindApprovalsByPoAAndAction(
	ctx context.Context,
	poaID, actionType string,
) ([]*DualControlApproval, error) {
	if s.db == nil {
		return []*DualControlApproval{}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, poa_id, action_type, action_description,
		       requested_by, requested_at, required_approvers,
		       approval_threshold, status, approved_by, rejected_by,
		       expires_at, metadata, created_at, updated_at
		FROM dual_control_approvals
		WHERE poa_id = $1 AND action_type = $2
		ORDER BY requested_at DESC
	`, poaID, actionType)

	if err != nil {
		return nil, fmt.Errorf("failed to find approvals: %w", err)
	}
	defer rows.Close()

	var approvals []*DualControlApproval
	for rows.Next() {
		approval := &DualControlApproval{}
		var approvedByJSON, rejectedByJSON, metadataJSON []byte
		var expiresAt sql.NullTime

		err := rows.Scan(
			&approval.ID, &approval.POAID, &approval.ActionType,
			&approval.ActionDescription, &approval.RequestedBy,
			&approval.RequestedAt, &approval.RequiredApprovers,
			&approval.ApprovalThreshold, &approval.Status,
			&approvedByJSON, &rejectedByJSON, &expiresAt,
			&metadataJSON, &approval.CreatedAt, &approval.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}

		if err := json.Unmarshal(approvedByJSON, &approval.ApprovedBy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal approved_by: %w", err)
		}

		if err := json.Unmarshal(rejectedByJSON, &approval.RejectedBy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rejected_by: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &approval.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		if expiresAt.Valid {
			approval.ExpiresAt = &expiresAt.Time
		}

		approvals = append(approvals, approval)
	}

	return approvals, rows.Err()
}

// GetPendingApprovals returns approvals awaiting decision
func (s *PostgreSQLDualControlService) GetPendingApprovals(
	ctx context.Context,
	approverID string,
) ([]*DualControlApproval, error) {
	if s.db == nil {
		return []*DualControlApproval{}, nil
	}

	// This query would need to check if approverID is authorized
	// For simplicity, returning all pending approvals
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, poa_id, action_type, action_description,
		       requested_by, requested_at, required_approvers,
		       approval_threshold, status, approved_by, rejected_by,
		       expires_at, created_at, updated_at
		FROM dual_control_approvals
		WHERE status = 'pending' AND expires_at > NOW()
		ORDER BY requested_at DESC
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}
	defer rows.Close()

	var approvals []*DualControlApproval
	for rows.Next() {
		approval := &DualControlApproval{}
		var approvedByJSON, rejectedByJSON []byte
		var expiresAt sql.NullTime

		err := rows.Scan(
			&approval.ID, &approval.POAID, &approval.ActionType,
			&approval.ActionDescription, &approval.RequestedBy,
			&approval.RequestedAt, &approval.RequiredApprovers,
			&approval.ApprovalThreshold, &approval.Status,
			&approvedByJSON, &rejectedByJSON, &expiresAt,
			&approval.CreatedAt, &approval.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}

		if err := json.Unmarshal(approvedByJSON, &approval.ApprovedBy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal approved_by: %w", err)
		}

		if err := json.Unmarshal(rejectedByJSON, &approval.RejectedBy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rejected_by: %w", err)
		}

		if expiresAt.Valid {
			approval.ExpiresAt = &expiresAt.Time
		}

		approvals = append(approvals, approval)
	}

	return approvals, rows.Err()
}

// Helper methods

func (s *PostgreSQLDualControlService) getApproval(ctx context.Context, approvalID string) (*DualControlApproval, error) {
	approval := &DualControlApproval{}
	var approvedByJSON, rejectedByJSON []byte
	var expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, poa_id, action_type, action_description,
		       requested_by, requested_at, required_approvers,
		       approval_threshold, status, approved_by, rejected_by,
		       expires_at, created_at, updated_at
		FROM dual_control_approvals
		WHERE id = $1
	`, approvalID).Scan(
		&approval.ID, &approval.POAID, &approval.ActionType,
		&approval.ActionDescription, &approval.RequestedBy,
		&approval.RequestedAt, &approval.RequiredApprovers,
		&approval.ApprovalThreshold, &approval.Status,
		&approvedByJSON, &rejectedByJSON, &expiresAt,
		&approval.CreatedAt, &approval.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	if err := json.Unmarshal(approvedByJSON, &approval.ApprovedBy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approved_by: %w", err)
	}

	if err := json.Unmarshal(rejectedByJSON, &approval.RejectedBy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rejected_by: %w", err)
	}

	if expiresAt.Valid {
		approval.ExpiresAt = &expiresAt.Time
	}

	return approval, nil
}

func (s *PostgreSQLDualControlService) calculateApprovalStatus(approval *DualControlApproval) string {
	approvalCount := len(approval.ApprovedBy)

	switch approval.ApprovalThreshold {
	case "all":
		if approvalCount >= approval.RequiredApprovers {
			return "approved"
		}
	case "majority":
		if approvalCount > approval.RequiredApprovers/2 {
			return "approved"
		}
	case "quorum":
		// Typically 2/3
		if approvalCount >= (approval.RequiredApprovers*2)/3 {
			return "approved"
		}
	case "weighted":
		// Calculate total weight
		totalWeight := 0
		for _, record := range approval.ApprovedBy {
			totalWeight += record.Weight
		}
		if totalWeight >= approval.RequiredApprovers {
			return "approved"
		}
	}

	return "pending"
}

// PostgreSQLFiduciaryDutyService implements FiduciaryDutyService
type PostgreSQLFiduciaryDutyService struct {
	db *sql.DB
}

// NewPostgreSQLFiduciaryDutyService creates a new fiduciary duty service
func NewPostgreSQLFiduciaryDutyService(db *sql.DB) *PostgreSQLFiduciaryDutyService {
	return &PostgreSQLFiduciaryDutyService{db: db}
}

// RecordViolation records a fiduciary duty breach
func (s *PostgreSQLFiduciaryDutyService) RecordViolation(
	ctx context.Context,
	violation *FiduciaryDutyViolation,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	if violation.ID == "" {
		violation.ID = uuid.New().String()
	}

	violation.CreatedAt = time.Now().UTC()
	violation.UpdatedAt = time.Now().UTC()
	violation.ResolutionStatus = "open"

	consequencesJSON, err := json.Marshal(violation.Consequences)
	if err != nil {
		return fmt.Errorf("failed to marshal consequences: %w", err)
	}

	evidenceJSON, err := json.Marshal(violation.Evidence)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO fiduciary_duty_violations (
			id, poa_id, agent_id, duty_type, violation_description,
			severity, detected_at, detected_by, resolution_status,
			consequences, evidence, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, violation.ID, violation.POAID, violation.AgentID, violation.DutyType,
		violation.ViolationDescription, violation.Severity, violation.DetectedAt,
		violation.DetectedBy, violation.ResolutionStatus, consequencesJSON,
		evidenceJSON, violation.CreatedAt, violation.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to record violation: %w", err)
	}

	return nil
}

// GetViolations returns violations for agent or PoA
func (s *PostgreSQLFiduciaryDutyService) GetViolations(
	ctx context.Context,
	poaID, agentID string,
) ([]*FiduciaryDutyViolation, error) {
	if s.db == nil {
		return []*FiduciaryDutyViolation{}, nil
	}

	query := `
		SELECT id, poa_id, agent_id, duty_type, violation_description,
		       severity, detected_at, detected_by, reviewed_by, reviewed_at,
		       resolution_status, resolution_notes, created_at, updated_at
		FROM fiduciary_duty_violations
		WHERE 1=1
	`
	args := []interface{}{}

	if poaID != "" {
		query += " AND poa_id = $" + fmt.Sprint(len(args)+1)
		args = append(args, poaID)
	}

	if agentID != "" {
		query += " AND agent_id = $" + fmt.Sprint(len(args)+1)
		args = append(args, agentID)
	}

	query += " ORDER BY detected_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations: %w", err)
	}
	defer rows.Close()

	var violations []*FiduciaryDutyViolation
	for rows.Next() {
		violation := &FiduciaryDutyViolation{}
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		var resolutionNotes sql.NullString

		err := rows.Scan(
			&violation.ID, &violation.POAID, &violation.AgentID,
			&violation.DutyType, &violation.ViolationDescription,
			&violation.Severity, &violation.DetectedAt, &violation.DetectedBy,
			&reviewedBy, &reviewedAt, &violation.ResolutionStatus,
			&resolutionNotes, &violation.CreatedAt, &violation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan violation: %w", err)
		}

		if reviewedBy.Valid {
			violation.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			violation.ReviewedAt = &reviewedAt.Time
		}
		if resolutionNotes.Valid {
			violation.ResolutionNotes = resolutionNotes.String
		}

		violations = append(violations, violation)
	}

	return violations, rows.Err()
}

// ResolveViolation marks violation as resolved
func (s *PostgreSQLFiduciaryDutyService) ResolveViolation(
	ctx context.Context,
	violationID, reviewedBy, notes string,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE fiduciary_duty_violations
		SET resolution_status = 'resolved',
		    reviewed_by = $1,
		    reviewed_at = $2,
		    resolution_notes = $3,
		    updated_at = $4
		WHERE id = $5
	`, reviewedBy, now, notes, now, violationID)

	if err != nil {
		return fmt.Errorf("failed to resolve violation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no violation found with ID: %s", violationID)
	}

	return nil
}

// GetViolationsBySeverity returns violations above severity threshold
func (s *PostgreSQLFiduciaryDutyService) GetViolationsBySeverity(
	ctx context.Context,
	minSeverity string,
) ([]*FiduciaryDutyViolation, error) {
	if s.db == nil {
		return []*FiduciaryDutyViolation{}, nil
	}

	severityOrder := map[string]int{
		"minor": 1, "moderate": 2, "major": 3, "critical": 4,
	}

	minOrder, ok := severityOrder[minSeverity]
	if !ok {
		return nil, fmt.Errorf("invalid severity: %s", minSeverity)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, poa_id, agent_id, duty_type, violation_description,
		       severity, detected_at, detected_by, resolution_status,
		       created_at, updated_at
		FROM fiduciary_duty_violations
		WHERE CASE severity
			WHEN 'critical' THEN 4
			WHEN 'major' THEN 3
			WHEN 'moderate' THEN 2
			WHEN 'minor' THEN 1
		END >= $1
		ORDER BY detected_at DESC
	`, minOrder)

	if err != nil {
		return nil, fmt.Errorf("failed to get violations by severity: %w", err)
	}
	defer rows.Close()

	var violations []*FiduciaryDutyViolation
	for rows.Next() {
		violation := &FiduciaryDutyViolation{}
		err := rows.Scan(
			&violation.ID, &violation.POAID, &violation.AgentID,
			&violation.DutyType, &violation.ViolationDescription,
			&violation.Severity, &violation.DetectedAt, &violation.DetectedBy,
			&violation.ResolutionStatus, &violation.CreatedAt, &violation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan violation: %w", err)
		}

		violations = append(violations, violation)
	}

	return violations, rows.Err()
}
