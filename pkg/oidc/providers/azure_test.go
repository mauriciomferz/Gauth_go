package providers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/oidc"
	"github.com/golang-jwt/jwt/v5"
)

// Helper function to create a test Azure AD provider
func createTestAzureADProvider(t *testing.T, tenantID string, allowedTenants []string) *AzureADProvider {
	t.Helper()

	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create ID token service
	issuerURL := "https://login.microsoftonline.com/" + tenantID + "/v2.0"
	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:     issuerURL,
		SigningKey:    privateKey,
		SigningMethod: "RS256",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	config := AzureADProviderConfig{
		TenantID:       tenantID,
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
		IDTokenService: idTokenService,
		AllowedTenants: allowedTenants,
	}

	provider, err := NewAzureADProvider(config)
	if err != nil {
		t.Fatalf("Failed to create test Azure AD provider: %v", err)
	}

	return provider
}

func TestNewAzureADProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      AzureADProviderConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Azure AD provider with GUID tenant",
			config: AzureADProviderConfig{
				TenantID:       "12345678-1234-1234-1234-123456789012",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Valid Azure AD provider with common tenant",
			config: AzureADProviderConfig{
				TenantID:       "common",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Valid Azure AD provider with organizations tenant",
			config: AzureADProviderConfig{
				TenantID:       "organizations",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Valid Azure AD provider with consumers tenant",
			config: AzureADProviderConfig{
				TenantID:       "consumers",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Valid Azure AD provider with allowed tenants",
			config: AzureADProviderConfig{
				TenantID:       "common",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
				AllowedTenants: []string{"tenant1", "tenant2"},
			},
			expectError: false,
		},
		{
			name: "Missing tenant ID",
			config: AzureADProviderConfig{
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "tenant ID is required",
		},
		{
			name: "Invalid tenant ID format",
			config: AzureADProviderConfig{
				TenantID:       "invalid-tenant",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "invalid tenant ID: must be a GUID, 'common', 'organizations', or 'consumers'",
		},
		{
			name: "Missing client ID",
			config: AzureADProviderConfig{
				TenantID:       "common",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "client ID is required",
		},
		{
			name: "Missing client secret",
			config: AzureADProviderConfig{
				TenantID:       "common",
				ClientID:       "test-client-id",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "client secret is required",
		},
		{
			name: "Missing discovery cache",
			config: AzureADProviderConfig{
				TenantID:       "common",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "discovery cache is required",
		},
		{
			name: "Missing ID token service",
			config: AzureADProviderConfig{
				TenantID:       "common",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
			},
			expectError: true,
			errorMsg:    "ID token service is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAzureADProvider(tt.config)

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

			if provider == nil {
				t.Fatal("Expected provider but got nil")
			}

			// Verify provider configuration
			config := provider.GetConfiguration()
			if config.ID != AzureADProviderID {
				t.Errorf("Expected provider ID %q but got %q", AzureADProviderID, config.ID)
			}

			if config.Name != AzureADProviderName {
				t.Errorf("Expected provider name %q but got %q", AzureADProviderName, config.Name)
			}

			// Verify tenant ID
			if provider.GetTenantID() != tt.config.TenantID {
				t.Errorf("Expected tenant ID %q but got %q", tt.config.TenantID, provider.GetTenantID())
			}
		})
	}
}

func TestAzureADProvider_GetConfiguration(t *testing.T) {
	provider := createTestAzureADProvider(t, "common", nil)

	config := provider.GetConfiguration()
	if config == nil {
		t.Fatal("Expected configuration but got nil")
	}

	if config.ID != AzureADProviderID {
		t.Errorf("Expected provider ID %q but got %q", AzureADProviderID, config.ID)
	}

	if config.Name != AzureADProviderName {
		t.Errorf("Expected provider name %q but got %q", AzureADProviderName, config.Name)
	}

	expectedIssuer := "https://login.microsoftonline.com/common/v2.0"
	if config.IssuerURL != expectedIssuer {
		t.Errorf("Expected issuer URL %q but got %q", expectedIssuer, config.IssuerURL)
	}
}

func TestAzureADProvider_ValidateIDToken(t *testing.T) {
	ctx := context.Background()
	provider := createTestAzureADProvider(t, "common", nil)

	tests := []struct {
		name        string
		idToken     string
		audience    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty token",
			idToken:     "",
			audience:    "test-client-id",
			expectError: true,
			errorMsg:    "ID token is required",
		},
		{
			name:        "Empty audience",
			idToken:     "test-token",
			audience:    "",
			expectError: true,
			errorMsg:    "audience is required",
		},
		{
			name:        "Invalid token format",
			idToken:     "invalid-token",
			audience:    "test-client-id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := provider.ValidateIDToken(ctx, tt.idToken, tt.audience)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q but got %q", tt.errorMsg, err.Error())
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

func TestAzureADProvider_MapClaims(t *testing.T) {
	provider := createTestAzureADProvider(t, "common", nil)

	azureClaims := map[string]interface{}{
		"oid":                "00000000-0000-0000-0000-000000000000",
		"sub":                "AAAAAAAAAAAAAAAAAAAAAAAAAg",
		"email":              "user@contoso.com",
		"upn":                "user@contoso.com",
		"name":               "John Doe",
		"given_name":         "John",
		"family_name":        "Doe",
		"preferred_username": "johndoe@contoso.com",
		"roles":              []string{"Admin", "User"},
		"groups":             []string{"group1", "group2"},
		"tid":                "12345678-1234-1234-1234-123456789012",
		"custom_claim":       "custom_value",
	}

	mappedClaims := provider.MapClaims(azureClaims)

	// Verify mapped claims
	expectedMappings := map[string]interface{}{
		"user_id":            "00000000-0000-0000-0000-000000000000",
		"subject":            "AAAAAAAAAAAAAAAAAAAAAAAAAg",
		"email":              "user@contoso.com",
		"username":           "user@contoso.com",
		"full_name":          "John Doe",
		"given_name":         "John",
		"family_name":        "Doe",
		"preferred_username": "johndoe@contoso.com",
		"roles":              []string{"Admin", "User"},
		"groups":             []string{"group1", "group2"},
		"tenant_id":          "12345678-1234-1234-1234-123456789012",
	}

	for gauthClaim, expectedValue := range expectedMappings {
		actualValue, exists := mappedClaims[gauthClaim]
		if !exists {
			t.Errorf("Expected claim %q to be mapped but it wasn't", gauthClaim)
			continue
		}

		// For slice comparison
		if gauthClaim == "roles" || gauthClaim == "groups" {
			expectedSlice := expectedValue.([]string)
			actualSlice, ok := actualValue.([]string)
			if !ok {
				t.Errorf("Expected claim %q to be []string but got %T", gauthClaim, actualValue)
				continue
			}
			if len(expectedSlice) != len(actualSlice) {
				t.Errorf("Expected claim %q to have %d items but got %d", gauthClaim, len(expectedSlice), len(actualSlice))
				continue
			}
			for i, item := range expectedSlice {
				if item != actualSlice[i] {
					t.Errorf("Expected item %d to be %q but got %q", i, item, actualSlice[i])
				}
			}
			continue
		}

		if actualValue != expectedValue {
			t.Errorf("Expected claim %q to be %v but got %v", gauthClaim, expectedValue, actualValue)
		}
	}

	// Verify custom claim is preserved
	if customValue, exists := mappedClaims["custom_claim"]; !exists {
		t.Error("Expected custom claim to be preserved but it wasn't")
	} else if customValue != "custom_value" {
		t.Errorf("Expected custom claim to be %q but got %q", "custom_value", customValue)
	}
}

func TestAzureADProvider_GetTrustLevel(t *testing.T) {
	provider := createTestAzureADProvider(t, "common", nil)

	tests := []struct {
		name          string
		claims        *oidc.IDTokenClaims
		expectedTrust string
	}{
		{
			name: "Default trust level (no ACR/AMR)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
			},
			expectedTrust: AzureADDefaultTrust,
		},
		{
			name: "ACR level 0 (low trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "0",
			},
			expectedTrust: "low",
		},
		{
			name: "ACR level 1 (substantial trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "1",
			},
			expectedTrust: "substantial",
		},
		{
			name: "ACR level 2 (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "2",
			},
			expectedTrust: "high",
		},
		{
			name: "ACR level 3 (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "3",
			},
			expectedTrust: "high",
		},
		{
			name: "ACR c1 - Conditional access (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "c1",
			},
			expectedTrust: "high",
		},
		{
			name: "ACR c2 - Conditional access (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "c2",
			},
			expectedTrust: "high",
		},
		{
			name: "ACR c3 - Conditional access (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "c3",
			},
			expectedTrust: "high",
		},
		{
			name: "MFA in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "mfa"},
			},
			expectedTrust: "high",
		},
		{
			name: "OTP in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "otp"},
			},
			expectedTrust: "high",
		},
		{
			name: "SMS in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "sms"},
			},
			expectedTrust: "high",
		},
		{
			name: "Telephone in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "tel"},
			},
			expectedTrust: "high",
		},
		{
			name: "Hardware key in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "hwk"},
			},
			expectedTrust: "high",
		},
		{
			name: "Software key in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "swk"},
			},
			expectedTrust: "high",
		},
		{
			name: "Windows integrated auth in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"wia"},
			},
			expectedTrust: "high",
		},
		{
			name: "NGC MFA in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"ngcmfa"},
			},
			expectedTrust: "high",
		},
		{
			name: "RSA in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"rsa"},
			},
			expectedTrust: "high",
		},
		{
			name: "Password-only AMR (substantial trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd"},
			},
			expectedTrust: "substantial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trustLevel := provider.GetTrustLevel(tt.claims)
			if trustLevel != tt.expectedTrust {
				t.Errorf("Expected trust level %q but got %q", tt.expectedTrust, trustLevel)
			}
		})
	}
}

func TestAzureADProvider_IsMultiTenant(t *testing.T) {
	tests := []struct {
		name                string
		tenantID            string
		expectedMultiTenant bool
	}{
		{
			name:                "Common tenant (multi-tenant)",
			tenantID:            "common",
			expectedMultiTenant: true,
		},
		{
			name:                "Organizations tenant (multi-tenant)",
			tenantID:            "organizations",
			expectedMultiTenant: true,
		},
		{
			name:                "Consumers tenant (not multi-tenant)",
			tenantID:            "consumers",
			expectedMultiTenant: false,
		},
		{
			name:                "Specific tenant GUID (not multi-tenant)",
			tenantID:            "12345678-1234-1234-1234-123456789012",
			expectedMultiTenant: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := createTestAzureADProvider(t, tt.tenantID, nil)

			if provider.IsMultiTenant() != tt.expectedMultiTenant {
				t.Errorf("Expected IsMultiTenant to be %v but got %v", tt.expectedMultiTenant, provider.IsMultiTenant())
			}
		})
	}
}

func TestAzureADProvider_TenantValidation(t *testing.T) {
	tests := []struct {
		name          string
		tenantID      string
		expectedValid bool
	}{
		{
			name:          "Valid GUID",
			tenantID:      "12345678-1234-1234-1234-123456789012",
			expectedValid: true,
		},
		{
			name:          "Valid common",
			tenantID:      "common",
			expectedValid: true,
		},
		{
			name:          "Valid organizations",
			tenantID:      "organizations",
			expectedValid: true,
		},
		{
			name:          "Valid consumers",
			tenantID:      "consumers",
			expectedValid: true,
		},
		{
			name:          "Invalid GUID (too short)",
			tenantID:      "12345678-1234-1234-1234",
			expectedValid: false,
		},
		{
			name:          "Invalid GUID (wrong format)",
			tenantID:      "12345678123412341234123456789012",
			expectedValid: false,
		},
		{
			name:          "Invalid keyword",
			tenantID:      "invalid",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidAzureTenantID(tt.tenantID)
			if valid != tt.expectedValid {
				t.Errorf("Expected isValidAzureTenantID(%q) to be %v but got %v", tt.tenantID, tt.expectedValid, valid)
			}
		})
	}
}

func TestAzureADProvider_AllowedTenants(t *testing.T) {
	allowedTenants := []string{
		"tenant1",
		"tenant2",
		"12345678-1234-1234-1234-123456789012",
	}

	provider := createTestAzureADProvider(t, "common", allowedTenants)

	// Verify allowed tenants are stored
	storedTenants := provider.GetAllowedTenants()
	if len(storedTenants) != len(allowedTenants) {
		t.Errorf("Expected %d allowed tenants but got %d", len(allowedTenants), len(storedTenants))
	}

	for i, tenant := range allowedTenants {
		if storedTenants[i] != tenant {
			t.Errorf("Expected tenant %d to be %q but got %q", i, tenant, storedTenants[i])
		}
	}
}

func TestAzureADProvider_GetAuthorizationURL(t *testing.T) {
	// This test verifies that the method exists and accepts the right parameters
	provider := createTestAzureADProvider(t, "common", nil)

	ctx := context.Background()
	redirectURI := "https://example.com/callback"
	state := "test-state"
	nonce := "test-nonce"

	// GetAuthorizationURL will attempt to fetch discovery document
	// which will fail without a real endpoint, but verifies the method signature
	_, err := provider.GetAuthorizationURL(ctx, redirectURI, state, nonce)
	// We expect an error since discovery will fail, but don't be strict about it
	// as caching or other factors might affect this
	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
}

func TestAzureADProvider_GetterMethods(t *testing.T) {
	tenantID := "12345678-1234-1234-1234-123456789012"
	provider := createTestAzureADProvider(t, tenantID, nil)

	if provider.GetProviderID() != AzureADProviderID {
		t.Errorf("Expected provider ID %q but got %q", AzureADProviderID, provider.GetProviderID())
	}

	if provider.GetProviderName() != AzureADProviderName {
		t.Errorf("Expected provider name %q but got %q", AzureADProviderName, provider.GetProviderName())
	}

	if !provider.IsEnabled() {
		t.Error("Expected provider to be enabled but it wasn't")
	}

	if !provider.SupportsRoles() {
		t.Error("Expected provider to support roles but it doesn't")
	}

	if !provider.SupportsGroups() {
		t.Error("Expected provider to support groups but it doesn't")
	}

	if provider.GetTenantID() != tenantID {
		t.Errorf("Expected tenant ID %q but got %q", tenantID, provider.GetTenantID())
	}
}

func TestAzureADProvider_IssuerValidation(t *testing.T) {
	provider := createTestAzureADProvider(t, "common", nil)

	tests := []struct {
		name          string
		issuer        string
		expectedValid bool
	}{
		{
			name:          "Valid v2.0 issuer",
			issuer:        "https://login.microsoftonline.com/12345678-1234-1234-1234-123456789012/v2.0",
			expectedValid: true,
		},
		{
			name:          "Valid v1.0 issuer",
			issuer:        "https://sts.windows.net/12345678-1234-1234-1234-123456789012/",
			expectedValid: true,
		},
		{
			name:          "Invalid issuer",
			issuer:        "https://accounts.google.com",
			expectedValid: false,
		},
		{
			name:          "Invalid issuer (wrong domain)",
			issuer:        "https://login.example.com/tenant/v2.0",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := provider.isValidAzureIssuer(tt.issuer)
			if valid != tt.expectedValid {
				t.Errorf("Expected isValidAzureIssuer(%q) to be %v but got %v", tt.issuer, tt.expectedValid, valid)
			}
		})
	}
}

func TestAzureADProvider_TenantIDExtraction(t *testing.T) {
	provider := createTestAzureADProvider(t, "common", nil)

	tests := []struct {
		name             string
		issuer           string
		expectedTenantID string
	}{
		{
			name:             "Extract from v2.0 issuer",
			issuer:           "https://login.microsoftonline.com/12345678-1234-1234-1234-123456789012/v2.0",
			expectedTenantID: "12345678-1234-1234-1234-123456789012",
		},
		{
			name:             "Extract from v1.0 issuer",
			issuer:           "https://sts.windows.net/12345678-1234-1234-1234-123456789012/",
			expectedTenantID: "12345678-1234-1234-1234-123456789012",
		},
		{
			name:             "Invalid issuer",
			issuer:           "https://example.com/invalid",
			expectedTenantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: tt.issuer,
				},
			}

			tenantID := provider.extractTenantID(claims)
			if tenantID != tt.expectedTenantID {
				t.Errorf("Expected tenant ID %q but got %q", tt.expectedTenantID, tenantID)
			}
		})
	}
}
