// Package oidc - OIDC Discovery Service
// Implements OpenID Connect Discovery 1.0
package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

const (
	signingAlgorithmRS256 = "RS256"
)

// DiscoveryService provides OpenID Connect Discovery functionality
// Spec: OpenID Connect Discovery 1.0
// Endpoint: /.well-known/openid-configuration
type DiscoveryService struct {
	config     *OIDCConfiguration
	configLock sync.RWMutex
	issuerURL  string
}

// NewDiscoveryService creates a new OIDC Discovery Service
func NewDiscoveryService(issuerURL string) *DiscoveryService {
	service := &DiscoveryService{
		issuerURL: issuerURL,
	}

	// Initialize default configuration
	service.config = service.buildDefaultConfiguration()

	return service
}

// buildDefaultConfiguration creates default OIDC configuration
func (s *DiscoveryService) buildDefaultConfiguration() *OIDCConfiguration {
	return &OIDCConfiguration{
		// Core Discovery Fields (REQUIRED)
		Issuer:                issuerURL(s.issuerURL),
		AuthorizationEndpoint: issuerURL(s.issuerURL + "/oauth/authorize"),
		TokenEndpoint:         issuerURL(s.issuerURL + "/oauth/token"),
		JWKSUri:               issuerURL(s.issuerURL + "/.well-known/jwks.json"),

		// Optional but Recommended
		UserInfoEndpoint:     issuerURL(s.issuerURL + "/oauth/userinfo"),
		RegistrationEndpoint: issuerURL(s.issuerURL + "/oauth/register"),

		// Supported Values
		ResponseTypesSupported: []string{
			"code",           // Authorization Code Flow
			"id_token",       // Implicit Flow (ID Token only)
			"token id_token", // Implicit Flow (Access Token + ID Token)
			"code id_token",  // Hybrid Flow
		},
		SubjectTypesSupported: []string{
			"public",   // Subject identifier: same for all clients
			"pairwise", // Subject identifier: different per client
		},
		IDTokenSigningAlgValuesSupported: []string{
			signingAlgorithmRS256, // RSA with SHA-256 (REQUIRED by spec)
			"RS384",
			"RS512",
			"ES256", // ECDSA with SHA-256
			"ES384",
			"ES512",
		},
		ScopesSupported: []string{
			"openid",         // REQUIRED for OIDC
			"profile",        // Name, family_name, given_name, etc.
			"email",          // Email, email_verified
			"phone",          // Phone number, phone_number_verified
			"address",        // Physical address
			"offline_access", // Refresh tokens
			// AgentAuth-specific scopes
			"gauth:owner",        // Owner authorization scope
			"gauth:client",       // Client authorization scope
			"gauth:resource",     // Resource access scope
			"gauth:legal_entity", // Legal entity information
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic", // HTTP Basic Auth
			"client_secret_post",  // POST body
			"private_key_jwt",     // JWT with private key
			"none",                // Public clients
		},
		ClaimsSupported: []string{
			// Standard OIDC Claims
			"sub", "name", "given_name", "family_name", "middle_name",
			"nickname", "preferred_username", "profile", "picture",
			"website", "email", "email_verified", "gender", "birthdate",
			"zoneinfo", "locale", "phone_number", "phone_number_verified",
			"updated_at",
			// Authentication Context
			"acr", "amr",
			// AgentAuth Extensions
			"entity_type", "entity_id", "legal_entity_name", "jurisdiction",
			"tsp_name", "tsp_id",
		},

		// ACR (Authentication Context Class Reference) Values
		// Maps to AgentAuth TrustLevel
		ACRValuesSupported: []string{
			"0",                            // No specific authentication context
			"1",                            // Basic authentication
			"2",                            // Multi-factor authentication
			"substantial",                  // eIDAS Substantial
			"high",                         // eIDAS High
			"loa-4",                        // NIST LOA-4
			"urn:mace:incommon:iap:bronze", // InCommon Bronze
			"urn:mace:incommon:iap:silver", // InCommon Silver
		},

		ServiceDocumentation: issuerURL(s.issuerURL + "/docs/oidc"),
	}
}

// GetConfiguration returns the current OIDC configuration
func (s *DiscoveryService) GetConfiguration() *OIDCConfiguration {
	s.configLock.RLock()
	defer s.configLock.RUnlock()

	return s.config
}

// UpdateConfiguration updates the OIDC configuration
// Use this to customize endpoints, supported values, etc.
func (s *DiscoveryService) UpdateConfiguration(config *OIDCConfiguration) {
	s.configLock.Lock()
	defer s.configLock.Unlock()

	s.config = config
}

// ServeHTTP implements http.Handler for /.well-known/openid-configuration
// Spec: OpenID Connect Discovery 1.0 Section 4
func (s *DiscoveryService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := s.GetConfiguration()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// ValidateConfiguration validates OIDC configuration against spec requirements
func (s *DiscoveryService) ValidateConfiguration() error {
	config := s.GetConfiguration()

	// Check REQUIRED fields (per OpenID Connect Discovery 1.0)
	if config.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}
	if config.AuthorizationEndpoint == "" {
		return fmt.Errorf("authorization_endpoint is required")
	}
	if config.TokenEndpoint == "" {
		return fmt.Errorf("token_endpoint is required")
	}
	if config.JWKSUri == "" {
		return fmt.Errorf("jwks_uri is required")
	}

	// Check response_types_supported (REQUIRED)
	if len(config.ResponseTypesSupported) == 0 {
		return fmt.Errorf("response_types_supported is required")
	}

	// Check subject_types_supported (REQUIRED)
	if len(config.SubjectTypesSupported) == 0 {
		return fmt.Errorf("subject_types_supported is required")
	}

	// Check id_token_signing_alg_values_supported (REQUIRED)
	if len(config.IDTokenSigningAlgValuesSupported) == 0 {
		return fmt.Errorf("id_token_signing_alg_values_supported is required")
	}

	// RS256 MUST be supported (per spec)
	hasRS256 := false
	for _, alg := range config.IDTokenSigningAlgValuesSupported {
		if alg == signingAlgorithmRS256 {
			hasRS256 = true
			break
		}
	}
	if !hasRS256 {
		return fmt.Errorf("RS256 must be supported for ID token signing")
	}

	// Validate scopes_supported includes "openid"
	if len(config.ScopesSupported) > 0 {
		hasOpenID := false
		for _, scope := range config.ScopesSupported {
			if scope == "openid" {
				hasOpenID = true
				break
			}
		}
		if !hasOpenID {
			return fmt.Errorf("openid scope must be supported")
		}
	}

	return nil
}

// Helper function to ensure consistent URL formatting
func issuerURL(url string) string {
	// Remove trailing slash if present
	if len(url) > 0 && url[len(url)-1] == '/' {
		return url[:len(url)-1]
	}
	return url
}

// GetWellKnownEndpoint returns the well-known configuration endpoint path
func (s *DiscoveryService) GetWellKnownEndpoint() string {
	return "/.well-known/openid-configuration"
}

// SupportsACR checks if a specific ACR value is supported
func (s *DiscoveryService) SupportsACR(acr string) bool {
	config := s.GetConfiguration()

	for _, supportedACR := range config.ACRValuesSupported {
		if supportedACR == acr {
			return true
		}
	}

	return false
}

// SupportsScope checks if a specific scope is supported
func (s *DiscoveryService) SupportsScope(scope string) bool {
	config := s.GetConfiguration()

	for _, supportedScope := range config.ScopesSupported {
		if supportedScope == scope {
			return true
		}
	}

	return false
}

// GetSupportedSigningAlgorithms returns supported ID token signing algorithms
func (s *DiscoveryService) GetSupportedSigningAlgorithms() []string {
	config := s.GetConfiguration()
	return config.IDTokenSigningAlgValuesSupported
}
