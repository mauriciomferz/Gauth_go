package oidc

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestPARService_PushAuthorizationRequest tests PAR request creation.
func TestPARService_PushAuthorizationRequest(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	tests := []struct {
		name    string
		req     *PushedAuthorizationRequest
		wantErr bool
		errCode string
	}{
		{
			name: "valid request with PKCE",
			req: &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        "code",
				RedirectURI:         "https://example.com/callback",
				Scope:               "openid profile",
				State:               "random-state",
				CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				CodeChallengeMethod: "S256",
			},
			wantErr: false,
		},
		{
			name: "valid request with minimal fields",
			req: &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        "code",
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				CodeChallengeMethod: "S256",
			},
			wantErr: false,
		},
		{
			name: "missing client_id",
			req: &PushedAuthorizationRequest{
				ResponseType:        "code",
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       "challenge",
				CodeChallengeMethod: "S256",
			},
			wantErr: true,
			errCode: ErrorInvalidRequest,
		},
		{
			name: "missing response_type",
			req: &PushedAuthorizationRequest{
				ClientID:            "test-client",
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       "challenge",
				CodeChallengeMethod: "S256",
			},
			wantErr: true,
			errCode: ErrorInvalidRequest,
		},
		{
			name: "missing redirect_uri when required",
			req: &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        "code",
				CodeChallenge:       "challenge",
				CodeChallengeMethod: "S256",
			},
			wantErr: true,
			errCode: ErrorInvalidRequest,
		},
		{
			name: "missing PKCE when required",
			req: &PushedAuthorizationRequest{
				ClientID:     "test-client",
				ResponseType: "code",
				RedirectURI:  "https://example.com/callback",
			},
			wantErr: true,
			errCode: ErrorInvalidRequest,
		},
		{
			name: "invalid response_type",
			req: &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        "invalid",
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       "challenge",
				CodeChallengeMethod: "S256",
			},
			wantErr: true,
			errCode: ErrorUnsupportedResponseType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.PushAuthorizationRequest(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				parErr, ok := err.(*PushedAuthorizationError)
				if !ok {
					t.Fatalf("expected PushedAuthorizationError, got %T", err)
				}
				if tt.errCode != "" && parErr.ErrorCode != tt.errCode {
					t.Errorf("error_code = %s, want %s", parErr.ErrorCode, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate response
			if resp.RequestURI == "" {
				t.Error("request_uri is empty")
			}
			if !strings.HasPrefix(resp.RequestURI, "urn:ietf:params:oauth:request_uri:") {
				t.Errorf("request_uri doesn't have correct prefix: %s", resp.RequestURI)
			}
			if resp.ExpiresIn <= 0 {
				t.Errorf("expires_in = %d, want > 0", resp.ExpiresIn)
			}
			if resp.ExpiresIn != int(DefaultPARConfig().RequestURILifetime.Seconds()) {
				t.Errorf("expires_in = %d, want %d", resp.ExpiresIn, int(DefaultPARConfig().RequestURILifetime.Seconds()))
			}
		})
	}
}

// TestPARService_RequestURIGeneration tests request URI generation uniqueness.
func TestPARService_RequestURIGeneration(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	uris := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := &PushedAuthorizationRequest{
			ClientID:            "test-client",
			ResponseType:        "code",
			RedirectURI:         "https://example.com/callback",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "S256",
		}
		resp, err := service.PushAuthorizationRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to push authorization request: %v", err)
		}

		// Check uniqueness
		if uris[resp.RequestURI] {
			t.Errorf("duplicate request_uri generated: %s", resp.RequestURI)
		}
		uris[resp.RequestURI] = true

		// Check format
		if !strings.HasPrefix(resp.RequestURI, "urn:ietf:params:oauth:request_uri:") {
			t.Errorf("request_uri has incorrect format: %s", resp.RequestURI)
		}

		// Check minimum entropy (should be at least 32 characters after prefix)
		uriParts := strings.Split(resp.RequestURI, ":")
		if len(uriParts) < 5 {
			t.Errorf("request_uri has too few parts: %s", resp.RequestURI)
		}
		identifier := uriParts[len(uriParts)-1]
		if len(identifier) < 32 {
			t.Errorf("request_uri identifier too short: %d bytes, want >= 32", len(identifier))
		}
	}
}

// TestPARService_GetAuthorizationRequest tests retrieval of PAR requests.
func TestPARService_GetAuthorizationRequest(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		Scope:               "openid profile email",
		State:               "test-state",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
		Nonce:               "test-nonce",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// Retrieve request
	getReq, err := service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err != nil {
		t.Fatalf("failed to get authorization request: %v", err)
	}

	// Validate retrieved request
	if getReq.ClientID != pushReq.ClientID {
		t.Errorf("client_id = %s, want %s", getReq.ClientID, pushReq.ClientID)
	}
	if getReq.ResponseType != pushReq.ResponseType {
		t.Errorf("response_type = %s, want %s", getReq.ResponseType, pushReq.ResponseType)
	}
	if getReq.RedirectURI != pushReq.RedirectURI {
		t.Errorf("redirect_uri = %s, want %s", getReq.RedirectURI, pushReq.RedirectURI)
	}
	if getReq.Scope != pushReq.Scope {
		t.Errorf("scope = %s, want %s", getReq.Scope, pushReq.Scope)
	}
	if getReq.State != pushReq.State {
		t.Errorf("state = %s, want %s", getReq.State, pushReq.State)
	}
	if getReq.CodeChallenge != pushReq.CodeChallenge {
		t.Errorf("code_challenge = %s, want %s", getReq.CodeChallenge, pushReq.CodeChallenge)
	}
	if getReq.CodeChallengeMethod != pushReq.CodeChallengeMethod {
		t.Errorf("code_challenge_method = %s, want %s", getReq.CodeChallengeMethod, pushReq.CodeChallengeMethod)
	}
	if getReq.Nonce != pushReq.Nonce {
		t.Errorf("nonce = %s, want %s", getReq.Nonce, pushReq.Nonce)
	}
}

// TestPARService_GetAuthorizationRequest_NotFound tests not found error.
func TestPARService_GetAuthorizationRequest_NotFound(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	_, err := service.GetAuthorizationRequest(context.Background(), "urn:ietf:params:oauth:request_uri:nonexistent", "test-client")
	if err == nil {
		t.Fatal("expected error but got none")
	}

	parErr, ok := err.(*PushedAuthorizationError)
	if !ok {
		t.Fatalf("expected PushedAuthorizationError, got %T", err)
	}
	if parErr.ErrorCode != ErrorInvalidRequest {
		t.Errorf("error_code = %s, want %s", parErr.ErrorCode, ErrorInvalidRequest)
	}
}

// TestPARService_GetAuthorizationRequest_WrongClient tests client mismatch.
func TestPARService_GetAuthorizationRequest_WrongClient(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// Try to retrieve with different client
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "wrong-client")
	if err == nil {
		t.Fatal("expected error but got none")
	}

	parErr, ok := err.(*PushedAuthorizationError)
	if !ok {
		t.Fatalf("expected PushedAuthorizationError, got %T", err)
	}
	if parErr.ErrorCode != ErrorInvalidClient {
		t.Errorf("error_code = %s, want %s", parErr.ErrorCode, ErrorInvalidClient)
	}
}

// TestPARService_SingleUseEnforcement tests request URI can only be used once.
func TestPARService_SingleUseEnforcement(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// First retrieval - should succeed
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err != nil {
		t.Fatalf("first retrieval failed: %v", err)
	}

	// Second retrieval - should fail (single-use enforcement)
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err == nil {
		t.Fatal("expected error on second retrieval but got none")
	}

	parErr, ok := err.(*PushedAuthorizationError)
	if !ok {
		t.Fatalf("expected PushedAuthorizationError, got %T", err)
	}
	if parErr.ErrorCode != ErrorInvalidRequest {
		t.Errorf("error_code = %s, want %s", parErr.ErrorCode, ErrorInvalidRequest)
	}
	if !strings.Contains(parErr.ErrorDescription, "already been used") {
		t.Errorf("error_description = %s, want 'already been used'", parErr.ErrorDescription)
	}
}

// TestPARService_Expiration tests request URI expiration.
func TestPARService_Expiration(t *testing.T) {
	config := DefaultPARConfig()
	config.RequestURILifetime = 100 * time.Millisecond
	service := NewPARService(config)
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to retrieve - should fail
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err == nil {
		t.Fatal("expected error after expiration but got none")
	}

	parErr, ok := err.(*PushedAuthorizationError)
	if !ok {
		t.Fatalf("expected PushedAuthorizationError, got %T", err)
	}
	if parErr.ErrorCode != ErrorInvalidRequest {
		t.Errorf("error_code = %s, want %s", parErr.ErrorCode, ErrorInvalidRequest)
	}
	if !strings.Contains(parErr.ErrorDescription, "expired") {
		t.Errorf("error_description = %s, want 'expired'", parErr.ErrorDescription)
	}
}

// TestPARService_Cleanup tests expired request URI cleanup.
func TestPARService_Cleanup(t *testing.T) {
	config := DefaultPARConfig()
	config.RequestURILifetime = 50 * time.Millisecond
	service := NewPARService(config)
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Trigger cleanup
	service.cleanupExpiredRequests()

	// Check if request was removed
	service.mu.RLock()
	_, exists := service.requests[pushResp.RequestURI]
	service.mu.RUnlock()

	if exists {
		t.Error("expired request_uri was not cleaned up")
	}
}

// TestPARService_PKCEValidation tests PKCE requirement enforcement.
func TestPARService_PKCEValidation(t *testing.T) {
	config := DefaultPARConfig()
	config.RequirePKCE = true
	service := NewPARService(config)
	defer service.Stop()

	tests := []struct {
		name                string
		codeChallenge       string
		codeChallengeMethod string
		wantErr             bool
	}{
		{
			name:                "valid S256",
			codeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallengeMethod: "S256",
			wantErr:             false,
		},
		{
			name:                "valid plain",
			codeChallenge:       "test-challenge",
			codeChallengeMethod: "plain",
			wantErr:             false,
		},
		{
			name:                "missing code_challenge",
			codeChallenge:       "",
			codeChallengeMethod: "S256",
			wantErr:             true,
		},
		{
			name:                "missing code_challenge_method defaults to plain",
			codeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallengeMethod: "",
			wantErr:             false, // Defaults to "plain" per RFC 7636
		},
		{
			name:                "invalid code_challenge_method",
			codeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallengeMethod: "invalid",
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        "code",
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       tt.codeChallenge,
				CodeChallengeMethod: tt.codeChallengeMethod,
			}
			_, err := service.PushAuthorizationRequest(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestPARService_ResponseTypeValidation tests response_type validation.
func TestPARService_ResponseTypeValidation(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	tests := []struct {
		name         string
		responseType string
		wantErr      bool
	}{
		{
			name:         "valid code",
			responseType: "code",
			wantErr:      false,
		},
		{
			name:         "valid code id_token",
			responseType: "code id_token",
			wantErr:      false,
		},
		{
			name:         "valid id_token token",
			responseType: "id_token token",
			wantErr:      false,
		},
		{
			name:         "invalid token",
			responseType: "token",
			wantErr:      true,
		},
		{
			name:         "invalid custom",
			responseType: "custom",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        tt.responseType,
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				CodeChallengeMethod: "S256",
			}
			_, err := service.PushAuthorizationRequest(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestPARService_ConcurrentRequests tests concurrent PAR request creation.
func TestPARService_ConcurrentRequests(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex
	uris := make(map[string]bool)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := &PushedAuthorizationRequest{
				ClientID:            "test-client",
				ResponseType:        "code",
				RedirectURI:         "https://example.com/callback",
				CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				CodeChallengeMethod: "S256",
			}
			resp, err := service.PushAuthorizationRequest(context.Background(), req)
			if err == nil {
				mu.Lock()
				successCount++
				if uris[resp.RequestURI] {
					t.Errorf("duplicate request_uri in concurrent test: %s", resp.RequestURI)
				}
				uris[resp.RequestURI] = true
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 50 {
		t.Errorf("success count = %d, want 50", successCount)
	}
}

// TestPARService_RevokeRequestURI tests request URI revocation.
func TestPARService_RevokeRequestURI(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// Revoke request URI
	err = service.RevokeRequestURI(context.Background(), pushResp.RequestURI)
	if err != nil {
		t.Fatalf("failed to revoke request_uri: %v", err)
	}

	// Try to retrieve - should fail (marked as used)
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err == nil {
		t.Fatal("expected error after revocation but got none")
	}
}

// TestPARService_MultipleClients tests isolation between different clients.
func TestPARService_MultipleClients(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	clients := []string{"client1", "client2", "client3"}
	requestURIs := make(map[string]string) // clientID -> requestURI

	// Create PAR requests for each client
	for _, clientID := range clients {
		req := &PushedAuthorizationRequest{
			ClientID:            clientID,
			ResponseType:        "code",
			RedirectURI:         "https://example.com/callback",
			Scope:               clientID + "-scope",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "S256",
		}
		resp, err := service.PushAuthorizationRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to push authorization request for %s: %v", clientID, err)
		}
		requestURIs[clientID] = resp.RequestURI
	}

	// Verify each client can only access their own request
	for _, clientID := range clients {
		requestURI := requestURIs[clientID]

		// Should succeed with correct client
		req, err := service.GetAuthorizationRequest(context.Background(), requestURI, clientID)
		if err != nil {
			t.Errorf("client %s failed to retrieve their own request: %v", clientID, err)
		}
		if req.Scope != clientID+"-scope" {
			t.Errorf("retrieved wrong request for client %s", clientID)
		}

		// Should fail with other clients (already used due to single-use)
		// Create new request for cross-client test
		newReq := &PushedAuthorizationRequest{
			ClientID:            clientID,
			ResponseType:        "code",
			RedirectURI:         "https://example.com/callback",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "S256",
		}
		newResp, _ := service.PushAuthorizationRequest(context.Background(), newReq)

		for _, otherClientID := range clients {
			if otherClientID != clientID {
				_, err := service.GetAuthorizationRequest(context.Background(), newResp.RequestURI, otherClientID)
				if err == nil {
					t.Errorf("client %s should not access request from %s", otherClientID, clientID)
				}
			}
		}
	}
}

// TestPARService_WithoutPKCE tests service with PKCE disabled.
func TestPARService_WithoutPKCE(t *testing.T) {
	config := DefaultPARConfig()
	config.RequirePKCE = false
	service := NewPARService(config)
	defer service.Stop()

	req := &PushedAuthorizationRequest{
		ClientID:     "test-client",
		ResponseType: "code",
		RedirectURI:  "https://example.com/callback",
		// No PKCE parameters
	}
	resp, err := service.PushAuthorizationRequest(context.Background(), req)
	if err != nil {
		t.Errorf("request without PKCE should succeed when not required: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.RequestURI == "" {
		t.Error("request_uri is empty")
	}
}

// TestPARService_WithoutSingleUse tests service with single-use disabled.
func TestPARService_WithoutSingleUse(t *testing.T) {
	config := DefaultPARConfig()
	config.SingleUse = false
	service := NewPARService(config)
	defer service.Stop()

	// Create PAR request
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// First retrieval
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err != nil {
		t.Fatalf("first retrieval failed: %v", err)
	}

	// Second retrieval - should also succeed (no single-use enforcement)
	_, err = service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err != nil {
		t.Errorf("second retrieval failed when single-use disabled: %v", err)
	}
}

// TestPARService_Stop tests service shutdown.
func TestPARService_Stop(t *testing.T) {
	service := NewPARService(DefaultPARConfig())

	// Create some PAR requests
	for i := 0; i < 5; i++ {
		req := &PushedAuthorizationRequest{
			ClientID:            "test-client",
			ResponseType:        "code",
			RedirectURI:         "https://example.com/callback",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "S256",
		}
		_, err := service.PushAuthorizationRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to push authorization request: %v", err)
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

// TestPARService_OIDCParameters tests OIDC-specific parameters preservation.
func TestPARService_OIDCParameters(t *testing.T) {
	service := NewPARService(DefaultPARConfig())
	defer service.Stop()

	// Create PAR request with OIDC parameters
	pushReq := &PushedAuthorizationRequest{
		ClientID:            "test-client",
		ResponseType:        "code",
		RedirectURI:         "https://example.com/callback",
		Scope:               "openid profile email",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
		Nonce:               "test-nonce-12345",
		ResponseMode:        "query",
		Display:             "page",
		Prompt:              "consent",
		MaxAge:              3600,
		UILocales:           []string{"en-US", "es-ES"},
		LoginHint:           "user@example.com",
		ACRValues:           []string{"urn:mace:incommon:iap:silver", "urn:mace:incommon:iap:bronze"},
	}
	pushResp, err := service.PushAuthorizationRequest(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push authorization request: %v", err)
	}

	// Retrieve and verify OIDC parameters
	getReq, err := service.GetAuthorizationRequest(context.Background(), pushResp.RequestURI, "test-client")
	if err != nil {
		t.Fatalf("failed to get authorization request: %v", err)
	}

	if getReq.Nonce != pushReq.Nonce {
		t.Errorf("nonce = %s, want %s", getReq.Nonce, pushReq.Nonce)
	}
	if getReq.ResponseMode != pushReq.ResponseMode {
		t.Errorf("response_mode = %s, want %s", getReq.ResponseMode, pushReq.ResponseMode)
	}
	if getReq.Display != pushReq.Display {
		t.Errorf("display = %s, want %s", getReq.Display, pushReq.Display)
	}
	if getReq.Prompt != pushReq.Prompt {
		t.Errorf("prompt = %s, want %s", getReq.Prompt, pushReq.Prompt)
	}
	if getReq.MaxAge != pushReq.MaxAge {
		t.Errorf("max_age = %d, want %d", getReq.MaxAge, pushReq.MaxAge)
	}
	if len(getReq.UILocales) != len(pushReq.UILocales) {
		t.Errorf("ui_locales length = %d, want %d", len(getReq.UILocales), len(pushReq.UILocales))
	}
	if getReq.LoginHint != pushReq.LoginHint {
		t.Errorf("login_hint = %s, want %s", getReq.LoginHint, pushReq.LoginHint)
	}
	if len(getReq.ACRValues) != len(pushReq.ACRValues) {
		t.Errorf("acr_values length = %d, want %d", len(getReq.ACRValues), len(pushReq.ACRValues))
	}
}
