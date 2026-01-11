// Package agentauth - Database Policy Store Implementation
// AAP001 Section 3.1 - P*P Architecture
package agentauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// DatabasePolicyStore provides a PostgreSQL-backed implementation of PolicyStore.
// This implementation is suitable for production environments requiring persistence
// and scalability.
type DatabasePolicyStore struct {
	db *sql.DB
}

// NewDatabasePolicyStore creates a new database-backed policy store
func NewDatabasePolicyStore(db *sql.DB) (*DatabasePolicyStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	store := &DatabasePolicyStore{db: db}

	// Initialize schema if needed
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the policies table if it doesn't exist
func (s *DatabasePolicyStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS authorization_policies (
		policy_id VARCHAR(255) PRIMARY KEY,
		policy_type VARCHAR(50) NOT NULL,
		policy_version INTEGER NOT NULL DEFAULT 1,
		policy_name VARCHAR(255) NOT NULL,
		description TEXT,
		status VARCHAR(50) NOT NULL,
		created_by VARCHAR(255),
		owners_authorizer VARCHAR(255),
		client_owner VARCHAR(255),
		organization_id VARCHAR(255),
		policy_rules JSONB NOT NULL,
		scope JSONB,
		restrictions JSONB,
		poa_template JSONB,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
		activated_at TIMESTAMP WITH TIME ZONE,
		expires_at TIMESTAMP WITH TIME ZONE,
		revoked_at TIMESTAMP WITH TIME ZONE,
		previous_version VARCHAR(255),
		change_log TEXT,
		enforcement_count BIGINT DEFAULT 0,
		last_enforced_at TIMESTAMP WITH TIME ZONE,
		violation_count BIGINT DEFAULT 0,
		last_violation_at TIMESTAMP WITH TIME ZONE,
		tags JSONB,
		metadata JSONB,
		created_at_index TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_policies_status ON authorization_policies(status);
	CREATE INDEX IF NOT EXISTS idx_policies_type ON authorization_policies(policy_type);
	CREATE INDEX IF NOT EXISTS idx_policies_client_owner ON authorization_policies(client_owner);
	CREATE INDEX IF NOT EXISTS idx_policies_org ON authorization_policies(organization_id);
	CREATE INDEX IF NOT EXISTS idx_policies_created_at ON authorization_policies(created_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Create stores a new policy
func (s *DatabasePolicyStore) Create(ctx context.Context, policy *AuthorizationPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}
	if policy.PolicyID == "" {
		return fmt.Errorf("policy ID is required")
	}

	// Marshal JSON fields
	policyRulesJSON, err := json.Marshal(policy.PolicyRules)
	if err != nil {
		return fmt.Errorf("failed to marshal policy rules: %w", err)
	}

	var scopeJSON, restrictionsJSON, poaTemplateJSON, tagsJSON, metadataJSON []byte
	if policy.Scope != nil {
		scopeJSON, err = json.Marshal(policy.Scope)
		if err != nil {
			return fmt.Errorf("failed to marshal scope: %w", err)
		}
	}
	if policy.Restrictions != nil {
		restrictionsJSON, err = json.Marshal(policy.Restrictions)
		if err != nil {
			return fmt.Errorf("failed to marshal restrictions: %w", err)
		}
	}
	if policy.PoATemplate != nil {
		poaTemplateJSON, err = json.Marshal(policy.PoATemplate)
		if err != nil {
			return fmt.Errorf("failed to marshal PoA template: %w", err)
		}
	}
	if policy.Tags != nil {
		tagsJSON, err = json.Marshal(policy.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
	}
	if policy.Metadata != nil {
		metadataJSON, err = json.Marshal(policy.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO authorization_policies (
			policy_id, policy_type, policy_version, policy_name, description,
			status, created_by, owners_authorizer, client_owner, organization_id,
			policy_rules, scope, restrictions, poa_template,
			created_at, updated_at, activated_at, expires_at, revoked_at,
			previous_version, change_log, enforcement_count, last_enforced_at,
			tags, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
				  $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`

	_, err = s.db.ExecContext(ctx, query,
		policy.PolicyID, policy.PolicyType, policy.PolicyVersion, policy.PolicyName, policy.Description,
		policy.Status, policy.CreatedBy, policy.OwnersAuthorizer, policy.ClientOwner, policy.OrganizationID,
		policyRulesJSON, scopeJSON, restrictionsJSON, poaTemplateJSON,
		policy.CreatedAt, policy.UpdatedAt, policy.ActivatedAt, policy.ExpiresAt, policy.RevokedAt,
		policy.PreviousVersion, policy.ChangeLog, policy.EnforcementCount, policy.LastEnforcedAt,
		tagsJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// Get retrieves a policy by ID
func (s *DatabasePolicyStore) Get(ctx context.Context, policyID string) (*AuthorizationPolicy, error) {
	query := `
		SELECT policy_id, policy_type, policy_version, policy_name, description,
			   status, created_by, owners_authorizer, client_owner, organization_id,
			   policy_rules, scope, restrictions, poa_template,
			   created_at, updated_at, activated_at, expires_at, revoked_at,
			   previous_version, change_log, enforcement_count, last_enforced_at,
			   tags, metadata
		FROM authorization_policies
		WHERE policy_id = $1
	`

	policy := &AuthorizationPolicy{}
	var policyRulesJSON, scopeJSON, restrictionsJSON, poaTemplateJSON, tagsJSON, metadataJSON []byte

	err := s.db.QueryRowContext(ctx, query, policyID).Scan(
		&policy.PolicyID, &policy.PolicyType, &policy.PolicyVersion, &policy.PolicyName, &policy.Description,
		&policy.Status, &policy.CreatedBy, &policy.OwnersAuthorizer, &policy.ClientOwner, &policy.OrganizationID,
		&policyRulesJSON, &scopeJSON, &restrictionsJSON, &poaTemplateJSON,
		&policy.CreatedAt, &policy.UpdatedAt, &policy.ActivatedAt, &policy.ExpiresAt, &policy.RevokedAt,
		&policy.PreviousVersion, &policy.ChangeLog, &policy.EnforcementCount, &policy.LastEnforcedAt,
		&tagsJSON, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(policyRulesJSON, &policy.PolicyRules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy rules: %w", err)
	}
	if len(scopeJSON) > 0 {
		if err := json.Unmarshal(scopeJSON, &policy.Scope); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
		}
	}
	if len(restrictionsJSON) > 0 {
		if err := json.Unmarshal(restrictionsJSON, &policy.Restrictions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal restrictions: %w", err)
		}
	}
	if len(poaTemplateJSON) > 0 {
		if err := json.Unmarshal(poaTemplateJSON, &policy.PoATemplate); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PoA template: %w", err)
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &policy.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return policy, nil
}

// Update updates an existing policy
func (s *DatabasePolicyStore) Update(ctx context.Context, policy *AuthorizationPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}
	if policy.PolicyID == "" {
		return fmt.Errorf("policy ID is required")
	}

	// Marshal JSON fields
	policyRulesJSON, err := json.Marshal(policy.PolicyRules)
	if err != nil {
		return fmt.Errorf("failed to marshal policy rules: %w", err)
	}

	var scopeJSON, restrictionsJSON, poaTemplateJSON, tagsJSON, metadataJSON []byte
	if policy.Scope != nil {
		scopeJSON, err = json.Marshal(policy.Scope)
		if err != nil {
			return fmt.Errorf("failed to marshal scope: %w", err)
		}
	}
	if policy.Restrictions != nil {
		restrictionsJSON, err = json.Marshal(policy.Restrictions)
		if err != nil {
			return fmt.Errorf("failed to marshal restrictions: %w", err)
		}
	}
	if policy.PoATemplate != nil {
		poaTemplateJSON, err = json.Marshal(policy.PoATemplate)
		if err != nil {
			return fmt.Errorf("failed to marshal PoA template: %w", err)
		}
	}
	if policy.Tags != nil {
		tagsJSON, err = json.Marshal(policy.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
	}
	if policy.Metadata != nil {
		metadataJSON, err = json.Marshal(policy.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE authorization_policies SET
			policy_type = $2, policy_version = $3, policy_name = $4, description = $5,
			status = $6, created_by = $7, owners_authorizer = $8, client_owner = $9,
			organization_id = $10, policy_rules = $11, scope = $12, restrictions = $13,
			poa_template = $14, updated_at = $15, activated_at = $16, expires_at = $17,
			revoked_at = $18, previous_version = $19, change_log = $20,
			enforcement_count = $21, last_enforced_at = $22, tags = $23, metadata = $24
		WHERE policy_id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
		policy.PolicyID, policy.PolicyType, policy.PolicyVersion, policy.PolicyName, policy.Description,
		policy.Status, policy.CreatedBy, policy.OwnersAuthorizer, policy.ClientOwner, policy.OrganizationID,
		policyRulesJSON, scopeJSON, restrictionsJSON, poaTemplateJSON,
		policy.UpdatedAt, policy.ActivatedAt, policy.ExpiresAt, policy.RevokedAt,
		policy.PreviousVersion, policy.ChangeLog, policy.EnforcementCount, policy.LastEnforcedAt,
		tagsJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found: %s", policy.PolicyID)
	}

	return nil
}

// Delete removes a policy from storage
func (s *DatabasePolicyStore) Delete(ctx context.Context, policyID string) error {
	query := `DELETE FROM authorization_policies WHERE policy_id = $1`

	result, err := s.db.ExecContext(ctx, query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	return nil
}

// List returns all policies, optionally filtered by status
func (s *DatabasePolicyStore) List(ctx context.Context, status *PolicyStatus) ([]*AuthorizationPolicy, error) {
	var query string
	var args []interface{}

	if status != nil {
		query = `
			SELECT policy_id, policy_type, policy_version, policy_name, description,
				   status, created_by, owners_authorizer, client_owner, organization_id,
				   policy_rules, scope, restrictions, poa_template,
				   created_at, updated_at, activated_at, expires_at, revoked_at,
				   previous_version, change_log, enforcement_count, last_enforced_at,
				   tags, metadata
			FROM authorization_policies
			WHERE status = $1
			ORDER BY created_at DESC
		`
		args = append(args, *status)
	} else {
		query = `
			SELECT policy_id, policy_type, policy_version, policy_name, description,
				   status, created_by, owners_authorizer, client_owner, organization_id,
				   policy_rules, scope, restrictions, poa_template,
				   created_at, updated_at, activated_at, expires_at, revoked_at,
				   previous_version, change_log, enforcement_count, last_enforced_at,
				   tags, metadata
			FROM authorization_policies
			ORDER BY created_at DESC
		`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var policies []*AuthorizationPolicy

	for rows.Next() {
		policy := &AuthorizationPolicy{}
		var policyRulesJSON, scopeJSON, restrictionsJSON, poaTemplateJSON, tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&policy.PolicyID, &policy.PolicyType, &policy.PolicyVersion, &policy.PolicyName, &policy.Description,
			&policy.Status, &policy.CreatedBy, &policy.OwnersAuthorizer, &policy.ClientOwner, &policy.OrganizationID,
			&policyRulesJSON, &scopeJSON, &restrictionsJSON, &poaTemplateJSON,
			&policy.CreatedAt, &policy.UpdatedAt, &policy.ActivatedAt, &policy.ExpiresAt, &policy.RevokedAt,
			&policy.PreviousVersion, &policy.ChangeLog, &policy.EnforcementCount, &policy.LastEnforcedAt,
			&tagsJSON, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(policyRulesJSON, &policy.PolicyRules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal policy rules: %w", err)
		}
		if len(scopeJSON) > 0 {
			if err := json.Unmarshal(scopeJSON, &policy.Scope); err != nil {
				return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
			}
		}
		if len(restrictionsJSON) > 0 {
			if err := json.Unmarshal(restrictionsJSON, &policy.Restrictions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal restrictions: %w", err)
			}
		}
		if len(poaTemplateJSON) > 0 {
			if err := json.Unmarshal(poaTemplateJSON, &policy.PoATemplate); err != nil {
				return nil, fmt.Errorf("failed to unmarshal PoA template: %w", err)
			}
		}
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &policy.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %w", err)
	}

	return policies, nil
}

// Search finds policies matching the given criteria
func (s *DatabasePolicyStore) Search(ctx context.Context, criteria *PolicySearchCriteria) ([]*AuthorizationPolicy, error) {
	query, args := s.buildSearchQuery(criteria)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search policies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var policies []*AuthorizationPolicy

	for rows.Next() {
		policy, err := s.scanPolicyRows(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %w", err)
	}

	return policies, nil
}

func (s *DatabasePolicyStore) buildSearchQuery(criteria *PolicySearchCriteria) (string, []interface{}) {
	query := `
		SELECT policy_id, policy_type, policy_version, policy_name, description,
			   status, created_by, owners_authorizer, client_owner, organization_id,
			   policy_rules, scope, restrictions, poa_template,
			   created_at, updated_at, activated_at, expires_at, revoked_at,
			   previous_version, change_log, enforcement_count, last_enforced_at,
			   tags, metadata
		FROM authorization_policies
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if criteria != nil {
		if len(criteria.PolicyTypes) > 0 {
			query += fmt.Sprintf(" AND policy_type = ANY($%d)", argCount)
			policyTypes := make([]string, len(criteria.PolicyTypes))
			for i, pt := range criteria.PolicyTypes {
				policyTypes[i] = string(pt)
			}
			args = append(args, policyTypes)
			argCount++
		}
		if len(criteria.Statuses) > 0 {
			query += fmt.Sprintf(" AND status = ANY($%d)", argCount)
			statuses := make([]string, len(criteria.Statuses))
			for i, s := range criteria.Statuses {
				statuses[i] = string(s)
			}
			args = append(args, statuses)
			argCount++
		}
		if criteria.ClientOwner != "" {
			query += fmt.Sprintf(" AND client_owner = $%d", argCount)
			args = append(args, criteria.ClientOwner)
			argCount++
		}
		if criteria.OwnersAuthorizer != "" {
			query += fmt.Sprintf(" AND owners_authorizer = $%d", argCount)
			args = append(args, criteria.OwnersAuthorizer)
			argCount++
		}
		if criteria.SearchText != "" {
			query += fmt.Sprintf(" AND (policy_name ILIKE $%d OR description ILIKE $%d)", argCount, argCount+1)
			searchPattern := "%" + criteria.SearchText + "%"
			args = append(args, searchPattern, searchPattern)
			argCount += 2
		}
		if len(criteria.Tags) > 0 {
			query += fmt.Sprintf(" AND tags ?& $%d", argCount)
			args = append(args, criteria.Tags)
			argCount++
		}
		if !criteria.CreatedAfter.IsZero() {
			query += fmt.Sprintf(" AND created_at >= $%d", argCount)
			args = append(args, criteria.CreatedAfter)
			argCount++
		}
		if !criteria.CreatedBefore.IsZero() {
			query += fmt.Sprintf(" AND created_at <= $%d", argCount)
			args = append(args, criteria.CreatedBefore)
			argCount++
		}
	}

	query += " ORDER BY created_at DESC"

	if criteria != nil && criteria.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, criteria.Limit)
	}

	return query, args
}

func (s *DatabasePolicyStore) scanPolicyRows(rows *sql.Rows) (*AuthorizationPolicy, error) {
	policy := &AuthorizationPolicy{}
	var policyRulesJSON, scopeJSON, restrictionsJSON, poaTemplateJSON, tagsJSON, metadataJSON []byte

	err := rows.Scan(
		&policy.PolicyID, &policy.PolicyType, &policy.PolicyVersion, &policy.PolicyName, &policy.Description,
		&policy.Status, &policy.CreatedBy, &policy.OwnersAuthorizer, &policy.ClientOwner, &policy.OrganizationID,
		&policyRulesJSON, &scopeJSON, &restrictionsJSON, &poaTemplateJSON,
		&policy.CreatedAt, &policy.UpdatedAt, &policy.ActivatedAt, &policy.ExpiresAt, &policy.RevokedAt,
		&policy.PreviousVersion, &policy.ChangeLog, &policy.EnforcementCount, &policy.LastEnforcedAt,
		&tagsJSON, &metadataJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan policy: %w", err)
	}

	if err := json.Unmarshal(policyRulesJSON, &policy.PolicyRules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy rules: %w", err)
	}
	if len(scopeJSON) > 0 {
		if err := json.Unmarshal(scopeJSON, &policy.Scope); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
		}
	}
	if len(restrictionsJSON) > 0 {
		if err := json.Unmarshal(restrictionsJSON, &policy.Restrictions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal restrictions: %w", err)
		}
	}
	if len(poaTemplateJSON) > 0 {
		if err := json.Unmarshal(poaTemplateJSON, &policy.PoATemplate); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PoA template: %w", err)
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &policy.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return policy, nil
}

// Exists checks if a policy with the given ID exists
func (s *DatabasePolicyStore) Exists(ctx context.Context, policyID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM authorization_policies WHERE policy_id = $1)`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, policyID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check policy existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of policies, optionally filtered by status
func (s *DatabasePolicyStore) Count(ctx context.Context, status *PolicyStatus) (int, error) {
	var query string
	var args []interface{}

	if status != nil {
		query = `SELECT COUNT(*) FROM authorization_policies WHERE status = $1`
		args = append(args, *status)
	} else {
		query = `SELECT COUNT(*) FROM authorization_policies`
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count policies: %w", err)
	}

	return count, nil
}

// Close releases the database connection
func (s *DatabasePolicyStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ensure DatabasePolicyStore implements PolicyStore
var _ PolicyStore = (*DatabasePolicyStore)(nil)
