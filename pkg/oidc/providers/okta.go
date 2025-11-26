// Package providers implements provider-specific OIDC integrations.
package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mauriciomferz/Gauth_go/pkg/oidc"
)

// Okta OIDC provider constants
const (
	OktaProviderID   = "okta"
	OktaProviderName = "Okta"
	OktaDefaultTrust = "substantial"
)

// OktaClaimMappings defines how Okta OIDC claims map to GAuth claims.
// Okta provides enterprise-grade claims with group information.
var OktaClaimMappings = map[string]string{
	"sub":                "user_id",        // Unique user identifier
	"email":              "email",          // User's email address
	"email_verified":     "email_verified", // Email verification status
	"name":               "full_name",      // Full name
	"given_name":         "given_name",     // Given/first name
	"family_name":        "family_name",    // Family/last name
	"preferred_username": "username",       // Username
	"locale":             "locale",         // User's locale
	"zoneinfo":           "timezone",       // User's timezone
	"groups":             "roles",          // User's groups/roles
}

// OktaProvider implements Okta-specific OIDC integration.
type OktaProvider struct {
	config         *oidc.ProviderConfig
	discoveryCache oidc.DiscoveryCache
	idTokenService *oidc.IDTokenService
	domain         string // Okta domain (e.g., "dev-12345.okta.com")
}

// OktaProviderConfig holds configuration for Okta provider.
type OktaProviderConfig struct {
	Domain         string // Okta domain (e.g., "dev-12345.okta.com" or "mycompany.okta.com")
	ClientID       string
	ClientSecret   string
	DiscoveryCache oidc.DiscoveryCache
	IDTokenService *oidc.IDTokenService
	RequireMFA     bool // If true, only accept tokens with MFA
}

// NewOktaProvider creates a new Okta OIDC provider.
func NewOktaProvider(cfg OktaProviderConfig) (*OktaProvider, error) {
	if cfg.Domain == "" {
		return nil, fmt.Errorf("Okta domain is required")
	}

	if cfg.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	if cfg.DiscoveryCache == nil {
		return nil, fmt.Errorf("discovery cache is required")
	}

	if cfg.IDTokenService == nil {
		return nil, fmt.Errorf("ID token service is required")
	}

	// Normalize domain (remove https:// if present)
	domain := strings.TrimPrefix(cfg.Domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	// Construct issuer URL
	issuerURL := "https://" + domain

	// Create provider configuration
	providerConfig := oidc.ProviderConfig{
		ID:                OktaProviderID,
		Name:              OktaProviderName,
		IssuerURL:         issuerURL,
		ClientID:          cfg.ClientID,
		ClientSecret:      cfg.ClientSecret,
		Scopes:            []string{"openid", "profile", "email", "groups"},
		ClaimMappings:     OktaClaimMappings,
		DefaultTrustLevel: OktaDefaultTrust,
		Enabled:           true,
		Metadata:          make(map[string]interface{}),
	}

	// Add MFA requirement to metadata
	if cfg.RequireMFA {
		providerConfig.Metadata["require_mfa"] = true
	}

	// Add domain to metadata
	providerConfig.Metadata["domain"] = domain

	// Validate configuration
	if err := providerConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	return &OktaProvider{
		config:         &providerConfig,
		discoveryCache: cfg.DiscoveryCache,
		idTokenService: cfg.IDTokenService,
		domain:         domain,
	}, nil
}

// GetConfiguration returns the provider configuration.
func (p *OktaProvider) GetConfiguration() *oidc.ProviderConfig {
	return p.config
}

// GetDiscoveryDocument fetches Okta's OIDC discovery document.
func (p *OktaProvider) GetDiscoveryDocument(ctx context.Context) (*oidc.OIDCConfiguration, error) {
	return p.discoveryCache.Get(ctx, p.config.IssuerURL)
}

// ValidateIDToken validates an Okta ID token.
func (p *OktaProvider) ValidateIDToken(ctx context.Context, idToken string, audience string) (*oidc.IDTokenClaims, error) {
	if idToken == "" {
		return nil, fmt.Errorf("ID token is required")
	}

	if audience == "" {
		return nil, fmt.Errorf("audience is required")
	}

	// Validate token using ID token service
	claims, err := p.idTokenService.ValidateIDToken(ctx, idToken, audience)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Verify issuer matches Okta
	if claims.Issuer != p.config.IssuerURL {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", p.config.IssuerURL, claims.Issuer)
	}

	// If MFA is required, check for MFA in AMR
	if requireMFA, ok := p.config.Metadata["require_mfa"].(bool); ok && requireMFA {
		if !p.hasMFA(claims) {
			return nil, fmt.Errorf("MFA is required but not present in token")
		}
	}

	return claims, nil
}

// hasMFA checks if the token has MFA authentication.
func (p *OktaProvider) hasMFA(claims *oidc.IDTokenClaims) bool {
	for _, method := range claims.AMR {
		switch method {
		case "mfa", "otp", "sms", "hwk", "swk", "tel", "kba":
			return true
		}
	}
	return false
}

// MapClaims maps Okta OIDC claims to GAuth format.
func (p *OktaProvider) MapClaims(claims map[string]interface{}) map[string]interface{} {
	mapped := make(map[string]interface{})

	for oktaClaim, gauthClaim := range OktaClaimMappings {
		if value, exists := claims[oktaClaim]; exists {
			mapped[gauthClaim] = value
		}
	}

	// Preserve unmapped claims
	for key, value := range claims {
		if _, isMapped := OktaClaimMappings[key]; !isMapped {
			mapped[key] = value
		}
	}

	return mapped
}

// GetTrustLevel determines the trust level for an Okta ID token.
// Okta provides detailed authentication context through AMR claims.
func (p *OktaProvider) GetTrustLevel(claims *oidc.IDTokenClaims) string {
	// Check if ACR is present in claims
	if claims.ACR != "" {
		// Map Okta ACR to GAuth trust level
		switch claims.ACR {
		case "urn:okta:loa:2fa:any", "urn:okta:loa:2fa:any:ifpossible":
			return "high" // Two-factor authentication
		case "urn:okta:loa:1fa:pwd":
			return "substantial" // Password only
		case "urn:okta:loa:1fa:any":
			return "low" // Single factor, any method
		}
	}

	// Check AMR for MFA indicators
	if p.hasMFA(claims) {
		return "high"
	}

	// Check for password authentication
	for _, method := range claims.AMR {
		if method == "pwd" {
			return "substantial" // Password authentication
		}
	}

	// Default to substantial for authenticated users
	return OktaDefaultTrust
}

// RequiresMFA checks if MFA is required for this provider.
func (p *OktaProvider) RequiresMFA() bool {
	requireMFA, ok := p.config.Metadata["require_mfa"].(bool)
	return ok && requireMFA
}

// GetDomain returns the Okta domain.
func (p *OktaProvider) GetDomain() string {
	return p.domain
}

// GetAuthorizationURL constructs the Okta OAuth2 authorization URL.
func (p *OktaProvider) GetAuthorizationURL(ctx context.Context, redirectURI string, state string, nonce string) (string, error) {
	discovery, err := p.GetDiscoveryDocument(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get discovery document: %w", err)
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&nonce=%s",
		discovery.AuthorizationEndpoint,
		p.config.ClientID,
		redirectURI,
		"openid profile email groups",
		state,
		nonce,
	)

	return authURL, nil
}

// GetProviderID returns the provider identifier.
func (p *OktaProvider) GetProviderID() string {
	return OktaProviderID
}

// GetProviderName returns the provider name.
func (p *OktaProvider) GetProviderName() string {
	return OktaProviderName
}

// IsEnabled checks if the provider is enabled.
func (p *OktaProvider) IsEnabled() bool {
	return p.config.Enabled
}

// SupportsGroups checks if group claims are supported.
func (p *OktaProvider) SupportsGroups() bool {
	for _, scope := range p.config.Scopes {
		if scope == "groups" {
			return true
		}
	}
	return false
}
