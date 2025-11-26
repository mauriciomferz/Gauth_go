package providers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/oidc"
	"github.com/golang-jwt/jwt/v5"
)

// Helper function to create a test Okta provider
func createTestOktaProvider(t *testing.T, domain string, requireMFA bool) *OktaProvider {
	t.Helper()

	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create ID token service
	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:     "https://" + domain,
		SigningKey:    privateKey,
		SigningMethod: "RS256",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	config := OktaProviderConfig{
		Domain:         domain,
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
		IDTokenService: idTokenService,
		RequireMFA:     requireMFA,
	}

	provider, err := NewOktaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create test Okta provider: %v", err)
	}

	return provider
}

func TestNewOktaProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      OktaProviderConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Okta provider",
			config: OktaProviderConfig{
				Domain:         "dev-12345.okta.com",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Valid Okta provider with custom domain",
			config: OktaProviderConfig{
				Domain:         "login.mycompany.com",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: false,
		},
		{
			name: "Valid Okta provider with MFA required",
			config: OktaProviderConfig{
				Domain:         "dev-12345.okta.com",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
				RequireMFA:     true,
			},
			expectError: false,
		},
		{
			name: "Missing domain",
			config: OktaProviderConfig{
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "Okta domain is required",
		},
		{
			name: "Missing client ID",
			config: OktaProviderConfig{
				Domain:         "dev-12345.okta.com",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "client ID is required",
		},
		{
			name: "Missing client secret",
			config: OktaProviderConfig{
				Domain:         "dev-12345.okta.com",
				ClientID:       "test-client-id",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "client secret is required",
		},
		{
			name: "Missing discovery cache",
			config: OktaProviderConfig{
				Domain:         "dev-12345.okta.com",
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				IDTokenService: &oidc.IDTokenService{},
			},
			expectError: true,
			errorMsg:    "discovery cache is required",
		},
		{
			name: "Missing ID token service",
			config: OktaProviderConfig{
				Domain:         "dev-12345.okta.com",
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
			provider, err := NewOktaProvider(tt.config)

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
			if config.ID != OktaProviderID {
				t.Errorf("Expected provider ID %q but got %q", OktaProviderID, config.ID)
			}

			if config.Name != OktaProviderName {
				t.Errorf("Expected provider name %q but got %q", OktaProviderName, config.Name)
			}

			// Verify domain normalization
			expectedDomain := tt.config.Domain
			if provider.GetDomain() != expectedDomain {
				t.Errorf("Expected domain %q but got %q", expectedDomain, provider.GetDomain())
			}
		})
	}
}

func TestOktaProvider_GetConfiguration(t *testing.T) {
	provider := createTestOktaProvider(t, "dev-12345.okta.com", false)

	config := provider.GetConfiguration()
	if config == nil {
		t.Fatal("Expected configuration but got nil")
	}

	if config.ID != OktaProviderID {
		t.Errorf("Expected provider ID %q but got %q", OktaProviderID, config.ID)
	}

	if config.Name != OktaProviderName {
		t.Errorf("Expected provider name %q but got %q", OktaProviderName, config.Name)
	}

	if config.IssuerURL != "https://dev-12345.okta.com" {
		t.Errorf("Expected issuer URL %q but got %q", "https://dev-12345.okta.com", config.IssuerURL)
	}
}

func TestOktaProvider_ValidateIDToken(t *testing.T) {
	ctx := context.Background()
	provider := createTestOktaProvider(t, "dev-12345.okta.com", false)

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

func TestOktaProvider_MapClaims(t *testing.T) {
	provider := createTestOktaProvider(t, "dev-12345.okta.com", false)

	oktaClaims := map[string]interface{}{
		"sub":                "00u1a2b3c4d5e6f7g8h9",
		"email":              "user@example.com",
		"email_verified":     true,
		"name":               "John Doe",
		"given_name":         "John",
		"family_name":        "Doe",
		"preferred_username": "johndoe",
		"locale":             "en-US",
		"zoneinfo":           "America/New_York",
		"groups":             []string{"Admins", "Developers"},
		"custom_claim":       "custom_value",
	}

	mappedClaims := provider.MapClaims(oktaClaims)

	// Verify mapped claims
	expectedMappings := map[string]interface{}{
		"user_id":        "00u1a2b3c4d5e6f7g8h9",
		"email":          "user@example.com",
		"email_verified": true,
		"full_name":      "John Doe",
		"given_name":     "John",
		"family_name":    "Doe",
		"username":       "johndoe",
		"locale":         "en-US",
		"timezone":       "America/New_York",
		"roles":          []string{"Admins", "Developers"},
	}

	for gauthClaim, expectedValue := range expectedMappings {
		actualValue, exists := mappedClaims[gauthClaim]
		if !exists {
			t.Errorf("Expected claim %q to be mapped but it wasn't", gauthClaim)
			continue
		}

		// For slice comparison, convert to string
		if gauthClaim == "roles" {
			expectedGroups := expectedValue.([]string)
			actualGroups, ok := actualValue.([]string)
			if !ok {
				t.Errorf("Expected claim %q to be []string but got %T", gauthClaim, actualValue)
				continue
			}
			if len(expectedGroups) != len(actualGroups) {
				t.Errorf("Expected claim %q to have %d groups but got %d", gauthClaim, len(expectedGroups), len(actualGroups))
				continue
			}
			for i, group := range expectedGroups {
				if group != actualGroups[i] {
					t.Errorf("Expected group %d to be %q but got %q", i, group, actualGroups[i])
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

func TestOktaProvider_GetTrustLevel(t *testing.T) {
	provider := createTestOktaProvider(t, "dev-12345.okta.com", false)

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
			expectedTrust: OktaDefaultTrust,
		},
		{
			name: "Two-factor ACR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "urn:okta:loa:2fa:any",
			},
			expectedTrust: "high",
		},
		{
			name: "Two-factor ACR if possible (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "urn:okta:loa:2fa:any:ifpossible",
			},
			expectedTrust: "high",
		},
		{
			name: "Password-only ACR (substantial trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "urn:okta:loa:1fa:pwd",
			},
			expectedTrust: "substantial",
		},
		{
			name: "Single-factor ACR (low trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				ACR: "urn:okta:loa:1fa:any",
			},
			expectedTrust: "low",
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
			name: "Knowledge-based authentication in AMR (high trust)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "test-user",
				},
				AMR: []string{"pwd", "kba"},
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

func TestOktaProvider_RequiresMFA(t *testing.T) {
	tests := []struct {
		name        string
		requireMFA  bool
		expectedMFA bool
	}{
		{
			name:        "MFA required",
			requireMFA:  true,
			expectedMFA: true,
		},
		{
			name:        "MFA not required",
			requireMFA:  false,
			expectedMFA: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := createTestOktaProvider(t, "dev-12345.okta.com", tt.requireMFA)

			if provider.RequiresMFA() != tt.expectedMFA {
				t.Errorf("Expected RequiresMFA to be %v but got %v", tt.expectedMFA, provider.RequiresMFA())
			}
		})
	}
}

func TestOktaProvider_CustomDomain(t *testing.T) {
	tests := []struct {
		name           string
		domain         string
		expectedDomain string
		expectedIssuer string
	}{
		{
			name:           "Standard Okta domain",
			domain:         "dev-12345.okta.com",
			expectedDomain: "dev-12345.okta.com",
			expectedIssuer: "https://dev-12345.okta.com",
		},
		{
			name:           "Okta preview domain",
			domain:         "dev-12345.oktapreview.com",
			expectedDomain: "dev-12345.oktapreview.com",
			expectedIssuer: "https://dev-12345.oktapreview.com",
		},
		{
			name:           "Custom domain",
			domain:         "login.mycompany.com",
			expectedDomain: "login.mycompany.com",
			expectedIssuer: "https://login.mycompany.com",
		},
		{
			name:           "Domain with https prefix",
			domain:         "https://dev-12345.okta.com",
			expectedDomain: "dev-12345.okta.com",
			expectedIssuer: "https://dev-12345.okta.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := createTestOktaProvider(t, tt.domain, false)

			if provider.GetDomain() != tt.expectedDomain {
				t.Errorf("Expected domain %q but got %q", tt.expectedDomain, provider.GetDomain())
			}

			config := provider.GetConfiguration()
			if config.IssuerURL != tt.expectedIssuer {
				t.Errorf("Expected issuer URL %q but got %q", tt.expectedIssuer, config.IssuerURL)
			}
		})
	}
}

func TestOktaProvider_GetAuthorizationURL(t *testing.T) {
	// This test would require mocking the discovery document
	// For now, we'll test that the method exists and accepts the right parameters
	provider := createTestOktaProvider(t, "dev-12345.okta.com", false)

	ctx := context.Background()
	redirectURI := "https://example.com/callback"
	state := "test-state"
	nonce := "test-nonce"

	// This will fail because we don't have a real discovery document
	// but it verifies the method signature
	_, err := provider.GetAuthorizationURL(ctx, redirectURI, state, nonce)
	if err == nil {
		t.Error("Expected error due to missing discovery document, but got none")
	}
}

func TestOktaProvider_GetterMethods(t *testing.T) {
	provider := createTestOktaProvider(t, "dev-12345.okta.com", false)

	if provider.GetProviderID() != OktaProviderID {
		t.Errorf("Expected provider ID %q but got %q", OktaProviderID, provider.GetProviderID())
	}

	if provider.GetProviderName() != OktaProviderName {
		t.Errorf("Expected provider name %q but got %q", OktaProviderName, provider.GetProviderName())
	}

	if !provider.IsEnabled() {
		t.Error("Expected provider to be enabled but it wasn't")
	}

	if !provider.SupportsGroups() {
		t.Error("Expected provider to support groups but it doesn't")
	}
}
