package gauth

import (
	"context"
	"sync"
	"time"
)

// PIPPolicyStore defines the interface for PIP policy caching
type PIPPolicyStore interface {
	Get(ctx context.Context, policyID string) (*PowerOfAttorneyPolicy, error)
	Set(ctx context.Context, policyID string, policy *PowerOfAttorneyPolicy, ttl time.Duration) error
	Delete(ctx context.Context, policyID string) error
}

// InMemoryPIPPolicyStore implements PIPPolicyStore using an in-memory map
type InMemoryPIPPolicyStore struct {
	cache sync.Map // map[string]*cachedPolicyItem
}

type cachedPolicyItem struct {
	policy    *PowerOfAttorneyPolicy
	expiresAt time.Time
}

// NewInMemoryPIPPolicyStore creates a new in-memory PIP policy store
func NewInMemoryPIPPolicyStore() *InMemoryPIPPolicyStore {
	return &InMemoryPIPPolicyStore{}
}

// Get retrieves a policy by ID
func (s *InMemoryPIPPolicyStore) Get(ctx context.Context, policyID string) (*PowerOfAttorneyPolicy, error) {
	val, ok := s.cache.Load(policyID)
	if !ok {
		return nil, ErrPolicyNotFound
	}

	item := val.(*cachedPolicyItem)
	if time.Now().After(item.expiresAt) {
		s.cache.Delete(policyID)
		return nil, ErrPolicyNotFound
	}

	return item.policy, nil
}

// Set stores a policy with an expiration time
func (s *InMemoryPIPPolicyStore) Set(ctx context.Context, policyID string, policy *PowerOfAttorneyPolicy, ttl time.Duration) error {
	s.cache.Store(policyID, &cachedPolicyItem{
		policy:    policy,
		expiresAt: time.Now().Add(ttl),
	})
	return nil
}

// Delete removes a policy from the store
// Delete removes a policy from the store
func (s *InMemoryPIPPolicyStore) Delete(ctx context.Context, policyID string) error {
	s.cache.Delete(policyID)
	return nil
}

// Cleanup removes all expired policies from the store
// This should be called periodically to prevent memory leaks
func (s *InMemoryPIPPolicyStore) Cleanup(ctx context.Context) (int, error) {
	count := 0
	now := time.Now()

	s.cache.Range(func(key, value interface{}) bool {
		item := value.(*cachedPolicyItem)
		if now.After(item.expiresAt) {
			s.cache.Delete(key)
			count++
		}
		return true
	})

	return count, nil
}
