package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorySubscriptionStore_Cleanup(t *testing.T) {
	store := NewMemorySubscriptionStore()
	ctx := context.Background()

	// Helper to create subs with specific timestamps
	createSub := func(id string, status SubscriptionStatus, age time.Duration) {
		sub := &Subscription{
			ID:        id,
			Status:    status,
			CreatedAt: time.Now().Add(-age),
			UpdatedAt: time.Now().Add(-age),
		}

		// Add some indexable data
		if id == "sub_completed" {
			sub.ClientAuthorizationGrant = &ClientAuthGrant{
				ClientID: "client_1",
			}
			sub.ResourceOwnerIdentity = &IdentityProofResult{
				SubjectID: "user_1",
			}
		}

		err := store.CreateSubscription(ctx, sub)
		require.NoError(t, err)

		// CreateSubscription resets timestamps to Now(). We need to force update them.
		// Since store is in-memory and locked, we access map directly via reflection or just use Save but Save resets UpdatedAt too.
		// Actually, CreateSubscription DOES overwrite CreatedAt/UpdatedAt to time.Now().
		// We need to bypass the public methods to set old timestamps for testing.
		store.mu.Lock()
		store.subscriptions[sub.ID].CreatedAt = time.Now().Add(-age)
		store.subscriptions[sub.ID].UpdatedAt = time.Now().Add(-age)
		store.mu.Unlock()
	}

	// 1. Pending - Recent (Should Keep)
	createSub("sub_pending_recent", SubscriptionStatusPending, 1*time.Hour)

	// 2. Pending - Stale (Should Delete) (Age > 24h)
	createSub("sub_pending_stale", SubscriptionStatusPending, 25*time.Hour)

	// 3. Completed - Recent (Should Keep)
	createSub("sub_completed_recent", SubscriptionStatusCompleted, 5*24*time.Hour)

	// 4. Completed - Old (Should Delete) (Age > 30d)
	createSub("sub_completed_old", SubscriptionStatusCompleted, 31*24*time.Hour)

	// 5. Failed - Recent (Should Keep)
	createSub("sub_failed_recent", SubscriptionStatusFailed, 1*time.Hour)

	// 6. Failed - Stale (Should Delete)
	createSub("sub_failed_stale", SubscriptionStatusFailed, 25*time.Hour)

	// Run Cleanup
	count, err := store.DeleteExpiredSubscriptions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count) // pending_stale, completed_old, failed_stale

	// Verify persistence
	_, err = store.GetSubscription(ctx, "sub_pending_recent")
	assert.NoError(t, err)

	_, err = store.GetSubscription(ctx, "sub_pending_stale")
	assert.ErrorIs(t, err, ErrSubscriptionNotFound)

	_, err = store.GetSubscription(ctx, "sub_completed_recent")
	assert.NoError(t, err)

	_, err = store.GetSubscription(ctx, "sub_completed_old")
	assert.ErrorIs(t, err, ErrSubscriptionNotFound)

	_, err = store.GetSubscription(ctx, "sub_failed_recent")
	assert.NoError(t, err)

	_, err = store.GetSubscription(ctx, "sub_failed_stale")
	assert.ErrorIs(t, err, ErrSubscriptionNotFound)
}
