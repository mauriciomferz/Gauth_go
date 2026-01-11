package agentauth

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// PowerAdministrationPoint represents a power administration point
type PowerAdministrationPoint struct {
	AgentAuth       AgentAuth
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	CreatedAt       time.Time   `json:"created_at"`
	store           PolicyStore // Pluggable policy storage backend
	policyIDCounter int64       // Atomic counter for unique policy IDs
}

// NewPowerAdministrationPoint creates a new power administration point with in-memory storage
func NewPowerAdministrationPoint(id, name, description string) *PowerAdministrationPoint {
	return &PowerAdministrationPoint{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		store:       NewInMemoryPolicyStore(),
	}
}

// NewPowerAdministrationPointWithStore creates a new power administration point with a custom policy store
func NewPowerAdministrationPointWithStore(id, name, description string, store PolicyStore) *PowerAdministrationPoint {
	return &PowerAdministrationPoint{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		store:       store,
	}
}

// InvalidateToken invalidates a token
func (p *PowerAdministrationPoint) InvalidateToken(token string) error {
	// Delegate to the underlying AgentAuth service
	_, err := p.AgentAuth.ValidateToken(token)
	if err != nil {
		return err
	}
	// In a real implementation, this would mark the token as invalid
	// For now, just return success
	return nil
}

// ============================================================================
// Policy Management Methods (using types from pap_types.go)
// ============================================================================

// CreatePolicy creates a new authorization policy
func (p *PowerAdministrationPoint) CreatePolicy(ctx context.Context, request *PolicyCreateRequest) (*AuthorizationPolicy, error) {
	if request == nil {
		return nil, fmt.Errorf("policy create request cannot be nil")
	}

	// Validate request
	if request.PolicyName == "" {
		return nil, fmt.Errorf("policy name is required")
	}
	if request.PolicyType == "" {
		return nil, fmt.Errorf("policy type is required")
	}

	// Generate unique policy ID using atomic counter to prevent collisions in concurrent scenarios
	counter := atomic.AddInt64(&p.policyIDCounter, 1)
	policyID := fmt.Sprintf("policy_%s_%d_%d", request.PolicyType, time.Now().UnixNano(), counter)

	now := time.Now()

	// Create policy
	policy := &AuthorizationPolicy{
		PolicyID:         policyID,
		PolicyType:       request.PolicyType,
		PolicyVersion:    1,
		PolicyName:       request.PolicyName,
		Description:      request.Description,
		Status:           PolicyStatusDraft,
		OwnersAuthorizer: request.OwnersAuthorizer,
		ClientOwner:      request.ClientOwner,
		PolicyRules:      request.PolicyRules,
		Scope:            request.Scope,
		Restrictions:     request.Restrictions,
		PoATemplate:      request.PoATemplate,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        request.ExpiresAt,
		Tags:             request.Tags,
		Metadata:         request.Metadata,
	}

	// Store policy using the policy store interface
	if err := p.store.Create(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	return policy, nil
}

// GetPolicy retrieves a policy by ID
func (p *PowerAdministrationPoint) GetPolicy(ctx context.Context, policyID string) (*AuthorizationPolicy, error) {
	return p.store.Get(ctx, policyID)
}

// UpdatePolicy updates an existing policy
func (p *PowerAdministrationPoint) UpdatePolicy(ctx context.Context, request *PolicyUpdateRequest) (*AuthorizationPolicy, error) {
	if request == nil {
		return nil, fmt.Errorf("policy update request cannot be nil")
	}

	// Get the existing policy
	policy, err := p.store.Get(ctx, request.PolicyID)
	if err != nil {
		return nil, err
	}

	// Only allow updates to draft or suspended policies
	if policy.Status != PolicyStatusDraft && policy.Status != PolicyStatusSuspended {
		return nil, fmt.Errorf("cannot update policy in status: %s", policy.Status)
	}

	// Update fields if provided
	if request.PolicyName != nil {
		policy.PolicyName = *request.PolicyName
	}
	if request.Description != nil {
		policy.Description = *request.Description
	}
	if request.PolicyRules != nil {
		policy.PolicyRules = *request.PolicyRules
	}
	if request.Scope != nil {
		policy.Scope = request.Scope
	}
	if request.Restrictions != nil {
		policy.Restrictions = *request.Restrictions
	}
	if request.ExpiresAt != nil {
		policy.ExpiresAt = request.ExpiresAt
	}
	if request.Tags != nil {
		policy.Tags = *request.Tags
	}
	if request.Metadata != nil {
		policy.Metadata = *request.Metadata
	}

	policy.PolicyVersion++
	policy.UpdatedAt = time.Now()
	policy.ChangeLog = request.ChangeLog

	// Save updated policy
	if err := p.store.Update(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	return policy, nil
}

// ActivatePolicy activates a draft policy
func (p *PowerAdministrationPoint) ActivatePolicy(ctx context.Context, policyID, approvedBy string) error {
	policy, err := p.store.Get(ctx, policyID)
	if err != nil {
		return err
	}

	if policy.Status != PolicyStatusDraft {
		return fmt.Errorf("only draft policies can be activated, current status: %s", policy.Status)
	}

	// Validate policy before activation
	if err := p.validatePolicy(policy); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	now := time.Now()
	policy.Status = PolicyStatusActive
	policy.ActivatedAt = &now
	policy.UpdatedAt = now
	if policy.Metadata == nil {
		policy.Metadata = make(map[string]interface{})
	}
	policy.Metadata["approved_by"] = approvedBy

	return p.store.Update(ctx, policy)
}

// SuspendPolicy temporarily suspends an active policy
func (p *PowerAdministrationPoint) SuspendPolicy(ctx context.Context, policyID, reason string) error {
	policy, err := p.store.Get(ctx, policyID)
	if err != nil {
		return err
	}

	if policy.Status != PolicyStatusActive {
		return fmt.Errorf("only active policies can be suspended, current status: %s", policy.Status)
	}

	policy.Status = PolicyStatusSuspended
	policy.UpdatedAt = time.Now()
	if policy.Metadata == nil {
		policy.Metadata = make(map[string]interface{})
	}
	policy.Metadata["suspension_reason"] = reason
	policy.Metadata["suspended_at"] = time.Now()

	return p.store.Update(ctx, policy)
}

// RevokePolicy permanently revokes a policy
func (p *PowerAdministrationPoint) RevokePolicy(ctx context.Context, policyID, reason string) error {
	policy, err := p.store.Get(ctx, policyID)
	if err != nil {
		return err
	}

	if policy.Status == PolicyStatusRevoked {
		return fmt.Errorf("policy already revoked")
	}

	now := time.Now()
	policy.Status = PolicyStatusRevoked
	policy.UpdatedAt = now
	policy.RevokedAt = &now
	if policy.Metadata == nil {
		policy.Metadata = make(map[string]interface{})
	}
	policy.Metadata["revocation_reason"] = reason
	policy.Metadata["revoked_at"] = now

	return p.store.Update(ctx, policy)
}

// DeletePolicy deletes a policy (only draft or revoked policies)
func (p *PowerAdministrationPoint) DeletePolicy(ctx context.Context, policyID string) error {
	policy, err := p.store.Get(ctx, policyID)
	if err != nil {
		return err
	}

	// Only allow deletion of draft or revoked policies
	if policy.Status != PolicyStatusDraft && policy.Status != PolicyStatusRevoked {
		return fmt.Errorf("cannot delete policy in status: %s (only draft or revoked)", policy.Status)
	}

	return p.store.Delete(ctx, policyID)
}

// SearchPolicies searches for policies based on criteria
func (p *PowerAdministrationPoint) SearchPolicies(
	ctx context.Context, criteria *PolicySearchCriteria,
) ([]*AuthorizationPolicy, error) {
	return p.store.Search(ctx, criteria)
}

// ListPolicies lists all policies (optionally filtered by status)
func (p *PowerAdministrationPoint) ListPolicies(ctx context.Context, status *PolicyStatus) ([]*AuthorizationPolicy, error) {
	return p.store.List(ctx, status)
}

// ValidatePolicy validates a policy's configuration
func (p *PowerAdministrationPoint) ValidatePolicy(ctx context.Context, policyID string) (*PolicyValidationResult, error) {
	policy, err := p.store.Get(ctx, policyID)
	if err != nil {
		return nil, err
	}

	result := &PolicyValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	if err := p.validatePolicy(policy); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	return result, nil
}

// PAPAggregateStatistics represents aggregate statistics across all policies
type PAPAggregateStatistics struct {
	TotalPolicies     int                `json:"total_policies"`
	ActivePolicies    int                `json:"active_policies"`
	DraftPolicies     int                `json:"draft_policies"`
	SuspendedPolicies int                `json:"suspended_policies"`
	RevokedPolicies   int                `json:"revoked_policies"`
	ExpiredPolicies   int                `json:"expired_policies"`
	PoliciesByType    map[PolicyType]int `json:"policies_by_type"`
}

// GetPolicyStatistics returns aggregate statistics about all policies
func (p *PowerAdministrationPoint) GetPolicyStatistics(ctx context.Context) (*PAPAggregateStatistics, error) {
	// Get all policies
	allPolicies, err := p.store.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve policies: %w", err)
	}

	stats := &PAPAggregateStatistics{
		TotalPolicies:     len(allPolicies),
		ActivePolicies:    0,
		DraftPolicies:     0,
		SuspendedPolicies: 0,
		RevokedPolicies:   0,
		ExpiredPolicies:   0,
		PoliciesByType:    make(map[PolicyType]int),
	}

	now := time.Now()

	for _, policy := range allPolicies {
		// Count by status
		switch policy.Status {
		case PolicyStatusActive:
			stats.ActivePolicies++
		case PolicyStatusDraft:
			stats.DraftPolicies++
		case PolicyStatusSuspended:
			stats.SuspendedPolicies++
		case PolicyStatusRevoked:
			stats.RevokedPolicies++
		case PolicyStatusExpired:
			stats.ExpiredPolicies++
		}

		// Check for expired policies
		if policy.ExpiresAt != nil && policy.ExpiresAt.Before(now) {
			stats.ExpiredPolicies++
		}

		// Count by type
		stats.PoliciesByType[policy.PolicyType]++
	}

	return stats, nil
}

// ============================================================================
// Internal helper methods
// ============================================================================

// validatePolicy performs comprehensive validation of a policy
func (p *PowerAdministrationPoint) validatePolicy(policy *AuthorizationPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if policy.PolicyName == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.PolicyType == "" {
		return fmt.Errorf("policy type is required")
	}

	// Validate expiration date (if set) - should be after creation
	if policy.ExpiresAt != nil && policy.ExpiresAt.Before(policy.CreatedAt) {
		return fmt.Errorf("expiration date cannot be before creation date")
	}

	// Validate rules
	if len(policy.PolicyRules.AllowedActions) == 0 && len(policy.PolicyRules.DeniedActions) == 0 {
		return fmt.Errorf("policy must have at least one allowed or denied action")
	}

	// Validate scope (if provided)
	if policy.Scope != nil {
		if len(policy.Scope.Countries) == 0 && len(policy.Scope.Sectors) == 0 &&
			len(policy.Scope.Entities) == 0 && len(policy.Scope.ClientIDs) == 0 {
			return fmt.Errorf("policy scope must have at least one defined constraint")
		}
	}

	return nil
}
