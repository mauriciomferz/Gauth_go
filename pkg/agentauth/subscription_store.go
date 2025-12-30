package agentauth

import (
	"context"
	"errors"
)

// SubscriptionStore defines the interface for storing and retrieving subscriptions
// This supports the AAP-001 ONE-OFF subscription enrollment (Steps I-VIII)
type SubscriptionStore interface {
	// CreateSubscription creates a new subscription and returns the subscription ID
	CreateSubscription(ctx context.Context, sub *Subscription) error

	// GetSubscription retrieves a subscription by ID
	GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)

	// SaveSubscription updates an existing subscription
	SaveSubscription(ctx context.Context, sub *Subscription) error

	// DeleteSubscription removes a subscription
	DeleteSubscription(ctx context.Context, subscriptionID string) error

	// ListSubscriptions returns all subscriptions for a client
	ListSubscriptions(ctx context.Context, clientID string) ([]*Subscription, error)

	// GetSubscriptionByClient finds an active subscription for a specific client and resource owner
	GetSubscriptionByClient(ctx context.Context, clientID, resourceOwnerID string) (*Subscription, error)

	// DeleteExpiredSubscriptions removes stale subscriptions (e.g. pending > 24h, completed > 30d)
	DeleteExpiredSubscriptions(ctx context.Context) (int, error)
}

// Common errors for subscription store operations
var (
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionInvalidStatus = errors.New("invalid subscription status")
	ErrSubscriptionExpired       = errors.New("subscription has expired")
)
