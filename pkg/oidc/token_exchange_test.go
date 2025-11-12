package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupTokenExchangeTest(t *testing.T) (*TokenExchangeService, *InMemoryProviderRegistry, *rsa.PrivateKey) {
	t.Helper()

	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create ID token service
	idTokenService, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:     "https://gauth.example.com",
		SigningKey:    privateKey,
		SigningMethod: "RS256",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	// Create provider registry
	registry := NewInMemoryProviderRegistry()

	// Register test providers
	googleProvider := ProviderConfig{
		ID:                "google",
		Name:              "Google",
		IssuerURL:         "https://accounts.google.com",
		ClientID:          "test-google-client",
		ClientSecret:      "test-google-secret",
		Scopes:            []string{"openid", "profile", "email"},
		ClaimMappings:     map[string]string{"sub": "user_id", "email": "email"},
		DefaultTrustLevel: "substantial",
		Enabled:           true,
	}
	if err := registry.Register(googleProvider); err != nil {
		t.Fatalf("Failed to register Google provider: %v", err)
	}

	oktaProvider := ProviderConfig{
		ID:                "okta",
		Name:              "Okta",
		IssuerURL:         "https://dev-12345.okta.com",
		ClientID:          "test-okta-client",
		ClientSecret:      "test-okta-secret",
		Scopes:            []string{"openid", "profile", "email"},
		ClaimMappings:     map[string]string{"sub": "user_id", "email": "email"},
		DefaultTrustLevel: "substantial",
		Enabled:           true,
	}
	if err := registry.Register(oktaProvider); err != nil {
		t.Fatalf("Failed to register Okta provider: %v", err)
	}

	azureProvider := ProviderConfig{
		ID:                "azure_ad",
		Name:              "Azure AD",
		IssuerURL:         "https://login.microsoftonline.com/common/v2.0",
		ClientID:          "test-azure-client",
		ClientSecret:      "test-azure-secret",
		Scopes:            []string{"openid", "profile", "email"},
		ClaimMappings:     map[string]string{"oid": "user_id", "email": "email"},
		DefaultTrustLevel: "substantial",
		Enabled:           true,
	}
	if err := registry.Register(azureProvider); err != nil {
		t.Fatalf("Failed to register Azure AD provider: %v", err)
	}

	// Create token exchange service
	service, err := NewTokenExchangeService(TokenExchangeConfig{
		ProviderRegistry: registry,
		IDTokenService:   idTokenService,
	})
	if err != nil {
		t.Fatalf("Failed to create token exchange service: %v", err)
	}

	return service, registry, privateKey
}

func TestNewTokenExchangeService(t *testing.T) {
	tests := []struct {
		name        string
		config      TokenExchangeConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: TokenExchangeConfig{
				ProviderRegistry: NewInMemoryProviderRegistry(),
				IDTokenService:   &IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Missing provider registry",
			config: TokenExchangeConfig{
				IDTokenService: &IDTokenService{},
			},
			expectError: true,
			errorMsg:    "provider registry is required",
		},
		{
			name: "Missing ID token service",
			config: TokenExchangeConfig{
				ProviderRegistry: NewInMemoryProviderRegistry(),
			},
			expectError: true,
			errorMsg:    "ID token service is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewTokenExchangeService(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q but got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service == nil {
				t.Fatal("Expected service but got nil")
			}
		})
	}
}

func TestTokenExchangeService_ExchangeToken(t *testing.T) {
	service, _, _ := setupTokenExchangeTest(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     ExchangeRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing provider ID",
			request: ExchangeRequest{
				ExternalToken: "test-token",
				Audience:      "test-audience",
			},
			expectError: true,
			errorMsg:    "provider ID is required",
		},
		{
			name: "Missing external token",
			request: ExchangeRequest{
				ProviderID: "google",
				Audience:   "test-audience",
			},
			expectError: true,
			errorMsg:    "external token is required",
		},
		{
			name: "Missing audience",
			request: ExchangeRequest{
				ProviderID:    "google",
				ExternalToken: "test-token",
			},
			expectError: true,
			errorMsg:    "audience is required",
		},
		{
			name: "Unknown provider",
			request: ExchangeRequest{
				ProviderID:    "unknown",
				ExternalToken: "test-token",
				Audience:      "test-audience",
			},
			expectError: true,
		},
		{
			name: "Valid request (will fail validation)",
			request: ExchangeRequest{
				ProviderID:    "google",
				ExternalToken: "test-token",
				Audience:      "test-audience",
			},
			expectError: true, // Will fail because validateExternalToken is not implemented
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.ExchangeToken(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q but got %q", tt.errorMsg, err.Error())
				}
				if response != nil {
					t.Errorf("Expected nil response but got %+v", response)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestTokenExchangeService_NormalizeClaims(t *testing.T) {
	service, registry, _ := setupTokenExchangeTest(t)

	provider, _ := registry.Get("google")

	externalClaims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "external-user-123",
			Issuer:    "https://accounts.google.com",
			Audience:  jwt.ClaimStrings{"test-client"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Name:              "John Doe",
		GivenName:         "John",
		FamilyName:        "Doe",
		Email:             "john.doe@example.com",
		EmailVerified:     true,
		PreferredUsername: "johndoe",
		Picture:           "https://example.com/photo.jpg",
		Locale:            "en-US",
		ACR:               "substantial",
		AMR:               []string{"pwd", "mfa"},
	}

	gauthClaims := service.normalizeClaims(provider, externalClaims)

	// Verify claims are copied correctly
	if gauthClaims.RegisteredClaims.Subject != externalClaims.RegisteredClaims.Subject {
		t.Errorf("Expected subject %q but got %q", externalClaims.RegisteredClaims.Subject, gauthClaims.RegisteredClaims.Subject)
	}

	if gauthClaims.Name != externalClaims.Name {
		t.Errorf("Expected name %q but got %q", externalClaims.Name, gauthClaims.Name)
	}

	if gauthClaims.Email != externalClaims.Email {
		t.Errorf("Expected email %q but got %q", externalClaims.Email, gauthClaims.Email)
	}

	if gauthClaims.ACR != externalClaims.ACR {
		t.Errorf("Expected ACR %q but got %q", externalClaims.ACR, gauthClaims.ACR)
	}

	if len(gauthClaims.AMR) != len(externalClaims.AMR) {
		t.Errorf("Expected %d AMR values but got %d", len(externalClaims.AMR), len(gauthClaims.AMR))
	}
}

func TestTokenExchangeService_MapTrustLevel(t *testing.T) {
	service, registry, _ := setupTokenExchangeTest(t)

	tests := []struct {
		name          string
		providerID    string
		claims        *IDTokenClaims
		expectedTrust string
	}{
		{
			name:       "eIDAS high level",
			providerID: "google",
			claims: &IDTokenClaims{
				ACR: "high",
			},
			expectedTrust: "high",
		},
		{
			name:       "eIDAS substantial level",
			providerID: "google",
			claims: &IDTokenClaims{
				ACR: "substantial",
			},
			expectedTrust: "substantial",
		},
		{
			name:       "eIDAS low level",
			providerID: "google",
			claims: &IDTokenClaims{
				ACR: "low",
			},
			expectedTrust: "low",
		},
		{
			name:       "eIDAS URN high",
			providerID: "google",
			claims: &IDTokenClaims{
				ACR: "urn:eidas:loa:high",
			},
			expectedTrust: "high",
		},
		{
			name:       "eIDAS URN substantial",
			providerID: "google",
			claims: &IDTokenClaims{
				ACR: "urn:eidas:loa:substantial",
			},
			expectedTrust: "substantial",
		},
		{
			name:       "eIDAS URN low",
			providerID: "google",
			claims: &IDTokenClaims{
				ACR: "urn:eidas:loa:low",
			},
			expectedTrust: "low",
		},
		{
			name:       "MFA in AMR",
			providerID: "google",
			claims: &IDTokenClaims{
				AMR: []string{"pwd", "mfa"},
			},
			expectedTrust: "high",
		},
		{
			name:       "OTP in AMR",
			providerID: "google",
			claims: &IDTokenClaims{
				AMR: []string{"pwd", "otp"},
			},
			expectedTrust: "high",
		},
		{
			name:       "SMS in AMR",
			providerID: "okta",
			claims: &IDTokenClaims{
				AMR: []string{"pwd", "sms"},
			},
			expectedTrust: "high",
		},
		{
			name:       "Hardware key in AMR",
			providerID: "azure_ad",
			claims: &IDTokenClaims{
				AMR: []string{"pwd", "hwk"},
			},
			expectedTrust: "high",
		},
		{
			name:       "Software key in AMR",
			providerID: "google",
			claims: &IDTokenClaims{
				AMR: []string{"pwd", "swk"},
			},
			expectedTrust: "high",
		},
		{
			name:       "Telephone in AMR",
			providerID: "okta",
			claims: &IDTokenClaims{
				AMR: []string{"pwd", "tel"},
			},
			expectedTrust: "high",
		},
		{
			name:       "Default trust level",
			providerID: "google",
			claims: &IDTokenClaims{
				AMR: []string{"pwd"},
			},
			expectedTrust: "substantial",
		},
		{
			name:       "No ACR/AMR - use provider default",
			providerID: "okta",
			claims:     &IDTokenClaims{},
			expectedTrust: "substantial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.Get(tt.providerID)
			if err != nil {
				t.Fatalf("Failed to get provider: %v", err)
			}

			trustLevel := service.mapTrustLevel(provider, tt.claims)
			if trustLevel != tt.expectedTrust {
				t.Errorf("Expected trust level %q but got %q", tt.expectedTrust, trustLevel)
			}
		})
	}
}

func TestTokenExchangeService_GetSupportedProviders(t *testing.T) {
	service, registry, _ := setupTokenExchangeTest(t)

	providers := service.GetSupportedProviders()

	// Should have 3 enabled providers (google, okta, azure_ad)
	if len(providers) != 3 {
		t.Errorf("Expected 3 supported providers but got %d", len(providers))
	}

	expectedProviders := map[string]bool{
		"google":   false,
		"okta":     false,
		"azure_ad": false,
	}

	for _, providerID := range providers {
		if _, exists := expectedProviders[providerID]; !exists {
			t.Errorf("Unexpected provider ID: %s", providerID)
		}
		expectedProviders[providerID] = true
	}

	for providerID, found := range expectedProviders {
		if !found {
			t.Errorf("Expected provider %s not found in supported providers", providerID)
		}
	}

	// Disable one provider and verify it's not in the list
	if err := registry.Disable("google"); err != nil {
		t.Fatalf("Failed to disable provider: %v", err)
	}

	providers = service.GetSupportedProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 supported providers after disabling one, but got %d", len(providers))
	}
}

func TestTokenExchangeService_GetProviderInfo(t *testing.T) {
	service, _, _ := setupTokenExchangeTest(t)

	tests := []struct {
		name        string
		providerID  string
		expectError bool
	}{
		{
			name:        "Valid provider - Google",
			providerID:  "google",
			expectError: false,
		},
		{
			name:        "Valid provider - Okta",
			providerID:  "okta",
			expectError: false,
		},
		{
			name:        "Valid provider - Azure AD",
			providerID:  "azure_ad",
			expectError: false,
		},
		{
			name:        "Unknown provider",
			providerID:  "unknown",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := service.GetProviderInfo(tt.providerID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if info != nil {
					t.Errorf("Expected nil info but got %+v", info)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if info == nil {
				t.Fatal("Expected provider info but got nil")
			}

			if info.ID != tt.providerID {
				t.Errorf("Expected provider ID %q but got %q", tt.providerID, info.ID)
			}
		})
	}
}

func TestTokenExchangeService_BatchExchangeTokens(t *testing.T) {
	service, _, _ := setupTokenExchangeTest(t)
	ctx := context.Background()

	requests := []ExchangeRequest{
		{
			ProviderID:    "google",
			ExternalToken: "token1",
			Audience:      "audience1",
		},
		{
			ProviderID:    "okta",
			ExternalToken: "token2",
			Audience:      "audience2",
		},
		{
			ProviderID:    "azure_ad",
			ExternalToken: "token3",
			Audience:      "audience3",
		},
	}

	batchReq := BatchExchangeRequest{
		Requests: requests,
	}

	response, err := service.BatchExchangeTokens(ctx, batchReq)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(response.Responses) != len(requests) {
		t.Errorf("Expected %d responses but got %d", len(requests), len(response.Responses))
	}

	if len(response.Errors) != len(requests) {
		t.Errorf("Expected %d errors but got %d", len(requests), len(response.Errors))
	}

	// All should fail because validateExternalToken is not implemented
	for i, err := range response.Errors {
		if err == nil {
			t.Errorf("Expected error for request %d but got none", i)
		}
	}
}

func TestTokenExchangeService_ValidateProviderToken(t *testing.T) {
	service, _, _ := setupTokenExchangeTest(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		providerID  string
		token       string
		audience    string
		expectError bool
	}{
		{
			name:        "Valid provider",
			providerID:  "google",
			token:       "test-token",
			audience:    "test-audience",
			expectError: true, // Will fail because validateExternalToken is not implemented
		},
		{
			name:        "Unknown provider",
			providerID:  "unknown",
			token:       "test-token",
			audience:    "test-audience",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateProviderToken(ctx, tt.providerID, tt.token, tt.audience)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if claims != nil {
					t.Errorf("Expected nil claims but got %+v", claims)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestTokenExchangeService_RevokeExchangedToken(t *testing.T) {
	service, _, _ := setupTokenExchangeTest(t)
	ctx := context.Background()

	// Revocation is not implemented yet
	err := service.RevokeExchangedToken(ctx, "test-token")
	if err == nil {
		t.Error("Expected error for unimplemented revocation but got none")
	}

	expectedMsg := "token revocation not implemented"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q but got %q", expectedMsg, err.Error())
	}
}

func TestTokenExchangeService_RefreshExchangedToken(t *testing.T) {
	service, _, _ := setupTokenExchangeTest(t)
	ctx := context.Background()

	// Refresh is not implemented yet
	response, err := service.RefreshExchangedToken(ctx, "test-refresh-token", "google")
	if err == nil {
		t.Error("Expected error for unimplemented refresh but got none")
	}

	if response != nil {
		t.Errorf("Expected nil response but got %+v", response)
	}

	expectedMsg := "token refresh not implemented"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q but got %q", expectedMsg, err.Error())
	}
}

func TestTokenExchangeService_DisabledProvider(t *testing.T) {
	service, registry, _ := setupTokenExchangeTest(t)
	ctx := context.Background()

	// Disable Google provider
	if err := registry.Disable("google"); err != nil {
		t.Fatalf("Failed to disable provider: %v", err)
	}

	// Try to exchange token with disabled provider
	request := ExchangeRequest{
		ProviderID:    "google",
		ExternalToken: "test-token",
		Audience:      "test-audience",
	}

	response, err := service.ExchangeToken(ctx, request)
	if err == nil {
		t.Error("Expected error for disabled provider but got none")
	}

	if response != nil {
		t.Errorf("Expected nil response but got %+v", response)
	}

	expectedMsg := "provider google is disabled"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q but got %q", expectedMsg, err.Error())
	}
}
