// Package gauthplus - Successor Management & Verification Service Implementation
package gauthplus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mauriciomferz/Gauth_go/pkg/database"
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
		policyBytes, err := json.Marshal(delegation.DelegationPolicy)
		if err != nil {
			return fmt.Errorf("failed to marshal delegation policy: %w", err)
		}
		policyJSON = policyBytes
	}

	_, err = s.db.Pool.Exec(ctx, `
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
	if s.db == nil {
		return nil
	}

	// Check if source agent exists and get their delegation policy
	var policyJSON []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT poa.metadata->>'delegation_policy' 
		FROM poa_records poa
		WHERE poa.representative_id = $1 AND poa.status = 'active'
		LIMIT 1
	`, sourceAgentID).Scan(&policyJSON)

	if err == pgx.ErrNoRows {
		// Possibly no delegation policy defined or no active PoA
		// For safety, assume disallowed if no PoA found, or allow if just no policy?
		// VerificationService should generally require explicit policy for delegation.
		return fmt.Errorf("source agent active PoA or delegation policy not found: %s", sourceAgentID)
	}
	if err != nil {
		return fmt.Errorf("failed to get delegation policy: %w", err)
	}

	if len(policyJSON) == 0 {
		// No policy means default deny for AI delegation usually, but let's say permissive?
		// VerificationService/requirements.go uses default deny if requirements not met.
		// Let's assume default deny.
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
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now().UTC()
	result, err := s.db.Pool.Exec(ctx, `
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

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no active delegation found with ID: %s", delegationID)
	}

	return nil
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

// CheckMaxDepthExceeded checks if adding delegation would exceed max depth
func (s *PostgreSQLDelegationService) CheckMaxDepthExceeded(
	ctx context.Context,
	sourceAgentID string,
	currentDepth int,
) (bool, error) {
	if s.db == nil {
		return false, nil
	}

	var maxDepth int
	var policyJSON []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT metadata->>'delegation_policy'
		FROM poa_records
		WHERE representative_id = $1 AND status = 'active'
		LIMIT 1
	`, sourceAgentID).Scan(&policyJSON)

	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil // No policy -> assume default depth 0 or handled elsewhere
		}
		return false, fmt.Errorf("failed to get max depth: %w", err)
	}
	if len(policyJSON) == 0 {
		return false, nil
	}
	policy, _ := UnmarshalDelegationPolicy(policyJSON)
	if policy != nil {
		maxDepth = policy.MaxDepth
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

// GetPoA gets the PoA by ID
func (s *PostgreSQLPoAStore) GetPoA(ctx context.Context, poaID string) (*EnhancedPoA, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT 
			id, grantor_id, representative_id, 
			scope_type, actions, geographic_regions, 
			status, valid_from, valid_until, 
			revoked_at, revoked_by, revocation_reason,
			created_at, metadata
		FROM poa_records
		WHERE id = $1
	`
	var id, grantorID, representativeID, scopeType, status string
	var actions, regions []string
	var validFrom, validUntil, createdAt time.Time
	var revokedAt *time.Time
	var revokedBy, revocationReason *string
	var metadataJSON []byte

	err := s.db.Pool.QueryRow(ctx, query, poaID).Scan(
		&id, &grantorID, &representativeID,
		&scopeType, &actions, &regions,
		&status, &validFrom, &validUntil,
		&revokedAt, &revokedBy, &revocationReason,
		&createdAt, &metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("PoA not found: %s", poaID)
		}
		return nil, fmt.Errorf("failed to get PoA: %w", err)
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
		Scope:            actions, // Default scope list
		StructuredScope: &StructuredScope{
			Transactions: actions, // Simplification
			Actions:      actions,
		},
		VersionNumber: 1,
	}

	if revokedBy != nil {
		poa.RevokedBy = revokedBy
	}

	// Parse metadata for other fields
	if len(metadataJSON) > 0 {
		var meta map[string]interface{}
		_ = json.Unmarshal(metadataJSON, &meta)
		// Try to extract policies if present
		if p, ok := meta["delegation_policy"]; ok {
			if pb, err := json.Marshal(p); err == nil {
				poa.DelegationPolicy, _ = UnmarshalDelegationPolicy(pb)
			}
		}
	}

	return poa, nil
}

// GetPoAsByGrantee gets PoAs by grantee ID
func (s *PostgreSQLPoAStore) GetPoAsByGrantee(ctx context.Context, granteeID string) ([]*EnhancedPoA, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	// TODO: Implementing simplified version
	return []*EnhancedPoA{}, nil
}

// IsRevoked checks if PoA is revoked
func (s *PostgreSQLPoAStore) IsRevoked(ctx context.Context, poaID string) (bool, *RevocationInfo, error) {
	poa, err := s.GetPoA(ctx, poaID)
	if err != nil {
		return false, nil, err
	}
	if poa.Status == "revoked" {
		info := &RevocationInfo{
			RevokedAt: time.Now(), // Fallback if nil
			Reason:    "unknown",
		}
		if poa.RevokedAt != nil {
			info.RevokedAt = *poa.RevokedAt
		}
		if poa.RevocationReason != nil {
			info.Reason = *poa.RevocationReason
		}
		if poa.RevokedBy != nil {
			info.RevokedBy = *poa.RevokedBy
		}
		return true, info, nil
	}
	return false, nil, nil
}

// StubAttestationVerifier implements AttestationVerifier (Stub)
type StubAttestationVerifier struct{}

func (v *StubAttestationVerifier) Verify(ctx context.Context, attestation Attestation) (bool, error) {
	return true, nil
}

// StubFiduciaryDutyService implements FiduciaryDutyService (Stub)
type StubFiduciaryDutyService struct{}

func (s *StubFiduciaryDutyService) RecordViolation(ctx context.Context, violation *FiduciaryDutyViolation) error {
	return nil
}
func (s *StubFiduciaryDutyService) GetViolations(ctx context.Context, poaID, agentID string) ([]*FiduciaryDutyViolation, error) {
	return []*FiduciaryDutyViolation{}, nil
}
func (s *StubFiduciaryDutyService) ResolveViolation(ctx context.Context, violationID, reviewedBy, notes string) error {
	return nil
}
func (s *StubFiduciaryDutyService) GetViolationsBySeverity(ctx context.Context, minSeverity string) ([]*FiduciaryDutyViolation, error) {
	return []*FiduciaryDutyViolation{}, nil
}

// StubCapabilityService implements CapabilityAssessmentService (Stub)
type StubCapabilityService struct{}

func (s *StubCapabilityService) CreateAssessment(ctx context.Context, assessment *AICapabilityAssessment) error {
	return nil
}
func (s *StubCapabilityService) GetLatestAssessment(ctx context.Context, agentID string) (*AICapabilityAssessment, error) {
	return nil, nil
}
func (s *StubCapabilityService) CheckCapabilityMatch(ctx context.Context, agentID string, requirements *CapabilityRequirements) (bool, []string, error) {
	return true, []string{}, nil
}
func (s *StubCapabilityService) GetExpiringAssessments(ctx context.Context, daysUntilExpiry int) ([]*AICapabilityAssessment, error) {
	return []*AICapabilityAssessment{}, nil
}
