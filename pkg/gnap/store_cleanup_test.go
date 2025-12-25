package gnap

import (
	"testing"
	"time"
)

// TestMemoryGrantStore_Cleanup verifies expired grant cleanup
func TestMemoryGrantStore_Cleanup(t *testing.T) {
	store := NewMemoryGrantStore()

	// Create grants with different expiration times
	req := &GrantRequest{
		Client: &ClientInstance{InstanceID: "client-123"},
	}

	// Create expired grant
	expiredGrant, _ := store.Create(req)
	expiredGrant.ExpiresAt = time.Now().Add(-1 * time.Hour) // Already expired
	_ = store.Update(expiredGrant)

	// Create grant that will expire soon
	soonGrant, _ := store.Create(req)
	soonGrant.ExpiresAt = time.Now().Add(-1 * time.Second)
	_ = store.Update(soonGrant)

	// Create valid grant
	validGrant, _ := store.Create(req)
	validGrant.ExpiresAt = time.Now().Add(1 * time.Hour)
	_ = store.Update(validGrant)

	// Create grant with no expiration
	noExpireGrant, _ := store.Create(req)
	noExpireGrant.ExpiresAt = time.Time{} // Zero value = no expiration
	_ = store.Update(noExpireGrant)

	// Verify all grants exist
	if len(store.grants) != 4 {
		t.Fatalf("Expected 4 grants before cleanup, got %d", len(store.grants))
	}

	// Run cleanup
	removed := store.Cleanup()

	// Should remove 2 expired grants
	if removed != 2 {
		t.Errorf("Expected 2 grants removed, got %d", removed)
	}

	// Verify only valid and non-expiring grants remain
	if len(store.grants) != 2 {
		t.Errorf("Expected 2 grants after cleanup, got %d", len(store.grants))
	}

	// Verify the right grants remain
	_, err := store.Get(validGrant.ID)
	if err != nil {
		t.Error("Valid grant should still exist")
	}

	_, err = store.Get(noExpireGrant.ID)
	if err != nil {
		t.Error("Non-expiring grant should still exist")
	}

	// Verify expired grants are gone
	_, err = store.Get(expiredGrant.ID)
	if err == nil {
		t.Error("Expired grant should be removed")
	}

	_, err = store.Get(soonGrant.ID)
	if err == nil {
		t.Error("Soon-expired grant should be removed")
	}
}

// TestMemoryGrantStore_CleanupClientIndex verifies client index is maintained
func TestMemoryGrantStore_CleanupClientIndex(t *testing.T) {
	store := NewMemoryGrantStore()

	clientID := "client-456"
	req := &GrantRequest{
		Client: &ClientInstance{InstanceID: clientID},
	}

	// Create 2 expired and 1 valid grant for the same client
	expired1, _ := store.Create(req)
	expired1.ExpiresAt = time.Now().Add(-1 * time.Hour)
	_ = store.Update(expired1)

	expired2, _ := store.Create(req)
	expired2.ExpiresAt = time.Now().Add(-1 * time.Minute)
	_ = store.Update(expired2)

	validGrant, _ := store.Create(req)
	validGrant.ExpiresAt = time.Now().Add(1 * time.Hour)
	_ = store.Update(validGrant)

	// Verify client has 3 grants
	grants, _ := store.ListByClient(clientID)
	if len(grants) != 3 {
		t.Fatalf("Expected 3 grants for client before cleanup, got %d", len(grants))
	}

	// Cleanup
	store.Cleanup()

	// Verify client index has only 1 grant
	grants, _ = store.ListByClient(clientID)
	if len(grants) != 1 {
		t.Errorf("Expected 1 grant for client after cleanup, got %d", len(grants))
	}

	if grants[0].ID != validGrant.ID {
		t.Error("Wrong grant remained in client index")
	}
}
