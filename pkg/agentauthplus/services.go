// Package agentauthplus - Successor Management & Verification Service Implementation
package agentauthplus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/database"
)

// PostgreSQLSuccessorService implements SuccessorManagementService using PostgreSQL (pgx)
type PostgreSQLSuccessorService struct {
	db *database.DB
}

// NewPostgreSQLSuccessorService creates a new successor management service
func NewPostgreSQLSuccessorService(db *database.DB) *PostgreSQLSuccessorService {
	return &PostgreSQLSuccessorService{db: db}
}

// ActivateSuccessor activates the successor AI when primary fails
func (s *PostgreSQLSuccessorService) ActivateSuccessor(
	ctx context.Context,
	poaID, primaryAgentID, successorAgentID, reason, activatedBy string,
) (*SuccessorActivation, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Check if there's already an active successor
	var existingID string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id FROM successor_activations 
		WHERE poa_id = $1 AND status = 'active'
		LIMIT 1
	`, poaID).Scan(&existingID)

	if err != nil && err != pgx.ErrNoRows {
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

	_, err = s.db.Pool.Exec(ctx, `
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
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now().UTC()
	result, err := s.db.Pool.Exec(ctx, `
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

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no active successor activation found with ID: %s", activationID)
	}

	return nil
}

// GetActiveSuccessor returns current successor activation if any
func (s *PostgreSQLSuccessorService) GetActiveSuccessor(
	ctx context.Context,
	poaID string,
) (*SuccessorActivation, error) {
	if s.db == nil {
		return nil, nil
	}

	activation := &SuccessorActivation{}
	err := s.db.Pool.QueryRow(ctx, `
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

	if err == pgx.ErrNoRows {
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
	if s.db == nil {
		return []*SuccessorActivation{}, nil
	}

	rows, err := s.db.Pool.Query(ctx, `
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
		var deactivatedAt *time.Time
		var deactivatedBy *string

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

		activation.DeactivatedAt = deactivatedAt
		if deactivatedBy != nil {
			activation.DeactivatedBy = *deactivatedBy
		}

		activations = append(activations, activation)
	}

	return activations, rows.Err()
}

// PostgreSQLDelegationService implements DelegationService using PostgreSQL (pgx)
type PostgreSQLDelegationService struct {
	db *database.DB
}

// NewPostgreSQLDelegationService creates a new delegation service
func NewPostgreSQLDelegationService(db *database.DB) *PostgreSQLDelegationService {
	return &PostgreSQLDelegationService{db: db}
}

// CreateDelegation creates new AI-to-AI delegation
func (s *PostgreSQLDelegationService) CreateDelegation(
	ctx context.Context,
	delegation *AIDelegation,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// 1. Validate source authority and determine depth (RR-014)
	if delegation.SourcePOAID == "" {
		return fmt.Errorf("source_poa_id is required")
	}

	// Fetch parent delegation or root PoA to determine depth and max_depth
	var parentMaxDepth int
	var parentDepth int
	var parentStatus string

	// First try to find a parent delegation for this agent in the same chain
	err := s.db.Pool.QueryRow(ctx, `
		SELECT status, delegation_depth, max_allowed_depth
		FROM ai_delegations
		WHERE target_agent_id = $1 AND source_poa_id = $2 AND status = 'active'
		ORDER BY delegation_depth DESC LIMIT 1
	`, delegation.SourceAgentID, delegation.SourcePOAID).Scan(&parentStatus, &parentDepth, &parentMaxDepth)

	if err == pgx.ErrNoRows {
		// If no parent delegation, check if SourceAgentID is the root grantee of the SourcePOAID
		var poaStatus string
		var metadataJSON []byte

		err = s.db.Pool.QueryRow(ctx, `
			SELECT status, metadata FROM poa_records
			WHERE id = $1 AND representative_id = $2
		`, delegation.SourcePOAID, delegation.SourceAgentID).Scan(&poaStatus, &metadataJSON)

		if err == pgx.ErrNoRows {
			return fmt.Errorf("source agent %s has no active authority for PoA %s", delegation.SourceAgentID, delegation.SourcePOAID)
		}
		if err != nil {
			return fmt.Errorf("failed to verify root authority: %w", err)
		}
		if poaStatus != "active" {
			return fmt.Errorf("root PoA %s is %s", delegation.SourcePOAID, poaStatus)
		}

		// Root PoA found: current agent is at depth 0 (relative to root)
		parentDepth = 0
		// Extract max_depth from PoA metadata
		parentMaxDepth = 5 // Default internal limit
		if len(metadataJSON) > 0 {
			var meta map[string]interface{}
			_ = json.Unmarshal(metadataJSON, &meta)
			if p, ok := meta["delegation_policy"]; ok {
				if policy, ok := p.(map[string]interface{}); ok {
					if md, ok := policy["max_depth"].(float64); ok {
						parentMaxDepth = int(md)
					}
				}
			}
		}
	} else if err != nil {
		return fmt.Errorf("failed to verify parent delegation: %w", err)
	}

	// Calculate and validate new depth
	delegation.DelegationDepth = parentDepth + 1
	delegation.MaxAllowedDepth = parentMaxDepth

	if delegation.DelegationDepth > delegation.MaxAllowedDepth {
		return fmt.Errorf("delegation depth %d exceeds max allowed %d", delegation.DelegationDepth, delegation.MaxAllowedDepth)
	}

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
	var policyJSON []byte
	if delegation.DelegationPolicy != nil {
		policyBytes, marshalErr := json.Marshal(delegation.DelegationPolicy)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal delegation policy: %w", marshalErr)
		}
		policyJSON = policyBytes
	}

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO ai_delegations (
			id, source_poa_id, source_agent_id, target_agent_id, 
			delegated_scope, delegation_depth, max_allowed_depth, 
			valid_from, valid_until, status, delegation_policy, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, delegation.ID, delegation.SourcePOAID, delegation.SourceAgentID,
		delegation.TargetAgentID, scopeJSON, delegation.DelegationDepth,
		delegation.MaxAllowedDepth, delegation.ValidFrom, delegation.ValidUntil,
		delegation.Status, policyJSON, delegation.CreatedAt, delegation.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create delegation: %w", err)
	}

	return nil
}

// ValidateDelegation validates if an agent has valid delegated authority
func (s *PostgreSQLDelegationService) ValidateDelegation(
	ctx context.Context,
	sourceAgentID, targetAgentID string,
	scope []string,
	depth int,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// 1. Check if delegation exists and is active
	var status string
	var maxDepth int
	var delegatedScopeJSON []byte

	err := s.db.Pool.QueryRow(ctx, `
		SELECT status, max_allowed_depth, delegated_scope
		FROM ai_delegations
		WHERE source_agent_id = $1 AND target_agent_id = $2 AND status = 'active'
		LIMIT 1
	`, sourceAgentID, targetAgentID).Scan(&status, &maxDepth, &delegatedScopeJSON)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("no active delegation from %s to %s", sourceAgentID, targetAgentID)
	}
	if err != nil {
		return fmt.Errorf("failed to validate delegation: %w", err)
	}

	// 2. Check depth
	if depth > maxDepth {
		return fmt.Errorf("delegation depth %d exceeds maximum allowed %d", depth, maxDepth)
	}

	// 3. Check scope (simplified)
	var delegatedScope []string
	if err := json.Unmarshal(delegatedScopeJSON, &delegatedScope); err != nil {
		return fmt.Errorf("failed to unmarshal delegated scope: %w", err)
	}

	// Basic check: all requested actions must be in delegated scope
	for _, s := range scope {
		found := false
		for _, ds := range delegatedScope {
			if s == ds {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("action %s not in delegated scope", s)
		}
	}

	return nil
}

// RevokeDelegation revokes an active delegation and cascades to descendants
func (s *PostgreSQLDelegationService) RevokeDelegation(
	ctx context.Context,
	delegationID, revokedBy, reason string,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// 1. Get delegation details for propagation
	var targetAgentID, sourcePoAID string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT target_agent_id, source_poa_id FROM ai_delegations 
		WHERE id = $1 AND status = 'active'
	`, delegationID).Scan(&targetAgentID, &sourcePoAID)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("no active delegation found with ID: %s", delegationID)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch delegation details: %w", err)
	}

	now := time.Now().UTC()
	_, err = s.db.Pool.Exec(ctx, `
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

	// 2. Cascade revocation to all descendants
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		s.propagateStatusUpdate(bgCtx, targetAgentID, sourcePoAID, revokedBy, "Parent Delegation Revoked: "+reason)
	}()

	return nil
}

func (s *PostgreSQLDelegationService) propagateStatusUpdate(ctx context.Context, sourceAgentID, poaID, revokedBy, reason string) {
	// Recursive query to mark all descendants as revoked
	query := `
		WITH RECURSIVE descendants AS (
			SELECT id, target_agent_id FROM ai_delegations
			WHERE source_agent_id = $1 AND source_poa_id = $2 AND status = 'active'
			UNION ALL
			SELECT d.id, d.target_agent_id FROM ai_delegations d
			INNER JOIN descendants desc ON d.source_agent_id = desc.target_agent_id
			WHERE d.source_poa_id = $2 AND d.status = 'active'
		)
		UPDATE ai_delegations
		SET status = 'revoked',
		    revoked_at = $3,
		    revoked_by = $4,
		    revocation_reason = $5,
		    updated_at = $3
		WHERE id IN (SELECT id FROM descendants)
	`
	_, _ = s.db.Pool.Exec(ctx, query, sourceAgentID, poaID, time.Now().UTC(), revokedBy, reason)
}

// GetDelegationChain returns full delegation chain
func (s *PostgreSQLDelegationService) GetDelegationChain(
	ctx context.Context,
	agentID string,
) ([]*AIDelegation, error) {
	if s.db == nil {
		return []*AIDelegation{}, nil
	}

	// Recursive query to get full chain
	rows, err := s.db.Pool.Query(ctx, `
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

// CheckMaxDepthExceeded checks if delegation depth exceeds policy
func (s *PostgreSQLDelegationService) CheckMaxDepthExceeded(
	ctx context.Context,
	sourceAgentID string,
	currentDepth int,
) (bool, error) {
	if s.db == nil {
		return false, nil
	}

	// Find the delegation identifying this agent's policy (simplified)
	var maxDepth int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT max_allowed_depth FROM ai_delegations
		WHERE target_agent_id = $1 AND status = 'active'
		ORDER BY delegation_depth DESC LIMIT 1
	`, sourceAgentID).Scan(&maxDepth)

	if err == pgx.ErrNoRows {
		// If no delegation, check if it's the root agent of a PoA (max depth in policy)
		return false, nil // Assume root for now
	}
	if err != nil {
		return false, fmt.Errorf("failed to check max depth: %w", err)
	}

	return currentDepth >= maxDepth, nil
}

// PostgreSQLPoAStore implements PoAStore using PostgreSQL (pgx)
type PostgreSQLPoAStore struct {
	db *database.DB
}

// NewPostgreSQLPoAStore creates a new PoA Store
func NewPostgreSQLPoAStore(db *database.DB) *PostgreSQLPoAStore {
	return &PostgreSQLPoAStore{db: db}
}

// GetPoA retrieves a specific PoA or AI-to-AI Delegation
func (s *PostgreSQLPoAStore) GetPoA(ctx context.Context, poaID string) (*EnhancedPoA, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// 1. Try to get from poa_records (Human-to-AI or Human-to-Human)
	var id, grantorID, representativeID, status string
	var actions []string
	var validFrom, validUntil, createdAt time.Time
	var revokedAt *time.Time
	var revokedBy, revocationReason *string
	var metadataJSON []byte

	err := s.db.Pool.QueryRow(ctx, `
		SELECT 
			id, grantor_id, representative_id, status, 
			actions, valid_from, valid_until, created_at,
			revoked_at, revoked_by, revocation_reason, metadata
		FROM poa_records
		WHERE id = $1
	`, poaID).Scan(
		&id, &grantorID, &representativeID, &status,
		&actions, &validFrom, &validUntil, &createdAt,
		&revokedAt, &revokedBy, &revocationReason, &metadataJSON,
	)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to query poa_records: %w", err)
	}

	if err == nil {
		poa := &EnhancedPoA{
			ID:               id,
			IssuerID:         grantorID,
			GranteeID:        representativeID,
			Status:           status,
			ValidFrom:        validFrom,
			ValidUntil:       validUntil,
			CreatedAt:        createdAt,
			RevokedAt:        revokedAt,
			RevocationReason: revocationReason,
			Scope:            actions,
			StructuredScope: &StructuredScope{
				Transactions: actions,
				Actions:      actions,
			},
			VersionNumber: 1,
		}
		if revokedBy != nil {
			poa.RevokedBy = revokedBy
		}
		// Parse metadata
		if len(metadataJSON) > 0 {
			var meta map[string]interface{}
			_ = json.Unmarshal(metadataJSON, &meta)
			if p, ok := meta["delegation_policy"]; ok {
				if pb, marshalErr := json.Marshal(p); marshalErr == nil {
					poa.DelegationPolicy, _ = UnmarshalDelegationPolicy(pb)
				}
			}
		}
		return poa, nil
	}

	// 2. Try to get from ai_delegations
	var sourcePoAID string
	err = s.db.Pool.QueryRow(ctx, `
		SELECT 
			id, source_poa_id, source_agent_id, target_agent_id, status, 
			delegated_scope, valid_from, valid_until, created_at,
			revoked_at, revoked_by, revocation_reason
		FROM ai_delegations
		WHERE id = $1
	`, poaID).Scan(
		&id, &sourcePoAID, &grantorID, &representativeID, &status,
		&metadataJSON, &validFrom, &validUntil, &createdAt,
		&revokedAt, &revokedBy, &revocationReason,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("PoA or Delegation not found: %s", poaID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ai_delegations: %w", err)
	}

	// Unmarshal scope from JSONB
	if err := json.Unmarshal(metadataJSON, &actions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal delegated scope: %w", err)
	}

	poa := &EnhancedPoA{
		ID:               id,
		IssuerID:         grantorID,
		GranteeID:        representativeID,
		SourcePOAID:      &sourcePoAID,
		Status:           status,
		ValidFrom:        validFrom,
		ValidUntil:       validUntil,
		CreatedAt:        createdAt,
		RevokedAt:        revokedAt,
		RevocationReason: revocationReason,
		Scope:            actions,
		StructuredScope: &StructuredScope{
			Transactions: actions,
			Actions:      actions,
		},
		VersionNumber: 1,
	}
	if revokedBy != nil {
		poa.RevokedBy = revokedBy
	}

	return poa, nil
}

// GetPoAsByGrantee gets PoAs by grantee ID
func (s *PostgreSQLPoAStore) GetPoAsByGrantee(ctx context.Context, granteeID string) ([]*EnhancedPoA, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT 
			id, grantor_id, representative_id, 
			status, actions, 
			valid_from, valid_until, created_at,
			revoked_at, revoked_by, revocation_reason, metadata
		FROM poa_records
		WHERE representative_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query, granteeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PoAs by grantee: %w", err)
	}
	defer rows.Close()

	var poas []*EnhancedPoA
	for rows.Next() {
		var id, grantorID, representativeID, status string
		var actions []string
		var validFrom, validUntil, createdAt time.Time
		var revokedAt *time.Time
		var revokedBy, revocationReason *string
		var metadataJSON []byte

		err := rows.Scan(
			&id, &grantorID, &representativeID, &status, &actions,
			&validFrom, &validUntil, &createdAt,
			&revokedAt, &revokedBy, &revocationReason, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PoA: %w", err)
		}

		poa := &EnhancedPoA{
			ID:               id,
			IssuerID:         grantorID,
			GranteeID:        representativeID,
			Status:           status,
			ValidFrom:        validFrom,
			ValidUntil:       validUntil,
			CreatedAt:        createdAt,
			RevokedAt:        revokedAt,
			RevocationReason: revocationReason,
			Scope:            actions,
			StructuredScope: &StructuredScope{
				Transactions: actions,
				Actions:      actions,
			},
			VersionNumber: 1,
		}
		if revokedBy != nil {
			poa.RevokedBy = revokedBy
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			var meta map[string]interface{}
			_ = json.Unmarshal(metadataJSON, &meta)
			if p, ok := meta["delegation_policy"]; ok {
				if pb, marshalErr := json.Marshal(p); marshalErr == nil {
					poa.DelegationPolicy, _ = UnmarshalDelegationPolicy(pb)
				}
			}
		}

		poas = append(poas, poa)
	}

	return poas, nil
}

// IsRevoked checks if PoA is revoked
func (s *PostgreSQLPoAStore) IsRevoked(ctx context.Context, poaID string) (bool, *RevocationInfo, error) {
	if s.db == nil {
		return false, nil, fmt.Errorf("database not available")
	}

	var status string
	var revokedAt *time.Time
	var revokedBy, revocationReason *string

	err := s.db.Pool.QueryRow(ctx, `
		SELECT status, revoked_at, revoked_by, revocation_reason 
		FROM poa_records WHERE id = $1
	`, poaID).Scan(&status, &revokedAt, &revokedBy, &revocationReason)

	if err != nil && err != pgx.ErrNoRows {
		// Also check ai_delegations
		err = s.db.Pool.QueryRow(ctx, `
			SELECT status, revoked_at, revoked_by, revocation_reason 
			FROM ai_delegations WHERE id = $1
		`, poaID).Scan(&status, &revokedAt, &revokedBy, &revocationReason)
	}

	if err == pgx.ErrNoRows {
		return false, nil, fmt.Errorf("PoA or Delegation not found: %s", poaID)
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check revocation status: %w", err)
	}

	if status == "revoked" && revokedAt != nil {
		info := &RevocationInfo{
			RevokedAt: *revokedAt,
		}
		if revokedBy != nil {
			info.RevokedBy = *revokedBy
		}
		if revocationReason != nil {
			info.Reason = *revocationReason
		}
		return true, info, nil
	}

	return false, nil, nil
}

// RevokePoA revokes an active PoA and triggers cascade revocation
func (s *PostgreSQLPoAStore) RevokePoA(ctx context.Context, poaID, revokedBy, reason string) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	// 1. Get PoA to know the grantee (who will have delegated powers revoked)
	poa, err := s.GetPoA(ctx, poaID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	_, err = s.db.Pool.Exec(ctx, `
		UPDATE poa_records
		SET status = 'revoked',
		    revoked_at = $1,
		    revoked_by = $2,
		    revocation_reason = $3
		WHERE id = $4 AND status = 'active'
	`, now, revokedBy, reason, poaID)

	if err != nil {
		return fmt.Errorf("failed to revoke PoA Record: %w", err)
	}

	// 2. Trigger cascade revocation of all AI-to-AI delegations branching from this PoA
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		s.propagateStatusUpdate(bgCtx, poa.GranteeID, poaID, revokedBy, "Parent PoA Revoked: "+reason)
	}()

	return nil
}

func (s *PostgreSQLPoAStore) propagateStatusUpdate(ctx context.Context, sourceAgentID, poaID, revokedBy, reason string) {
	// Optimized recursive update for cascade revocation
	query := `
		WITH RECURSIVE descendants AS (
			SELECT id, target_agent_id FROM ai_delegations
			WHERE source_agent_id = $1 AND source_poa_id = $2 AND status = 'active'
			UNION ALL
			SELECT d.id, d.target_agent_id FROM ai_delegations d
			INNER JOIN descendants desc ON d.source_agent_id = desc.target_agent_id
			WHERE d.source_poa_id = $2 AND d.status = 'active'
		)
		UPDATE ai_delegations
		SET status = 'revoked',
		    revoked_at = $3,
		    revoked_by = $4,
		    revocation_reason = $5,
		    updated_at = $3
		WHERE id IN (SELECT id FROM descendants)
	`
	_, _ = s.db.Pool.Exec(ctx, query, sourceAgentID, poaID, time.Now().UTC(), revokedBy, reason)
}
