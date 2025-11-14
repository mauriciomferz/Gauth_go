package gauth

import (
	"context"
	"sync"
	"time"
)

// MemorySubscriptionStore is an in-memory implementation of SubscriptionStore
// This is suitable for testing and development. For production, use PostgresSubscriptionStore.
type MemorySubscriptionStore struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription

	// Index for quick lookups
	clientIndex map[string]map[string]*Subscription // clientID -> resourceOwnerID -> subscription
}

// NewMemorySubscriptionStore creates a new in-memory subscription store
func NewMemorySubscriptionStore() *MemorySubscriptionStore {
	return &MemorySubscriptionStore{
		subscriptions: make(map[string]*Subscription),
		clientIndex:   make(map[string]map[string]*Subscription),
	}
}

// CreateSubscription creates a new subscription
func (s *MemorySubscriptionStore) CreateSubscription(ctx context.Context, sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[sub.ID]; exists {
		return ErrSubscriptionAlreadyExists
	}

	// Set timestamps
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	// Store subscription
	s.subscriptions[sub.ID] = sub

	return nil
}

// GetSubscription retrieves a subscription by ID
func (s *MemorySubscriptionStore) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, exists := s.subscriptions[subscriptionID]
	if !exists {
		return nil, ErrSubscriptionNotFound
	}

	return sub, nil
}

// SaveSubscription updates an existing subscription
func (s *MemorySubscriptionStore) SaveSubscription(ctx context.Context, sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[sub.ID]; !exists {
		return ErrSubscriptionNotFound
	}

	// Update timestamp
	sub.UpdatedAt = time.Now()

	// Store updated subscription
	s.subscriptions[sub.ID] = sub

	// Update index if we have client authorization grant
	if sub.ClientAuthorizationGrant != nil {
		clientID := sub.ClientAuthorizationGrant.ClientID
		if s.clientIndex[clientID] == nil {
			s.clientIndex[clientID] = make(map[string]*Subscription)
		}

		// Use resource owner ID if available
		resourceOwnerID := ""
		if sub.ResourceOwnerIdentity != nil {
			resourceOwnerID = sub.ResourceOwnerIdentity.SubjectID
		}
		s.clientIndex[clientID][resourceOwnerID] = sub
	}

	return nil
}

// DeleteSubscription removes a subscription
func (s *MemorySubscriptionStore) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[subscriptionID]
	if !exists {
		return ErrSubscriptionNotFound
	}

	// Remove from index
	if sub.ClientAuthorizationGrant != nil {
		clientID := sub.ClientAuthorizationGrant.ClientID
		resourceOwnerID := ""
		if sub.ResourceOwnerIdentity != nil {
			resourceOwnerID = sub.ResourceOwnerIdentity.SubjectID
		}

		if s.clientIndex[clientID] != nil {
			delete(s.clientIndex[clientID], resourceOwnerID)
			if len(s.clientIndex[clientID]) == 0 {
				delete(s.clientIndex, clientID)
			}
		}
	}

	// Remove subscription
	delete(s.subscriptions, subscriptionID)

	return nil
}

// ListSubscriptions returns all subscriptions for a client
func (s *MemorySubscriptionStore) ListSubscriptions(ctx context.Context, clientID string) ([]*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subscriptions []*Subscription

	for _, sub := range s.subscriptions {
		if sub.ClientAuthorizationGrant != nil && sub.ClientAuthorizationGrant.ClientID == clientID {
			subscriptions = append(subscriptions, sub)
		}
	}

	return subscriptions, nil
}

// GetSubscriptionByClient finds an active subscription for a specific client and resource owner
func (s *MemorySubscriptionStore) GetSubscriptionByClient(ctx context.Context, clientID, resourceOwnerID string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try index first
	if clientSubs, exists := s.clientIndex[clientID]; exists {
		if sub, found := clientSubs[resourceOwnerID]; found {
			// Verify subscription is completed
			if sub.Status == SubscriptionStatusCompleted {
				return sub, nil
			}
		}
	}

	// Fallback to full scan
	for _, sub := range s.subscriptions {
		if sub.ClientAuthorizationGrant != nil &&
			sub.ClientAuthorizationGrant.ClientID == clientID &&
			sub.ResourceOwnerIdentity != nil &&
			sub.ResourceOwnerIdentity.SubjectID == resourceOwnerID &&
			sub.Status == SubscriptionStatusCompleted {
			return sub, nil
		}
	}

	return nil, ErrSubscriptionNotFound
}

// GetStats returns statistics about stored subscriptions (useful for monitoring)
func (s *MemorySubscriptionStore) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)
	stats["total"] = len(s.subscriptions)

	statusCounts := make(map[SubscriptionStatus]int)
	for _, sub := range s.subscriptions {
		statusCounts[sub.Status]++
	}

	for status, count := range statusCounts {
		stats[string(status)] = count
	}

	return stats
}
