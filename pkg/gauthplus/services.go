// Package gauthplus - Successor Management Service Implementation
package gauthplus

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PostgreSQLSuccessorService implements SuccessorManagementService using PostgreSQL
type PostgreSQLSuccessorService struct {
	db *sql.DB
}

// NewPostgreSQLSuccessorService creates a new successor management service
func NewPostgreSQLSuccessorService(db *sql.DB) *PostgreSQLSuccessorService {
	return &PostgreSQLSuccessorService{db: db}
}

// ActivateSuccessor activates the successor AI when primary fails
func (s *PostgreSQLSuccessorService) ActivateSuccessor(
	ctx context.Context,
	poaID, primaryAgentID, successorAgentID, reason, activatedBy string,
) (*SuccessorActivation, error) {
	// Check if there's already an active successor
	var existingID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM successor_activations 
		WHERE poa_id = $1 AND status = 'active'
		LIMIT 1
	`, poaID).Scan(&existingID)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing activation: %w", err)
	}
	
	if existingID != "" {
		return nil, fmt.Errorf("successor already active for PoA %s", poaID)
	}
	
	activation := &SuccessorActivation{
		ID:               uuid.New().String(),
		POAID:            poaID,
		PrimaryAgentID:   primaryAgentID,
		SuccessorAgentID: successorAgentID,
		ActivationReason: reason,
		ActivatedAt:      time.Now().UTC(),
		ActivatedBy:      activatedBy,
		Status:           "active",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO successor_activations (
			id, poa_id, primary_agent_id, successor_agent_id, 
			activation_reason, activated_at, activated_by, 
			status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, activation.ID, activation.POAID, activation.PrimaryAgentID,
		activation.SuccessorAgentID, activation.ActivationReason,
		activation.ActivatedAt, activation.ActivatedBy,
		activation.Status, activation.CreatedAt, activation.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create successor activation: %w", err)
	}
	
	return activation, nil
}

// DeactivateSuccessor returns control to primary AI
func (s *PostgreSQLSuccessorService) DeactivateSuccessor(
	ctx context.Context,
	activationID, deactivatedBy string,
) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE successor_activations
		SET status = 'deactivated', 
		    deactivated_at = $1, 
		    deactivated_by = $2,
		    updated_at = $3
		WHERE id = $4 AND status = 'active'
	`, now, deactivatedBy, now, activationID)
	
	if err != nil {
		return fmt.Errorf("failed to deactivate successor: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no active successor activation found with ID: %s", activationID)
	}
	
	return nil
}

// GetActiveSuccessor returns current successor activation if any
func (s *PostgreSQLSuccessorService) GetActiveSuccessor(
	ctx context.Context,
	poaID string,
) (*SuccessorActivation, error) {
	activation := &SuccessorActivation{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, poa_id, primary_agent_id, successor_agent_id,
		       activation_reason, activated_at, activated_by,
		       status, created_at, updated_at
		FROM successor_activations
		WHERE poa_id = $1 AND status = 'active'
		LIMIT 1
	`, poaID).Scan(
		&activation.ID, &activation.POAID,
		&activation.PrimaryAgentID, &activation.SuccessorAgentID,
		&activation.ActivationReason, &activation.ActivatedAt,
		&activation.ActivatedBy, &activation.Status,
		&activation.CreatedAt, &activation.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // No active successor
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active successor: %w", err)
	}
	
	return activation, nil
}

// ListSuccessorHistory returns activation history
func (s *PostgreSQLSuccessorService) ListSuccessorHistory(
	ctx context.Context,
	poaID string,
) ([]*SuccessorActivation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, poa_id, primary_agent_id, successor_agent_id,
		       activation_reason, activated_at, activated_by,
		       deactivated_at, deactivated_by, status,
		       created_at, updated_at
		FROM successor_activations
		WHERE poa_id = $1
		ORDER BY activated_at DESC
	`, poaID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to list successor history: %w", err)
	}
	defer rows.Close()
	
	var activations []*SuccessorActivation
	for rows.Next() {
		activation := &SuccessorActivation{}
		var deactivatedAt sql.NullTime
		var deactivatedBy sql.NullString
		
		err := rows.Scan(
			&activation.ID, &activation.POAID,
			&activation.PrimaryAgentID, &activation.SuccessorAgentID,
			&activation.ActivationReason, &activation.ActivatedAt,
			&activation.ActivatedBy, &deactivatedAt, &deactivatedBy,
			&activation.Status, &activation.CreatedAt, &activation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan successor activation: %w", err)
		}
		
		if deactivatedAt.Valid {
			activation.DeactivatedAt = &deactivatedAt.Time
		}
		if deactivatedBy.Valid {
			activation.DeactivatedBy = deactivatedBy.String
		}
		
		activations = append(activations, activation)
	}
	
	return activations, rows.Err()
}

// PostgreSQLDelegationService implements DelegationService using PostgreSQL
type PostgreSQLDelegationService struct {
	db *sql.DB
}

// NewPostgreSQLDelegationService creates a new delegation service
func NewPostgreSQLDelegationService(db *sql.DB) *PostgreSQLDelegationService {
	return &PostgreSQLDelegationService{db: db}
}

// CreateDelegation creates new AI-to-AI delegation
func (s *PostgreSQLDelegationService) CreateDelegation(
	ctx context.Context,
	delegation *AIDelegation,
) error {
	if delegation.ID == "" {
		delegation.ID = uuid.New().String()
	}
	
	delegation.CreatedAt = time.Now().UTC()
	delegation.UpdatedAt = time.Now().UTC()
	
	// Set default status if empty
	if delegation.Status == "" {
		delegation.Status = "active"
	}
	
	// Marshal delegated scope
	scopeJSON, err := json.Marshal(delegation.DelegatedScope)
	if err != nil {
		return fmt.Errorf("failed to marshal delegated scope: %w", err)
	}
	
	// Marshal delegation policy to JSON (can be nil)
	var policyJSON interface{}
	if delegation.DelegationPolicy != nil {
		policyBytes, err := json.Marshal(delegation.DelegationPolicy)
		if err != nil {
			return fmt.Errorf("failed to marshal delegation policy: %w", err)
		}
		policyJSON = string(policyBytes)
	}
	
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO ai_delegations (
			id, source_poa_id, source_agent_id, target_agent_id,
			delegated_scope, delegation_depth, max_allowed_depth,
			valid_from, valid_until, status, delegation_policy,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11, $12, $13)
	`, delegation.ID, delegation.SourcePOAID, delegation.SourceAgentID,
		delegation.TargetAgentID, string(scopeJSON), delegation.DelegationDepth,
		delegation.MaxAllowedDepth, delegation.ValidFrom, delegation.ValidUntil,
		delegation.Status, policyJSON, delegation.CreatedAt, delegation.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create delegation: %w", err)
	}
	
	return nil
}

// ValidateDelegation checks if delegation is allowed per policy
func (s *PostgreSQLDelegationService) ValidateDelegation(
	ctx context.Context,
	sourceAgentID, targetAgentID string,
	scope []string,
	depth int,
) error {
	// Check if source agent exists and get their delegation policy
	var policyJSON []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT poa.delegation_policy 
		FROM power_of_attorney poa
		WHERE poa.representative_id = $1
		LIMIT 1
	`, sourceAgentID).Scan(&policyJSON)
	
	if err == sql.ErrNoRows {
		return fmt.Errorf("source agent not found: %s", sourceAgentID)
	}
	if err != nil {
		return fmt.Errorf("failed to get delegation policy: %w", err)
	}
	
	if policyJSON == nil {
		return fmt.Errorf("no delegation policy defined for agent: %s", sourceAgentID)
	}
	
	policy, err := UnmarshalDelegationPolicy(policyJSON)
	if err != nil {
		return fmt.Errorf("failed to unmarshal policy: %w", err)
	}
	
	// Validate policy rules
	if !policy.CanDelegate {
		return fmt.Errorf("agent %s is not allowed to delegate", sourceAgentID)
	}
	
	if depth > policy.MaxDepth {
		return fmt.Errorf("delegation depth %d exceeds max allowed depth %d", depth, policy.MaxDepth)
	}
	
	if len(policy.AllowedDelegates) > 0 {
		allowed := false
		for _, allowedID := range policy.AllowedDelegates {
			if allowedID == targetAgentID {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("target agent %s not in allowed delegates list", targetAgentID)
		}
	}
	
	return nil
}

// RevokeDelegation revokes an active delegation
func (s *PostgreSQLDelegationService) RevokeDelegation(
	ctx context.Context,
	delegationID, revokedBy, reason string,
) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE ai_delegations
		SET status = 'revoked',
		    revoked_at = $1,
		    revoked_by = $2,
		    revocation_reason = $3,
		    updated_at = $4
		WHERE id = $5 AND status = 'active'
	`, now, revokedBy, reason, now, delegationID)
	
	if err != nil {
		return fmt.Errorf("failed to revoke delegation: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no active delegation found with ID: %s", delegationID)
	}
	
	return nil
}

// GetDelegationChain returns full delegation chain
func (s *PostgreSQLDelegationService) GetDelegationChain(
	ctx context.Context,
	agentID string,
) ([]*AIDelegation, error) {
	// Recursive query to get full chain
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE delegation_chain AS (
			SELECT * FROM ai_delegations
			WHERE target_agent_id = $1 AND status = 'active'
			UNION ALL
			SELECT d.* FROM ai_delegations d
			INNER JOIN delegation_chain dc ON d.target_agent_id = dc.source_agent_id
			WHERE d.status = 'active'
		)
		SELECT id, source_poa_id, source_agent_id, target_agent_id,
		       delegated_scope, delegation_depth, max_allowed_depth,
		       valid_from, valid_until, status, created_at, updated_at
		FROM delegation_chain
		ORDER BY delegation_depth ASC
	`, agentID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get delegation chain: %w", err)
	}
	defer rows.Close()
	
	var delegations []*AIDelegation
	for rows.Next() {
		delegation := &AIDelegation{}
		var scopeJSON []byte
		
		err := rows.Scan(
			&delegation.ID, &delegation.SourcePOAID,
			&delegation.SourceAgentID, &delegation.TargetAgentID,
			&scopeJSON, &delegation.DelegationDepth,
			&delegation.MaxAllowedDepth, &delegation.ValidFrom,
			&delegation.ValidUntil, &delegation.Status,
			&delegation.CreatedAt, &delegation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delegation: %w", err)
		}
		
		if err := json.Unmarshal(scopeJSON, &delegation.DelegatedScope); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
		}
		
		delegations = append(delegations, delegation)
	}
	
	return delegations, rows.Err()
}

// CheckMaxDepthExceeded checks if adding delegation would exceed max depth
func (s *PostgreSQLDelegationService) CheckMaxDepthExceeded(
	ctx context.Context,
	sourceAgentID string,
	currentDepth int,
) (bool, error) {
	var maxDepth int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE((delegation_policy->>'max_depth')::int, 0)
		FROM power_of_attorney
		WHERE representative_id = $1
		LIMIT 1
	`, sourceAgentID).Scan(&maxDepth)
	
	if err != nil {
		return false, fmt.Errorf("failed to get max depth: %w", err)
	}
	
	return currentDepth >= maxDepth, nil
}
