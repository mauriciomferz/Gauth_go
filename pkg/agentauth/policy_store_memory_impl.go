package agentauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// InMemoryPolicyStore implements PolicyStore using an in-memory map
// Suitable for testing and single-instance deployments
type InMemoryPolicyStore struct {
	policies sync.Map // map[string]*AuthorizationPolicy
}

// NewInMemoryPolicyStore creates a new in-memory policy store
func NewInMemoryPolicyStore() *InMemoryPolicyStore {
	return &InMemoryPolicyStore{}
}

// deepCopy performs a deep copy of the policy
func deepCopy(src *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst AuthorizationPolicy
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// Create stores a new policy
func (s *InMemoryPolicyStore) Create(ctx context.Context, policy *AuthorizationPolicy) error {
	if _, ok := s.policies.Load(policy.PolicyID); ok {
		return fmt.Errorf("policy already exists: %s", policy.PolicyID)
	}

	copy, err := deepCopy(policy)
	if err != nil {
		return fmt.Errorf("failed to copy policy: %w", err)
	}

	s.policies.Store(policy.PolicyID, copy)
	return nil
}

// Get retrieves a policy by ID
func (s *InMemoryPolicyStore) Get(ctx context.Context, policyID string) (*AuthorizationPolicy, error) {
	val, ok := s.policies.Load(policyID)
	if !ok {
		return nil, ErrPolicyNotFound
	}

	policy := val.(*AuthorizationPolicy)
	return deepCopy(policy)
}

// Update updates an existing policy
func (s *InMemoryPolicyStore) Update(ctx context.Context, policy *AuthorizationPolicy) error {
	if _, ok := s.policies.Load(policy.PolicyID); !ok {
		return ErrPolicyNotFound
	}

	copy, err := deepCopy(policy)
	if err != nil {
		return fmt.Errorf("failed to copy policy: %w", err)
	}

	s.policies.Store(policy.PolicyID, copy)
	return nil
}

// Delete removes a policy from the store
func (s *InMemoryPolicyStore) Delete(ctx context.Context, policyID string) error {
	if _, ok := s.policies.Load(policyID); !ok {
		return ErrPolicyNotFound
	}
	s.policies.Delete(policyID)
	return nil
}

// List returns all policies, optionally filtered by status
func (s *InMemoryPolicyStore) List(ctx context.Context, status *PolicyStatus) ([]*AuthorizationPolicy, error) {
	var policies []*AuthorizationPolicy
	var err error

	s.policies.Range(func(key, value interface{}) bool {
		policy := value.(*AuthorizationPolicy)
		if status == nil || policy.Status == *status {
			var copy *AuthorizationPolicy
			copy, err = deepCopy(policy)
			if err != nil {
				return false
			}
			policies = append(policies, copy)
		}
		return true
	})

	if err != nil {
		return nil, err
	}
	return policies, nil
}

// Search finds policies matching the given criteria
func (s *InMemoryPolicyStore) Search(ctx context.Context, criteria *PolicySearchCriteria) ([]*AuthorizationPolicy, error) {
	var policies []*AuthorizationPolicy
	var err error

	s.policies.Range(func(key, value interface{}) bool {
		policy := value.(*AuthorizationPolicy)

		if s.matchesCriteria(policy, criteria) {
			var copy *AuthorizationPolicy
			copy, err = deepCopy(policy)
			if err != nil {
				return false
			}
			policies = append(policies, copy)
		}
		return true
	})

	if err != nil {
		return nil, err
	}

	return s.applyPagination(policies, criteria), nil
}

func (s *InMemoryPolicyStore) matchesCriteria(policy *AuthorizationPolicy, criteria *PolicySearchCriteria) bool {
	if criteria == nil {
		return true
	}

	if len(criteria.PolicyTypes) > 0 {
		found := false
		for _, pt := range criteria.PolicyTypes {
			if policy.PolicyType == pt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(criteria.Statuses) > 0 {
		found := false
		for _, st := range criteria.Statuses {
			if policy.Status == st {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if criteria.ClientOwner != "" && policy.ClientOwner != criteria.ClientOwner {
		return false
	}

	if criteria.OwnersAuthorizer != "" && policy.OwnersAuthorizer != criteria.OwnersAuthorizer {
		return false
	}

	if len(criteria.Tags) > 0 {
		for _, tag := range criteria.Tags {
			found := false
			for _, pt := range policy.Tags {
				if pt == tag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	if criteria.SearchText != "" {
		contains := func(s, substr string) bool {
			return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
		}
		if !contains(policy.PolicyName, criteria.SearchText) && !contains(policy.Description, criteria.SearchText) {
			return false
		}
	}

	if criteria.CreatedAfter != nil && !policy.CreatedAt.After(*criteria.CreatedAfter) {
		return false
	}

	if criteria.CreatedBefore != nil && !policy.CreatedAt.Before(*criteria.CreatedBefore) {
		return false
	}

	return true
}

func (s *InMemoryPolicyStore) applyPagination(
	policies []*AuthorizationPolicy, criteria *PolicySearchCriteria,
) []*AuthorizationPolicy {
	if criteria == nil {
		return policies
	}

	if criteria.Offset > 0 {
		if criteria.Offset >= len(policies) {
			return []*AuthorizationPolicy{}
		}
		policies = policies[criteria.Offset:]
	}
	if criteria.Limit > 0 && criteria.Limit < len(policies) {
		policies = policies[:criteria.Limit]
	}
	return policies
}

// Exists checks if a policy with the given ID exists
func (s *InMemoryPolicyStore) Exists(ctx context.Context, policyID string) (bool, error) {
	_, ok := s.policies.Load(policyID)
	return ok, nil
}

// Count returns the total number of policies, optionally filtered by status
func (s *InMemoryPolicyStore) Count(ctx context.Context, status *PolicyStatus) (int, error) {
	count := 0
	s.policies.Range(func(key, value interface{}) bool {
		policy := value.(*AuthorizationPolicy)
		if status == nil || policy.Status == *status {
			count++
		}
		return true
	})
	return count, nil
}

// Ensure InMemoryPolicyStore implements PolicyStore
var _ PolicyStore = (*InMemoryPolicyStore)(nil)
