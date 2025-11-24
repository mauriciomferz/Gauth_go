package authz

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for authorization policies and decisions
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new authorization repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// PolicyRecord represents an authorization policy in the database
type PolicyRecord struct {
	ID          string
	TenantID    string
	PolicyName  string
	PolicyType  string
	Version     int
	Status      string
	Description string
	Rules       map[string]interface{}
	Conditions  map[string]interface{}
	Actions     []string
	Resources   []string
	Effect      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   *string
	ValidFrom   *time.Time
	ValidUntil  *time.Time
	Priority    int
}

// AttributeRecord represents a policy attribute (PIP) in the database
type AttributeRecord struct {
	ID            string
	TenantID      string
	AttributeName string
	AttributeType string
	Source        string
	ValueType     string
	Description   string
	SampleValue   string
	CreatedAt     time.Time
}

// DecisionRecord represents an authorization decision log in the database
type DecisionRecord struct {
	ID                 string
	TenantID           string
	Timestamp          time.Time
	UserID             string
	Action             string
	Resource           string
	Decision           string
	PolicyID           *string
	PolicyName         *string
	IPAddress          *string
	UserAgent          *string
	RequestID          *string
	SessionID          *string
	Context            map[string]interface{}
	EvaluationTimeMs   int
}

// ListPolicies returns all policies for a tenant
func (r *Repository) ListPolicies(ctx context.Context, tenantID string) ([]PolicyRecord, error) {
	query := `
		SELECT 
			id, tenant_id, policy_name, policy_type, version, status,
			description, rules, conditions, actions, resources, effect,
			created_at, updated_at, created_by, valid_from, valid_until, priority
		FROM authorization_policies
		WHERE tenant_id = $1
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var policies []PolicyRecord
	for rows.Next() {
		var p PolicyRecord
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.PolicyName, &p.PolicyType, &p.Version, &p.Status,
			&p.Description, &p.Rules, &p.Conditions, &p.Actions, &p.Resources, &p.Effect,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.ValidFrom, &p.ValidUntil, &p.Priority,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, p)
	}

	return policies, nil
}

// GetPolicy retrieves a specific policy by ID
func (r *Repository) GetPolicy(ctx context.Context, tenantID, policyID string) (*PolicyRecord, error) {
	query := `
		SELECT 
			id, tenant_id, policy_name, policy_type, version, status,
			description, rules, conditions, actions, resources, effect,
			created_at, updated_at, created_by, valid_from, valid_until, priority
		FROM authorization_policies
		WHERE tenant_id = $1 AND id = $2
	`

	var p PolicyRecord
	err := r.db.QueryRow(ctx, query, tenantID, policyID).Scan(
		&p.ID, &p.TenantID, &p.PolicyName, &p.PolicyType, &p.Version, &p.Status,
		&p.Description, &p.Rules, &p.Conditions, &p.Actions, &p.Resources, &p.Effect,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.ValidFrom, &p.ValidUntil, &p.Priority,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return &p, nil
}

// CreatePolicy creates a new authorization policy
func (r *Repository) CreatePolicy(ctx context.Context, p *PolicyRecord) error {
	query := `
		INSERT INTO authorization_policies (
			tenant_id, policy_name, policy_type, version, status,
			description, rules, conditions, actions, resources, effect,
			created_by, valid_from, valid_until, priority,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		p.TenantID, p.PolicyName, p.PolicyType, p.Version, p.Status,
		p.Description, p.Rules, p.Conditions, p.Actions, p.Resources, p.Effect,
		p.CreatedBy, p.ValidFrom, p.ValidUntil, p.Priority,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// UpdatePolicy updates an existing policy
func (r *Repository) UpdatePolicy(ctx context.Context, p *PolicyRecord) error {
	query := `
		UPDATE authorization_policies
		SET policy_name = $1, description = $2, rules = $3, conditions = $4,
		    actions = $5, resources = $6, effect = $7, status = $8,
		    valid_from = $9, valid_until = $10, priority = $11,
		    updated_at = NOW()
		WHERE tenant_id = $12 AND id = $13
	`

	result, err := r.db.Exec(ctx, query,
		p.PolicyName, p.Description, p.Rules, p.Conditions,
		p.Actions, p.Resources, p.Effect, p.Status,
		p.ValidFrom, p.ValidUntil, p.Priority,
		p.TenantID, p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("policy not found")
	}

	return nil
}

// DeletePolicy removes a policy
func (r *Repository) DeletePolicy(ctx context.Context, tenantID, policyID string) error {
	query := `DELETE FROM authorization_policies WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.Exec(ctx, query, tenantID, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("policy not found")
	}

	return nil
}

// ListAttributes returns all policy attributes for a tenant
func (r *Repository) ListAttributes(ctx context.Context, tenantID string) ([]AttributeRecord, error) {
	query := `
		SELECT 
			id, tenant_id, attribute_name, attribute_type, source,
			value_type, description, sample_value, created_at
		FROM policy_attributes
		WHERE tenant_id = $1
		ORDER BY attribute_type, attribute_name
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attributes: %w", err)
	}
	defer rows.Close()

	var attributes []AttributeRecord
	for rows.Next() {
		var a AttributeRecord
		err := rows.Scan(
			&a.ID, &a.TenantID, &a.AttributeName, &a.AttributeType, &a.Source,
			&a.ValueType, &a.Description, &a.SampleValue, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}
		attributes = append(attributes, a)
	}

	return attributes, nil
}

// CreateAttribute creates a new policy attribute
func (r *Repository) CreateAttribute(ctx context.Context, a *AttributeRecord) error {
	query := `
		INSERT INTO policy_attributes (
			tenant_id, attribute_name, attribute_type, source,
			value_type, description, sample_value, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		a.TenantID, a.AttributeName, a.AttributeType, a.Source,
		a.ValueType, a.Description, a.SampleValue,
	).Scan(&a.ID, &a.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create attribute: %w", err)
	}

	return nil
}

// ListDecisions returns recent authorization decisions
func (r *Repository) ListDecisions(ctx context.Context, tenantID string, limit int) ([]DecisionRecord, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT 
			id, tenant_id, timestamp, user_id, action, resource, decision,
			policy_id, policy_name, ip_address, user_agent, request_id,
			session_id, context, evaluation_time_ms
		FROM authorization_logs
		WHERE tenant_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()

	var decisions []DecisionRecord
	for rows.Next() {
		var d DecisionRecord
		err := rows.Scan(
			&d.ID, &d.TenantID, &d.Timestamp, &d.UserID, &d.Action, &d.Resource, &d.Decision,
			&d.PolicyID, &d.PolicyName, &d.IPAddress, &d.UserAgent, &d.RequestID,
			&d.SessionID, &d.Context, &d.EvaluationTimeMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan decision: %w", err)
		}
		decisions = append(decisions, d)
	}

	return decisions, nil
}

// LogDecision records an authorization decision
func (r *Repository) LogDecision(ctx context.Context, d *DecisionRecord) error {
	query := `
		INSERT INTO authorization_logs (
			tenant_id, timestamp, user_id, action, resource, decision,
			policy_id, policy_name, ip_address, user_agent, request_id,
			session_id, context, evaluation_time_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		d.TenantID, d.Timestamp, d.UserID, d.Action, d.Resource, d.Decision,
		d.PolicyID, d.PolicyName, d.IPAddress, d.UserAgent, d.RequestID,
		d.SessionID, d.Context, d.EvaluationTimeMs,
	).Scan(&d.ID)

	if err != nil {
		return fmt.Errorf("failed to log decision: %w", err)
	}

	return nil
}
