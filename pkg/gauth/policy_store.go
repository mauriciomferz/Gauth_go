package gauth

import (
	"context"
	"errors"
)

// ErrPolicyNotFound is returned when a requested policy does not exist
var ErrPolicyNotFound = errors.New("policy not found")

// PolicyStore defines the interface for policy storage and retrieval
type PolicyStore interface {
	// Create stores a new policy
	Create(ctx context.Context, policy *AuthorizationPolicy) error

	// Get retrieves a policy by ID
	Get(ctx context.Context, policyID string) (*AuthorizationPolicy, error)

	// Update updates an existing policy
	Update(ctx context.Context, policy *AuthorizationPolicy) error

	// Delete removes a policy from the store
	Delete(ctx context.Context, policyID string) error

	// List returns all policies, optionally filtered by status
	List(ctx context.Context, status *PolicyStatus) ([]*AuthorizationPolicy, error)

	// Search finds policies matching the given criteria
	Search(ctx context.Context, criteria *PolicySearchCriteria) ([]*AuthorizationPolicy, error)

	// Exists checks if a policy with the given ID exists
	Exists(ctx context.Context, policyID string) (bool, error)

	// Count returns the total number of policies, optionally filtered by status
	Count(ctx context.Context, status *PolicyStatus) (int, error)
}
