package oidc

import (
	"context"
	"testing"
	"time"
)

// TestInMemoryStorage_RefreshTokens tests refresh token operations.
func TestInMemoryStorage_RefreshTokens(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create test token
	token := &RefreshTokenEntry{
		RefreshToken: "test-token-hash",
		ProviderID:   "test-client",
		Subject:      "user123",
		Scopes:       []string{"openid", "profile"},
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	// Test Store
	err := storage.StoreRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("StoreRefreshToken failed: %v", err)
	}

	// Test Get
	retrieved, err := storage.GetRefreshToken(ctx, "test-token-hash")
	if err != nil {
		t.Fatalf("GetRefreshToken failed: %v", err)
	}
	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("refresh_token = %s, want %s", retrieved.RefreshToken, token.RefreshToken)
	}
	if retrieved.ProviderID != token.ProviderID {
		t.Errorf("provider_id = %s, want %s", retrieved.ProviderID, token.ProviderID)
	}
	if retrieved.Subject != token.Subject {
		t.Errorf("subject = %s, want %s", retrieved.Subject, token.Subject)
	}

	// Test Get not found
	_, err = storage.GetRefreshToken(ctx, "nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("GetRefreshToken with invalid token, error = %v, want ErrTokenNotFound", err)
	}

	// Test ListByUser
	tokens, err := storage.ListRefreshTokensByUser(ctx, "user123")
	if err != nil {
		t.Fatalf("ListRefreshTokensByUser failed: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("ListRefreshTokensByUser count = %d, want 1", len(tokens))
	}

	// Test ListByClient
	tokens, err = storage.ListRefreshTokensByClient(ctx, "test-client")
	if err != nil {
		t.Fatalf("ListRefreshTokensByClient failed: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("ListRefreshTokensByClient count = %d, want 1", len(tokens))
	}

	// Test Delete
	err = storage.DeleteRefreshToken(ctx, "test-token-hash")
	if err != nil {
		t.Fatalf("DeleteRefreshToken failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetRefreshToken(ctx, "test-token-hash")
	if err != ErrTokenNotFound {
		t.Errorf("After delete, error = %v, want ErrTokenNotFound", err)
	}
}

// TestInMemoryStorage_RefreshTokenExpiration tests cleanup of expired tokens.
func TestInMemoryStorage_RefreshTokenExpiration(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create expired token
	expiredToken := &RefreshTokenEntry{
		RefreshToken: "expired-token",
		ProviderID:   "test-client",
		Subject:      "user123",
		IssuedAt:     time.Now().Add(-48 * time.Hour),
		ExpiresAt:    time.Now().Add(-24 * time.Hour),
	}
	_ = storage.StoreRefreshToken(ctx, expiredToken)

	// Create valid token
	validToken := &RefreshTokenEntry{
		RefreshToken: "valid-token",
		ProviderID:   "test-client",
		Subject:      "user456",
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	_ = storage.StoreRefreshToken(ctx, validToken)

	// Cleanup expired tokens
	count, err := storage.CleanupExpiredRefreshTokens(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredRefreshTokens failed: %v", err)
	}
	if count != 1 {
		t.Errorf("cleanup count = %d, want 1", count)
	}

	// Verify expired token is gone
	_, err = storage.GetRefreshToken(ctx, "expired-token")
	if err != ErrTokenNotFound {
		t.Errorf("expired token still exists")
	}

	// Verify valid token still exists
	_, err = storage.GetRefreshToken(ctx, "valid-token")
	if err != nil {
		t.Errorf("valid token was removed: %v", err)
	}
}

// TestInMemoryStorage_RevokedTokens tests revoked token operations.
func TestInMemoryStorage_RevokedTokens(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Test token not revoked initially
	revoked, err := storage.IsTokenRevoked(ctx, "test-token")
	if err != nil {
		t.Fatalf("IsTokenRevoked failed: %v", err)
	}
	if revoked {
		t.Error("token should not be revoked initially")
	}

	// Revoke token
	entry := &RevokedTokenEntry{
		TokenID:   "test-token",
		RevokedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err = storage.StoreRevokedToken(ctx, entry)
	if err != nil {
		t.Fatalf("StoreRevokedToken failed: %v", err)
	}

	// Verify token is revoked
	revoked, err = storage.IsTokenRevoked(ctx, "test-token")
	if err != nil {
		t.Fatalf("IsTokenRevoked failed: %v", err)
	}
	if !revoked {
		t.Error("token should be revoked")
	}

	// Test cleanup
	expiredRevocation := &RevokedTokenEntry{
		TokenID:   "expired-revocation",
		RevokedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}
	_ = storage.StoreRevokedToken(ctx, expiredRevocation)

	count, err := storage.CleanupExpiredRevocations(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredRevocations failed: %v", err)
	}
	if count != 1 {
		t.Errorf("cleanup count = %d, want 1", count)
	}
}

// TestInMemoryStorage_DeviceCodes tests device code operations.
func TestInMemoryStorage_DeviceCodes(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create device code entry
	entry := &DeviceCodeEntry{
		DeviceCode: "device-code-123",
		UserCode:   "ABCD-EFGH",
		ClientID:   "test-client",
		Scope:      []string{"openid"},
		Status:     DeviceCodePending,
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}

	// Test Store
	err := storage.StoreDeviceCode(ctx, entry)
	if err != nil {
		t.Fatalf("StoreDeviceCode failed: %v", err)
	}

	// Test GetByDeviceCode
	retrieved, err := storage.GetDeviceCodeByDeviceCode(ctx, "device-code-123")
	if err != nil {
		t.Fatalf("GetDeviceCodeByDeviceCode failed: %v", err)
	}
	if retrieved.DeviceCode != entry.DeviceCode {
		t.Errorf("device_code = %s, want %s", retrieved.DeviceCode, entry.DeviceCode)
	}
	if retrieved.UserCode != entry.UserCode {
		t.Errorf("user_code = %s, want %s", retrieved.UserCode, entry.UserCode)
	}
	if retrieved.Status != entry.Status {
		t.Errorf("status = %s, want %s", retrieved.Status, entry.Status)
	}

	// Test GetByUserCode
	retrieved, err = storage.GetDeviceCodeByUserCode(ctx, "ABCD-EFGH")
	if err != nil {
		t.Fatalf("GetDeviceCodeByUserCode failed: %v", err)
	}
	if retrieved.DeviceCode != entry.DeviceCode {
		t.Errorf("device_code = %s, want %s", retrieved.DeviceCode, entry.DeviceCode)
	}

	// Test UpdateStatus
	entry.Status = DeviceCodeAuthorized
	entry.AuthorizedBy = "user123"
	err = storage.UpdateDeviceCodeStatus(ctx, "device-code-123", entry)
	if err != nil {
		t.Fatalf("UpdateDeviceCodeStatus failed: %v", err)
	}

	// Verify update
	updated, err := storage.GetDeviceCodeByDeviceCode(ctx, "device-code-123")
	if err != nil {
		t.Fatalf("GetDeviceCodeByDeviceCode after update failed: %v", err)
	}
	if updated.Status != DeviceCodeAuthorized {
		t.Errorf("status after update = %s, want %s", updated.Status, DeviceCodeAuthorized)
	}

	// Test Delete
	err = storage.DeleteDeviceCode(ctx, "device-code-123")
	if err != nil {
		t.Fatalf("DeleteDeviceCode failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetDeviceCodeByDeviceCode(ctx, "device-code-123")
	if err == nil {
		t.Error("device code should not exist after deletion")
	}
}

// TestInMemoryStorage_DeviceCodeCleanup tests cleanup of expired device codes.
func TestInMemoryStorage_DeviceCodeCleanup(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create expired device code
	expiredEntry := &DeviceCodeEntry{
		DeviceCode: "expired-device-code",
		UserCode:   "EXPIRED1",
		ClientID:   "test-client",
		IssuedAt:   time.Now().Add(-30 * time.Minute),
		ExpiresAt:  time.Now().Add(-15 * time.Minute),
	}
	_ = storage.StoreDeviceCode(ctx, expiredEntry)

	// Create valid device code
	validEntry := &DeviceCodeEntry{
		DeviceCode: "valid-device-code",
		UserCode:   "VALID001",
		ClientID:   "test-client",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}
	_ = storage.StoreDeviceCode(ctx, validEntry)

	// Cleanup
	count, err := storage.CleanupExpiredDeviceCodes(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredDeviceCodes failed: %v", err)
	}
	if count != 1 {
		t.Errorf("cleanup count = %d, want 1", count)
	}

	// Verify cleanup
	_, err = storage.GetDeviceCodeByDeviceCode(ctx, "expired-device-code")
	if err == nil {
		t.Error("expired device code should be removed")
	}

	_, err = storage.GetDeviceCodeByDeviceCode(ctx, "valid-device-code")
	if err != nil {
		t.Errorf("valid device code was removed: %v", err)
	}
}

// TestInMemoryStorage_PARRequests tests PAR request operations.
func TestInMemoryStorage_PARRequests(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create PAR request entry
	entry := &RequestURIEntry{
		RequestURI: "urn:ietf:params:oauth:request_uri:test123",
		ClientID:   "test-client",
		Request: &PushedAuthorizationRequest{
			ClientID:            "test-client",
			ResponseType:        "code",
			RedirectURI:         "https://example.com/callback",
			Scope:               "openid profile",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
	}

	// Test Store
	err := storage.StorePARRequest(ctx, entry.RequestURI, entry)
	if err != nil {
		t.Fatalf("StorePARRequest failed: %v", err)
	}

	// Test Get
	retrieved, err := storage.GetPARRequest(ctx, entry.RequestURI)
	if err != nil {
		t.Fatalf("GetPARRequest failed: %v", err)
	}
	if retrieved.RequestURI != entry.RequestURI {
		t.Errorf("request_uri = %s, want %s", retrieved.RequestURI, entry.RequestURI)
	}
	if retrieved.ClientID != entry.ClientID {
		t.Errorf("client_id = %s, want %s", retrieved.ClientID, entry.ClientID)
	}
	if retrieved.Used {
		t.Error("request should not be marked as used")
	}

	// Test MarkUsed
	err = storage.MarkPARRequestUsed(ctx, entry.RequestURI)
	if err != nil {
		t.Fatalf("MarkPARRequestUsed failed: %v", err)
	}

	// Verify marked used
	marked, err := storage.GetPARRequest(ctx, entry.RequestURI)
	if err != nil {
		t.Fatalf("GetPARRequest after mark failed: %v", err)
	}
	if !marked.Used {
		t.Error("request should be marked as used")
	}

	// Test Delete
	err = storage.DeletePARRequest(ctx, entry.RequestURI)
	if err != nil {
		t.Fatalf("DeletePARRequest failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetPARRequest(ctx, entry.RequestURI)
	if err == nil {
		t.Error("PAR request should not exist after deletion")
	}
}

// TestInMemoryStorage_PARCleanup tests cleanup of expired PAR requests.
func TestInMemoryStorage_PARCleanup(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create expired PAR request
	expiredEntry := &RequestURIEntry{
		RequestURI: "urn:ietf:params:oauth:request_uri:expired",
		ClientID:   "test-client",
		Request:    &PushedAuthorizationRequest{ClientID: "test-client"},
		CreatedAt:  time.Now().Add(-10 * time.Minute),
		ExpiresAt:  time.Now().Add(-5 * time.Minute),
	}
	_ = storage.StorePARRequest(ctx, expiredEntry.RequestURI, expiredEntry)

	// Create valid PAR request
	validEntry := &RequestURIEntry{
		RequestURI: "urn:ietf:params:oauth:request_uri:valid",
		ClientID:   "test-client",
		Request:    &PushedAuthorizationRequest{ClientID: "test-client"},
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	_ = storage.StorePARRequest(ctx, validEntry.RequestURI, validEntry)

	// Cleanup
	count, err := storage.CleanupExpiredPARRequests(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredPARRequests failed: %v", err)
	}
	if count != 1 {
		t.Errorf("cleanup count = %d, want 1", count)
	}

	// Verify cleanup
	_, err = storage.GetPARRequest(ctx, expiredEntry.RequestURI)
	if err == nil {
		t.Error("expired PAR request should be removed")
	}

	_, err = storage.GetPARRequest(ctx, validEntry.RequestURI)
	if err != nil {
		t.Errorf("valid PAR request was removed: %v", err)
	}
}

// TestInMemoryStorage_Ping tests health check.
func TestInMemoryStorage_Ping(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	err := storage.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

// TestInMemoryStorage_Close tests connection cleanup.
func TestInMemoryStorage_Close(t *testing.T) {
	storage := NewInMemoryStorage()

	err := storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestInMemoryStorage_Concurrent tests concurrent operations.
func TestInMemoryStorage_Concurrent(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Test concurrent refresh token operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			token := &RefreshTokenEntry{
				RefreshToken: "token-" + string(rune('0'+id)),
				ProviderID:   "test-client",
				Subject:      "user123",
				IssuedAt:     time.Now(),
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			}
			_ = storage.StoreRefreshToken(ctx, token)
			_, _ = storage.GetRefreshToken(ctx, token.RefreshToken)
			_ = storage.DeleteRefreshToken(ctx, token.RefreshToken)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestInMemoryStorage_MultipleUsers tests isolation between users.
func TestInMemoryStorage_MultipleUsers(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create tokens for multiple users
	users := []string{"user1", "user2", "user3"}
	for i, userID := range users {
		token := &RefreshTokenEntry{
			RefreshToken: "token-" + string(rune('0'+i)),
			ProviderID:   "test-client",
			Subject:      userID,
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		_ = storage.StoreRefreshToken(ctx, token)
	}

	// Verify each user only sees their own tokens
	for _, userID := range users {
		tokens, err := storage.ListRefreshTokensByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListRefreshTokensByUser for %s failed: %v", userID, err)
		}
		if len(tokens) != 1 {
			t.Errorf("user %s has %d tokens, want 1", userID, len(tokens))
		}
		if tokens[0].Subject != userID {
			t.Errorf("token subject = %s, want %s", tokens[0].Subject, userID)
		}
	}
}

// TestInMemoryStorage_MultipleClients tests isolation between clients.
func TestInMemoryStorage_MultipleClients(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create tokens for multiple clients
	clients := []string{"client1", "client2", "client3"}
	for i, clientID := range clients {
		token := &RefreshTokenEntry{
			RefreshToken: "token-" + string(rune('0'+i)),
			ProviderID:   clientID,
			Subject:      "user123",
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		_ = storage.StoreRefreshToken(ctx, token)
	}

	// Verify each client only sees their own tokens
	for _, clientID := range clients {
		tokens, err := storage.ListRefreshTokensByClient(ctx, clientID)
		if err != nil {
			t.Fatalf("ListRefreshTokensByClient for %s failed: %v", clientID, err)
		}
		if len(tokens) != 1 {
			t.Errorf("client %s has %d tokens, want 1", clientID, len(tokens))
		}
		if tokens[0].ProviderID != clientID {
			t.Errorf("token provider_id = %s, want %s", tokens[0].ProviderID, clientID)
		}
	}
}

// TestInMemoryStorage_DeviceCodeUserCodeMapping tests device/user code mapping.
func TestInMemoryStorage_DeviceCodeUserCodeMapping(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create device code
	entry := &DeviceCodeEntry{
		DeviceCode: "device-123",
		UserCode:   "USER-123",
		ClientID:   "test-client",
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}
	_ = storage.StoreDeviceCode(ctx, entry)

	// Verify both lookups work
	byDevice, err := storage.GetDeviceCodeByDeviceCode(ctx, "device-123")
	if err != nil {
		t.Fatalf("GetDeviceCodeByDeviceCode failed: %v", err)
	}

	byUser, err := storage.GetDeviceCodeByUserCode(ctx, "USER-123")
	if err != nil {
		t.Fatalf("GetDeviceCodeByUserCode failed: %v", err)
	}

	if byDevice.DeviceCode != byUser.DeviceCode {
		t.Error("device code lookups returned different entries")
	}
	if byDevice.UserCode != byUser.UserCode {
		t.Error("user code lookups returned different entries")
	}

	// Delete and verify both removed
	_ = storage.DeleteDeviceCode(ctx, "device-123")

	_, err = storage.GetDeviceCodeByDeviceCode(ctx, "device-123")
	if err == nil {
		t.Error("device code still exists after deletion")
	}

	_, err = storage.GetDeviceCodeByUserCode(ctx, "USER-123")
	if err == nil {
		t.Error("user code still exists after deletion")
	}
}

// TestInMemoryStorage_EmptyResults tests operations with no data.
func TestInMemoryStorage_EmptyResults(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// List operations should return empty, not error
	tokens, err := storage.ListRefreshTokensByUser(ctx, "nonexistent")
	if err != nil {
		t.Errorf("ListRefreshTokensByUser with no data failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected empty list, got %d items", len(tokens))
	}

	tokens, err = storage.ListRefreshTokensByClient(ctx, "nonexistent")
	if err != nil {
		t.Errorf("ListRefreshTokensByClient with no data failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected empty list, got %d items", len(tokens))
	}

	// Cleanup operations should return 0, not error
	count, err := storage.CleanupExpiredRefreshTokens(ctx)
	if err != nil {
		t.Errorf("CleanupExpiredRefreshTokens with no data failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned, got %d", count)
	}

	count, err = storage.CleanupExpiredRevocations(ctx)
	if err != nil {
		t.Errorf("CleanupExpiredRevocations with no data failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned, got %d", count)
	}

	count, err = storage.CleanupExpiredDeviceCodes(ctx)
	if err != nil {
		t.Errorf("CleanupExpiredDeviceCodes with no data failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned, got %d", count)
	}

	count, err = storage.CleanupExpiredPARRequests(ctx)
	if err != nil {
		t.Errorf("CleanupExpiredPARRequests with no data failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned, got %d", count)
	}
}
