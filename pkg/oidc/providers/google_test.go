package providers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/oidc"
	"github.com/golang-jwt/jwt/v5"
)

func setupGoogleTest(t *testing.T) (*GoogleProvider, *rsa.PrivateKey) {
	t.Helper()

	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create ID token service
	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:     GoogleIssuerURL,
		SigningKey:    privateKey,
		SigningMethod: "RS256",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	// Create discovery cache
	discoveryCache := oidc.NewInMemoryDiscoveryCache()

	// Create Google provider
	provider, err := NewGoogleProvider(GoogleProviderConfig{
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		DiscoveryCache: discoveryCache,
		IDTokenService: idTokenService,
	})
	if err != nil {
		t.Fatalf("Failed to create Google provider: %v", err)
	}

	return provider, privateKey
}

func TestNewGoogleProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     GoogleProviderConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			cfg: GoogleProviderConfig{
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			wantErr: false,
		},
		{
			name: "with hosted domain",
			cfg: GoogleProviderConfig{
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
				HostedDomain:   "example.com",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			cfg: GoogleProviderConfig{
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			wantErr: true,
			errMsg:  "client ID is required",
		},
		{
			name: "missing client secret",
			cfg: GoogleProviderConfig{
				ClientID:       "test-client-id",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
				IDTokenService: &oidc.IDTokenService{},
			},
			wantErr: true,
			errMsg:  "client secret is required",
		},
		{
			name: "missing discovery cache",
			cfg: GoogleProviderConfig{
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				IDTokenService: &oidc.IDTokenService{},
			},
			wantErr: true,
			errMsg:  "discovery cache is required",
		},
		{
			name: "missing ID token service",
			cfg: GoogleProviderConfig{
				ClientID:       "test-client-id",
				ClientSecret:   "test-client-secret",
				DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
			},
			wantErr: true,
			errMsg:  "ID token service is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGoogleProvider(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGoogleProvider() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewGoogleProvider() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewGoogleProvider() unexpected error = %v", err)
				}
				if provider == nil {
					t.Error("NewGoogleProvider() returned nil provider")
				}
			}
		})
	}
}

func TestGoogleProvider_GetConfiguration(t *testing.T) {
	provider, _ := setupGoogleTest(t)

	cfg := provider.GetConfiguration()
	if cfg == nil {
		t.Fatal("GetConfiguration() returned nil")
	}

	if cfg.ID != GoogleProviderID {
		t.Errorf("GetConfiguration() ID = %q, want %q", cfg.ID, GoogleProviderID)
	}

	if cfg.Name != GoogleProviderName {
		t.Errorf("GetConfiguration() Name = %q, want %q", cfg.Name, GoogleProviderName)
	}

	if cfg.IssuerURL != GoogleIssuerURL {
		t.Errorf("GetConfiguration() IssuerURL = %q, want %q", cfg.IssuerURL, GoogleIssuerURL)
	}
}

func TestGoogleProvider_GetDiscoveryDocument(t *testing.T) {
	provider, _ := setupGoogleTest(t)

	// Create mock Google discovery endpoint
	mockDiscovery := oidc.OIDCConfiguration{
		Issuer:                           GoogleIssuerURL,
		AuthorizationEndpoint:            "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:                    "https://oauth2.googleapis.com/token",
		JWKSUri:                          "https://www.googleapis.com/oauth2/v3/certs",
		ResponseTypesSupported:           []string{"code", "token", "id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockDiscovery)
	}))
	defer mockServer.Close()

	// Use mock server URL for testing
	ctx := context.Background()
	err := provider.discoveryCache.Set(GoogleIssuerURL, &mockDiscovery, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set discovery document: %v", err)
	}

	doc, err := provider.GetDiscoveryDocument(ctx)
	if err != nil {
		t.Fatalf("GetDiscoveryDocument() unexpected error = %v", err)
	}

	if doc.Issuer != GoogleIssuerURL {
		t.Errorf("GetDiscoveryDocument() Issuer = %q, want %q", doc.Issuer, GoogleIssuerURL)
	}
}

func TestGoogleProvider_ValidateIDToken(t *testing.T) {
	provider, privateKey := setupGoogleTest(t)
	ctx := context.Background()

	// Create valid ID token
	now := time.Now()
	claims := &oidc.IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    GoogleIssuerURL,
			Subject:   "google-user-123",
			Audience:  jwt.ClaimStrings{"test-client-id"},
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	tests := []struct {
		name     string
		idToken  string
		audience string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid token",
			idToken:  tokenString,
			audience: "test-client-id",
			wantErr:  false,
		},
		{
			name:     "empty token",
			idToken:  "",
			audience: "test-client-id",
			wantErr:  true,
			errMsg:   "ID token is required",
		},
		{
			name:     "empty audience",
			idToken:  tokenString,
			audience: "",
			wantErr:  true,
			errMsg:   "audience is required",
		},
		{
			name:     "wrong audience",
			idToken:  tokenString,
			audience: "wrong-client-id",
			wantErr:  true,
			errMsg:   "token validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedClaims, err := provider.ValidateIDToken(ctx, tt.idToken, tt.audience)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateIDToken() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateIDToken() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateIDToken() unexpected error = %v", err)
				}
				if validatedClaims == nil {
					t.Error("ValidateIDToken() returned nil claims")
				}
			}
		})
	}
}

func TestGoogleProvider_MapClaims(t *testing.T) {
	provider, _ := setupGoogleTest(t)

	googleClaims := map[string]interface{}{
		"sub":            "google-user-123",
		"email":          "test@example.com",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
		"picture":        "https://example.com/photo.jpg",
		"locale":         "en",
		"hd":             "example.com",
		"custom_claim":   "custom_value", // Should be preserved
	}

	mapped := provider.MapClaims(googleClaims)

	// Check mapped claims
	expectedMappings := map[string]string{
		"user_id":       "google-user-123",
		"email":         "test@example.com",
		"full_name":     "Test User",
		"given_name":    "Test",
		"family_name":   "User",
		"avatar_url":    "https://example.com/photo.jpg",
		"locale":        "en",
		"hosted_domain": "example.com",
	}

	for gauthClaim, expectedValue := range expectedMappings {
		if value, exists := mapped[gauthClaim]; !exists {
			t.Errorf("MapClaims() missing mapped claim %q", gauthClaim)
		} else if value != expectedValue {
			t.Errorf("MapClaims() %q = %v, want %v", gauthClaim, value, expectedValue)
		}
	}

	// Check email_verified is preserved (both mapped and unmapped)
	if emailVerified, ok := mapped["email_verified"].(bool); !ok || !emailVerified {
		t.Error("MapClaims() email_verified should be preserved")
	}

	// Check custom claim is preserved
	if customValue, ok := mapped["custom_claim"].(string); !ok || customValue != "custom_value" {
		t.Error("MapClaims() custom claims should be preserved")
	}
}

func TestGoogleProvider_GetTrustLevel(t *testing.T) {
	provider, _ := setupGoogleTest(t)

	tests := []struct {
		name   string
		claims *oidc.IDTokenClaims
		want   string
	}{
		{
			name: "default trust level (no ACR/AMR)",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
			},
			want: GoogleDefaultTrust,
		},
		{
			name: "multi-factor ACR",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
				ACR: "http://schemas.openid.net/pape/policies/2007/06/multi-factor",
			},
			want: "high",
		},
		{
			name: "phishing-resistant ACR",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
				ACR: "http://schemas.openid.net/pape/policies/2007/06/phishing-resistant",
			},
			want: "high",
		},
		{
			name: "MFA via AMR",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
				AMR: []string{"pwd", "mfa"},
			},
			want: "high",
		},
		{
			name: "OTP via AMR",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
				AMR: []string{"otp"},
			},
			want: "high",
		},
		{
			name: "password only AMR",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
				AMR: []string{"pwd"},
			},
			want: GoogleDefaultTrust,
		},
		{
			name: "unknown ACR",
			claims: &oidc.IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user-123",
				},
				ACR: "unknown-acr",
			},
			want: GoogleDefaultTrust,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.GetTrustLevel(tt.claims)
			if got != tt.want {
				t.Errorf("GetTrustLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoogleProvider_HostedDomain(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:     GoogleIssuerURL,
		SigningKey:    privateKey,
		SigningMethod: "RS256",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	provider, err := NewGoogleProvider(GoogleProviderConfig{
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		DiscoveryCache: oidc.NewInMemoryDiscoveryCache(),
		IDTokenService: idTokenService,
		HostedDomain:   "example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create Google provider: %v", err)
	}

	if !provider.SupportsHostedDomain() {
		t.Error("SupportsHostedDomain() = false, want true")
	}

	hd := provider.GetHostedDomain()
	if hd != "example.com" {
		t.Errorf("GetHostedDomain() = %q, want %q", hd, "example.com")
	}
}

func TestGoogleProvider_GetAuthorizationURL(t *testing.T) {
	provider, _ := setupGoogleTest(t)

	// Set up discovery document
	mockDiscovery := oidc.OIDCConfiguration{
		Issuer:                GoogleIssuerURL,
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
	}

	ctx := context.Background()
	err := provider.discoveryCache.Set(GoogleIssuerURL, &mockDiscovery, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set discovery document: %v", err)
	}

	authURL, err := provider.GetAuthorizationURL(ctx, "https://example.com/callback", "test-state", "test-nonce")
	if err != nil {
		t.Fatalf("GetAuthorizationURL() unexpected error = %v", err)
	}

	// Check URL contains required parameters
	requiredParams := []string{
		"client_id=test-client-id",
		"redirect_uri=https://example.com/callback",
		"response_type=code",
		"scope=openid profile email",
		"state=test-state",
		"nonce=test-nonce",
	}

	for _, param := range requiredParams {
		if !contains(authURL, param) {
			t.Errorf("GetAuthorizationURL() missing parameter %q", param)
		}
	}
}

func TestGoogleProvider_GetterMethods(t *testing.T) {
	provider, _ := setupGoogleTest(t)

	if id := provider.GetProviderID(); id != GoogleProviderID {
		t.Errorf("GetProviderID() = %q, want %q", id, GoogleProviderID)
	}

	if name := provider.GetProviderName(); name != GoogleProviderName {
		t.Errorf("GetProviderName() = %q, want %q", name, GoogleProviderName)
	}

	if !provider.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
