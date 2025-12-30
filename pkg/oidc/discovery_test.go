package oidc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDiscoveryService(t *testing.T) {
	issuerURL := "https://agentauth.example.com"
	service := NewDiscoveryService(issuerURL)

	if service == nil {
		t.Fatal("NewDiscoveryService returned nil")
	}

	if service.issuerURL != issuerURL {
		t.Errorf("Expected issuer URL %s, got %s", issuerURL, service.issuerURL)
	}

	config := service.GetConfiguration()
	if config == nil {
		t.Fatal("Configuration is nil")
	}

	if config.Issuer != issuerURL {
		t.Errorf("Expected issuer %s, got %s", issuerURL, config.Issuer)
	}
}

func TestDiscoveryService_GetConfiguration(t *testing.T) {
	service := NewDiscoveryService("https://agentauth.example.com")
	config := service.GetConfiguration()

	// Verify required fields
	if config.Issuer == "" {
		t.Error("Issuer is empty")
	}
	if config.AuthorizationEndpoint == "" {
		t.Error("AuthorizationEndpoint is empty")
	}
	if config.TokenEndpoint == "" {
		t.Error("TokenEndpoint is empty")
	}
	if config.JWKSUri == "" {
		t.Error("JWKSUri is empty")
	}

	// Verify supported values
	if len(config.ResponseTypesSupported) == 0 {
		t.Error("ResponseTypesSupported is empty")
	}
	if len(config.SubjectTypesSupported) == 0 {
		t.Error("SubjectTypesSupported is empty")
	}
	if len(config.IDTokenSigningAlgValuesSupported) == 0 {
		t.Error("IDTokenSigningAlgValuesSupported is empty")
	}

	// Verify RS256 is supported (REQUIRED by spec)
	hasRS256 := false
	for _, alg := range config.IDTokenSigningAlgValuesSupported {
		if alg == "RS256" {
			hasRS256 = true
			break
		}
	}
	if !hasRS256 {
		t.Error("RS256 must be supported")
	}

	// Verify openid scope is supported
	hasOpenID := false
	for _, scope := range config.ScopesSupported {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		t.Error("openid scope must be supported")
	}
}

func TestDiscoveryService_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*OIDCConfiguration)
		wantErr bool
	}{
		{
			name:    "Valid configuration",
			setup:   func(c *OIDCConfiguration) {},
			wantErr: false,
		},
		{
			name: "Missing issuer",
			setup: func(c *OIDCConfiguration) {
				c.Issuer = ""
			},
			wantErr: true,
		},
		{
			name: "Missing authorization endpoint",
			setup: func(c *OIDCConfiguration) {
				c.AuthorizationEndpoint = ""
			},
			wantErr: true,
		},
		{
			name: "Missing token endpoint",
			setup: func(c *OIDCConfiguration) {
				c.TokenEndpoint = ""
			},
			wantErr: true,
		},
		{
			name: "Missing JWKS URI",
			setup: func(c *OIDCConfiguration) {
				c.JWKSUri = ""
			},
			wantErr: true,
		},
		{
			name: "Missing RS256 support",
			setup: func(c *OIDCConfiguration) {
				c.IDTokenSigningAlgValuesSupported = []string{"HS256"}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewDiscoveryService("https://agentauth.example.com")
			config := service.GetConfiguration()
			tt.setup(config)
			service.UpdateConfiguration(config)

			err := service.ValidateConfiguration()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscoveryService_ServeHTTP(t *testing.T) {
	service := NewDiscoveryService("https://agentauth.example.com")

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET request succeeds",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST request fails",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/.well-known/openid-configuration", nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.method == http.MethodGet {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}
			}
		})
	}
}

func TestDiscoveryService_SupportsACR(t *testing.T) {
	service := NewDiscoveryService("https://agentauth.example.com")

	tests := []struct {
		name string
		acr  string
		want bool
	}{
		{
			name: "Supports substantial",
			acr:  "substantial",
			want: true,
		},
		{
			name: "Supports high",
			acr:  "high",
			want: true,
		},
		{
			name: "Supports loa-4",
			acr:  "loa-4",
			want: true,
		},
		{
			name: "Does not support unknown ACR",
			acr:  "unknown_acr",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.SupportsACR(tt.acr); got != tt.want {
				t.Errorf("SupportsACR(%s) = %v, want %v", tt.acr, got, tt.want)
			}
		})
	}
}

func TestDiscoveryService_SupportsScope(t *testing.T) {
	service := NewDiscoveryService("https://agentauth.example.com")

	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{
			name:  "Supports openid",
			scope: "openid",
			want:  true,
		},
		{
			name:  "Supports profile",
			scope: "profile",
			want:  true,
		},
		{
			name:  "Supports agentauth:owner",
			scope: "agentauth:owner",
			want:  true,
		},
		{
			name:  "Does not support unknown scope",
			scope: "unknown_scope",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.SupportsScope(tt.scope); got != tt.want {
				t.Errorf("SupportsScope(%s) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

func TestDiscoveryService_AgentAuthExtensions(t *testing.T) {
	service := NewDiscoveryService("https://agentauth.example.com")
	config := service.GetConfiguration()

	// Verify AgentAuth-specific scopes
	agentauthScopes := []string{"agentauth:owner", "agentauth:client", "agentauth:resource", "agentauth:legal_entity"}
	for _, scope := range agentauthScopes {
		found := false
		for _, supported := range config.ScopesSupported {
			if supported == scope {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AgentAuth scope %s not found in supported scopes", scope)
		}
	}

	// Verify AgentAuth-specific claims
	AGENTAUTHClaims := []string{"entity_type", "entity_id", "legal_entity_name", "jurisdiction"}
	for _, claim := range AGENTAUTHClaims {
		found := false
		for _, supported := range config.ClaimsSupported {
			if supported == claim {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AgentAuth claim %s not found in supported claims", claim)
		}
	}
}

func TestIssuerURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "URL with trailing slash",
			input: "https://agentauth.example.com/",
			want:  "https://agentauth.example.com",
		},
		{
			name:  "URL without trailing slash",
			input: "https://agentauth.example.com",
			want:  "https://agentauth.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := issuerURL(tt.input); got != tt.want {
				t.Errorf("issuerURL(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}
