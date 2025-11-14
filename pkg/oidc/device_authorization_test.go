package oidc

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestDeviceAuthorizationService_AuthorizeDevice tests device code creation.
func TestDeviceAuthorizationService_AuthorizeDevice(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	tests := []struct {
		name    string
		req     *DeviceAuthorizationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: &DeviceAuthorizationRequest{
				ClientID: "test-client",
				Scope:    "openid profile",
			},
			wantErr: false,
		},
		{
			name: "valid request without scope",
			req: &DeviceAuthorizationRequest{
				ClientID: "test-client",
			},
			wantErr: false,
		},
		{
			name: "missing client_id",
			req: &DeviceAuthorizationRequest{
				Scope: "openid",
			},
			wantErr: true,
			errMsg:  "client_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.AuthorizeDevice(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want substring %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate response
			if resp.DeviceCode == "" {
				t.Error("device_code is empty")
			}
			if resp.UserCode == "" {
				t.Error("user_code is empty")
			}
			if resp.VerificationURI == "" {
				t.Error("verification_uri is empty")
			}
			if resp.ExpiresIn <= 0 {
				t.Errorf("expires_in = %d, want > 0", resp.ExpiresIn)
			}
			if resp.Interval <= 0 {
				t.Errorf("interval = %d, want > 0", resp.Interval)
			}
			if resp.VerificationURIComplete == "" {
				t.Error("verification_uri_complete is empty")
			}
			if !strings.Contains(resp.VerificationURIComplete, resp.UserCode) {
				t.Errorf("verification_uri_complete doesn't contain user_code")
			}
		})
	}
}

// TestDeviceAuthorizationService_UserCodeGeneration tests user code generation.
func TestDeviceAuthorizationService_UserCodeGeneration(t *testing.T) {
	config := DefaultDeviceAuthorizationConfig()
	service := NewDeviceAuthorizationService(config)
	defer service.Stop()

	// Generate multiple user codes
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := &DeviceAuthorizationRequest{
			ClientID: "test-client",
		}
		resp, err := service.AuthorizeDevice(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to authorize device: %v", err)
		}

		// Check uniqueness
		if codes[resp.UserCode] {
			t.Errorf("duplicate user_code generated: %s", resp.UserCode)
		}
		codes[resp.UserCode] = true

		// Check format (includes dash separator, so length is UserCodeLength + 1)
		// E.g., "ABCD-EFGH" for UserCodeLength=8
		expectedLen := config.UserCodeLength + 1
		if len(resp.UserCode) != expectedLen {
			t.Errorf("user_code length = %d, want %d", len(resp.UserCode), expectedLen)
		}

		// Check charset (should only contain allowed characters)
		for _, ch := range resp.UserCode {
			if !strings.ContainsRune(config.UserCodeCharset, ch) && ch != '-' {
				t.Errorf("user_code contains invalid character: %c", ch)
			}
		}
	}
}

// TestDeviceAuthorizationService_DeviceCodeGeneration tests device code generation.
func TestDeviceAuthorizationService_DeviceCodeGeneration(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Generate multiple device codes
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := &DeviceAuthorizationRequest{
			ClientID: "test-client",
		}
		resp, err := service.AuthorizeDevice(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to authorize device: %v", err)
		}

		// Check uniqueness
		if codes[resp.DeviceCode] {
			t.Errorf("duplicate device_code generated: %s", resp.DeviceCode)
		}
		codes[resp.DeviceCode] = true

		// Check minimum length (should be cryptographically secure)
		if len(resp.DeviceCode) < 32 {
			t.Errorf("device_code too short: %d bytes, want >= 32", len(resp.DeviceCode))
		}
	}
}

// TestDeviceAuthorizationService_PollToken_Pending tests polling before authorization.
func TestDeviceAuthorizationService_PollToken_Pending(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
		Scope:    "openid",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	// Poll before authorization
	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}
	_, err = service.PollToken(context.Background(), tokenReq)

	// Should get authorization_pending error
	if err == nil {
		t.Fatal("expected error but got none")
	}
	devErr, ok := err.(*DeviceAuthorizationError)
	if !ok {
		t.Fatalf("expected DeviceAuthorizationError, got %T", err)
	}
	if devErr.ErrorCode != ErrorAuthorizationPending {
		t.Errorf("error_code = %s, want %s", devErr.ErrorCode, ErrorAuthorizationPending)
	}
}

// TestDeviceAuthorizationService_PollToken_SlowDown tests slow_down error.
func TestDeviceAuthorizationService_PollToken_SlowDown(t *testing.T) {
	config := DefaultDeviceAuthorizationConfig()
	config.PollingInterval = 100 * time.Millisecond
	service := NewDeviceAuthorizationService(config)
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}

	// First poll
	service.PollToken(context.Background(), tokenReq)

	// Immediate second poll (should trigger slow_down)
	_, err = service.PollToken(context.Background(), tokenReq)
	if err == nil {
		t.Fatal("expected slow_down error but got none")
	}

	devErr, ok := err.(*DeviceAuthorizationError)
	if !ok {
		t.Fatalf("expected DeviceAuthorizationError, got %T", err)
	}
	if devErr.ErrorCode != ErrorSlowDown {
		t.Errorf("error_code = %s, want %s", devErr.ErrorCode, ErrorSlowDown)
	}
}

// TestDeviceAuthorizationService_PollToken_Expired tests expired device code.
func TestDeviceAuthorizationService_PollToken_Expired(t *testing.T) {
	config := DefaultDeviceAuthorizationConfig()
	config.DeviceCodeLifetime = 100 * time.Millisecond
	service := NewDeviceAuthorizationService(config)
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Poll after expiration
	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}
	_, err = service.PollToken(context.Background(), tokenReq)

	// Should get expired_token error
	if err == nil {
		t.Fatal("expected expired_token error but got none")
	}
	devErr, ok := err.(*DeviceAuthorizationError)
	if !ok {
		t.Fatalf("expected DeviceAuthorizationError, got %T", err)
	}
	if devErr.ErrorCode != ErrorDeviceCodeExpired {
		t.Errorf("error_code = %s, want %s", devErr.ErrorCode, ErrorDeviceCodeExpired)
	}
}

// TestDeviceAuthorizationService_ApproveDevice tests device approval.
func TestDeviceAuthorizationService_ApproveDevice(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
		Scope:    "openid profile",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	// Approve device (need to provide actual tokens)
	// For this test, we'll need to generate simple test tokens
	err = service.ApproveAuthorization(
		context.Background(),
		authResp.UserCode,
		"user123",
		"test-access-token",
		"test-refresh-token",
		"test-id-token",
	)
	if err != nil {
		t.Fatalf("failed to approve device: %v", err)
	}

	// Poll for token (should succeed now)
	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}
	tokenResp, err := service.PollToken(context.Background(), tokenReq)
	if err != nil {
		t.Fatalf("failed to poll token: %v", err)
	}

	// Validate response
	if tokenResp.AccessToken == "" {
		t.Error("access_token is empty")
	}
	if tokenResp.TokenType != "Bearer" {
		t.Errorf("token_type = %s, want Bearer", tokenResp.TokenType)
	}
	if tokenResp.ExpiresIn <= 0 {
		t.Errorf("expires_in = %d, want > 0", tokenResp.ExpiresIn)
	}
}

// TestDeviceAuthorizationService_DenyDevice tests device denial.
func TestDeviceAuthorizationService_DenyDevice(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	// Deny device
	err = service.DenyAuthorization(context.Background(), authResp.UserCode, "user123")
	if err != nil {
		t.Fatalf("failed to deny device: %v", err)
	}

	// Poll for token (should get access_denied)
	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}
	_, err = service.PollToken(context.Background(), tokenReq)

	// Should get access_denied error
	if err == nil {
		t.Fatal("expected access_denied error but got none")
	}
	devErr, ok := err.(*DeviceAuthorizationError)
	if !ok {
		t.Fatalf("expected DeviceAuthorizationError, got %T", err)
	}
	if devErr.ErrorCode != ErrorAccessDenied {
		t.Errorf("error_code = %s, want %s", devErr.ErrorCode, ErrorAccessDenied)
	}
}

// TestDeviceAuthorizationService_GetDeviceCodeByUserCode tests retrieval by user code.
func TestDeviceAuthorizationService_GetDeviceCodeByUserCode(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
		Scope:    "openid",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	// Retrieve by user code
	entry, err := service.VerifyUserCode(context.Background(), authResp.UserCode)
	if err != nil {
		t.Fatalf("failed to get device code: %v", err)
	}

	// Validate entry
	if entry.DeviceCode != authResp.DeviceCode {
		t.Errorf("device_code = %s, want %s", entry.DeviceCode, authResp.DeviceCode)
	}
	if entry.UserCode != authResp.UserCode {
		t.Errorf("user_code = %s, want %s", entry.UserCode, authResp.UserCode)
	}
	if entry.ClientID != "test-client" {
		t.Errorf("client_id = %s, want test-client", entry.ClientID)
	}
	if entry.Status != DeviceCodePending {
		t.Errorf("status = %s, want %s", entry.Status, DeviceCodePending)
	}
}

// TestDeviceAuthorizationService_VerifyUserCode_NotFound tests not found error.
func TestDeviceAuthorizationService_VerifyUserCode_NotFound(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	_, err := service.VerifyUserCode(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %v, want 'invalid user code'", err)
	}
}

// TestDeviceAuthorizationService_MaxPollAttempts tests maximum poll attempts.
func TestDeviceAuthorizationService_MaxPollAttempts(t *testing.T) {
	config := DefaultDeviceAuthorizationConfig()
	config.MaxPollAttempts = 3
	config.PollingInterval = 10 * time.Millisecond
	service := NewDeviceAuthorizationService(config)
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}

	// Poll until max attempts exceeded
	for i := 0; i < config.MaxPollAttempts; i++ {
		time.Sleep(config.PollingInterval)
		service.PollToken(context.Background(), tokenReq)
	}

	// Next poll should fail with error
	time.Sleep(config.PollingInterval)
	_, err = service.PollToken(context.Background(), tokenReq)
	if err == nil {
		t.Fatal("expected error after max poll attempts but got none")
	}
}

// TestDeviceAuthorizationService_Cleanup tests expired code cleanup.
func TestDeviceAuthorizationService_Cleanup(t *testing.T) {
	config := DefaultDeviceAuthorizationConfig()
	config.DeviceCodeLifetime = 50 * time.Millisecond
	service := NewDeviceAuthorizationService(config)
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Trigger cleanup
	service.cleanupExpiredCodes()

	// Code should be removed
	service.mu.RLock()
	_, existsDevice := service.deviceCodes[authResp.DeviceCode]
	_, existsUser := service.userCodes[authResp.UserCode]
	service.mu.RUnlock()

	if existsDevice {
		t.Error("expired device_code was not cleaned up")
	}
	if existsUser {
		t.Error("expired user_code was not cleaned up")
	}
}

// TestDeviceAuthorizationService_ConcurrentPolling tests concurrent token polling.
func TestDeviceAuthorizationService_ConcurrentPolling(t *testing.T) {
	config := DefaultDeviceAuthorizationConfig()
	config.PollingInterval = 10 * time.Millisecond
	service := NewDeviceAuthorizationService(config)
	defer service.Stop()

	// Create device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}

	// Simulate concurrent polling from same device
	var wg sync.WaitGroup
	errorCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(20 * time.Millisecond)
			_, err := service.PollToken(context.Background(), tokenReq)
			if err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All should get authorization_pending error (no race conditions)
	if errorCount != 10 {
		t.Errorf("error count = %d, want 10", errorCount)
	}
}

// TestDeviceAuthorizationService_ApproveInvalidUserCode tests approving invalid user code.
func TestDeviceAuthorizationService_ApproveInvalidUserCode(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	err := service.ApproveAuthorization(context.Background(), "INVALID", "user123", "token", "refresh", "id")
	if err == nil {
		t.Fatal("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %v, want 'invalid user code'", err)
	}
}

// TestDeviceAuthorizationService_DenyInvalidUserCode tests denying invalid user code.
func TestDeviceAuthorizationService_DenyInvalidUserCode(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	err := service.DenyAuthorization(context.Background(), "INVALID", "user123")
	if err == nil {
		t.Fatal("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %v, want 'invalid user code'", err)
	}
}

// TestDeviceAuthorizationService_DoubleApprove tests approving already approved code.
func TestDeviceAuthorizationService_DoubleApprove(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Create and approve device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	err = service.ApproveAuthorization(context.Background(), authResp.UserCode, "user123", "token1", "refresh1", "id1")
	if err != nil {
		t.Fatalf("failed to approve device: %v", err)
	}

	// Try to approve again
	err = service.ApproveAuthorization(context.Background(), authResp.UserCode, "user456", "token2", "refresh2", "id2")
	if err == nil {
		t.Fatal("expected error when approving already approved code")
	}
}

// TestDeviceAuthorizationService_PollAfterTokenIssued tests polling after token issuance.
func TestDeviceAuthorizationService_PollAfterTokenIssued(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())
	defer service.Stop()

	// Create and approve device code
	authReq := &DeviceAuthorizationRequest{
		ClientID: "test-client",
	}
	authResp, err := service.AuthorizeDevice(context.Background(), authReq)
	if err != nil {
		t.Fatalf("failed to authorize device: %v", err)
	}

	err = service.ApproveAuthorization(context.Background(), authResp.UserCode, "user123", "token", "refresh", "id")
	if err != nil {
		t.Fatalf("failed to approve device: %v", err)
	}

	tokenReq := &DeviceTokenRequest{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: authResp.DeviceCode,
		ClientID:   "test-client",
	}

	// First poll - should succeed
	_, err = service.PollToken(context.Background(), tokenReq)
	if err != nil {
		t.Fatalf("first poll failed: %v", err)
	}

	// Second poll - should fail (device code already used)
	_, err = service.PollToken(context.Background(), tokenReq)
	if err == nil {
		t.Fatal("expected error when polling used device code")
	}
}

// TestDeviceAuthorizationService_Stop tests service shutdown.
func TestDeviceAuthorizationService_Stop(t *testing.T) {
	service := NewDeviceAuthorizationService(DefaultDeviceAuthorizationConfig())

	// Create some device codes
	for i := 0; i < 5; i++ {
		authReq := &DeviceAuthorizationRequest{
			ClientID: "test-client",
		}
		_, err := service.AuthorizeDevice(context.Background(), authReq)
		if err != nil {
			t.Fatalf("failed to authorize device: %v", err)
		}
	}

	// Stop service
	service.Stop()

	// Verify cleanup ticker stopped
	select {
	case <-service.stopCleanup:
		// Good - channel is closed
	default:
		t.Error("stopCleanup channel not closed")
	}
}
