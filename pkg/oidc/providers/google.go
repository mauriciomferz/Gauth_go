// Package providers implements provider-specific OIDC integrations.
package providers

import (
	"context"
	"fmt"

	"github.com/mauriciomferz/AgentAuth/pkg/oidc"
)

// Google OIDC provider constants
const (
	GoogleIssuerURL    = "https://accounts.google.com"
	GoogleDiscoveryURL = "https://accounts.google.com/.well-known/openid-configuration"
	GoogleDefaultTrust = "substantial"
	GoogleProviderID   = "google"
	GoogleProviderName = "Google"
)

// GoogleClaimMappings defines how Google OIDC claims map to AgentAuth claims.
// Google follows standard OIDC claims closely.
var GoogleClaimMappings = map[string]string{
	"sub":            "user_id",        // Unique user identifier
	"email":          "email",          // User's email address
	"email_verified": "email_verified", // Email verification status
	"name":           "full_name",      // Full name
	"given_name":     "given_name",     // Given/first name
	"family_name":    "family_name",    // Family/last name
	"picture":        "avatar_url",     // Profile picture URL
	"locale":         "locale",         // User's locale
	"hd":             "hosted_domain",  // Google Workspace hosted domain
}

// GoogleProvider implements Google-specific OIDC integration.
type GoogleProvider struct {
	config         *oidc.ProviderConfig
	discoveryCache oidc.DiscoveryCache
	idTokenService *oidc.IDTokenService
}

// GoogleProviderConfig holds configuration for Google provider.
type GoogleProviderConfig struct {
	ClientID       string
	ClientSecret   string
	DiscoveryCache oidc.DiscoveryCache
	IDTokenService *oidc.IDTokenService
	HostedDomain   string // Optional: restrict to specific Google Workspace domain
}

// NewGoogleProvider creates a new Google OIDC provider.
func NewGoogleProvider(cfg GoogleProviderConfig) (*GoogleProvider, error) {
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

	// Create provider configuration
	providerConfig := oidc.ProviderConfig{
		ID:                GoogleProviderID,
		Name:              GoogleProviderName,
		IssuerURL:         GoogleIssuerURL,
		ClientID:          cfg.ClientID,
		ClientSecret:      cfg.ClientSecret,
		Scopes:            []string{"openid", "profile", "email"},
		ClaimMappings:     GoogleClaimMappings,
		DefaultTrustLevel: GoogleDefaultTrust,
		Enabled:           true,
		Metadata:          make(map[string]interface{}),
	}

	// Add hosted domain if specified
	if cfg.HostedDomain != "" {
		providerConfig.Metadata["hosted_domain"] = cfg.HostedDomain
	}

	// Validate configuration
	if err := providerConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	return &GoogleProvider{
		config:         &providerConfig,
		discoveryCache: cfg.DiscoveryCache,
		idTokenService: cfg.IDTokenService,
	}, nil
}

// GetConfiguration returns the provider configuration.
func (p *GoogleProvider) GetConfiguration() *oidc.ProviderConfig {
	return p.config
}

// GetDiscoveryDocument fetches Google's OIDC discovery document.
func (p *GoogleProvider) GetDiscoveryDocument(ctx context.Context) (*oidc.OIDCConfiguration, error) {
	return p.discoveryCache.Get(ctx, GoogleIssuerURL)
}

// ValidateIDToken validates a Google ID token.
func (p *GoogleProvider) ValidateIDToken(ctx context.Context, idToken string, audience string) (*oidc.IDTokenClaims, error) {
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

	// Verify issuer matches Google
	if claims.Issuer != GoogleIssuerURL {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", GoogleIssuerURL, claims.Issuer)
	}

	// Note: hosted domain (hd) verification would require custom JWT parsing
	// as it's not part of standard OIDC claims. For production, implement custom
	// claim extraction from the raw JWT token.

	return claims, nil
}

// MapClaims maps Google OIDC claims to AgentAuth format.
func (p *GoogleProvider) MapClaims(claims map[string]interface{}) map[string]interface{} {
	mapped := make(map[string]interface{})

	for googleClaim, agentauthClaim := range GoogleClaimMappings {
		if value, exists := claims[googleClaim]; exists {
			mapped[agentauthClaim] = value
		}
	}

	// Preserve unmapped claims
	for key, value := range claims {
		if _, isMapped := GoogleClaimMappings[key]; !isMapped {
			mapped[key] = value
		}
	}

	return mapped
}

// GetTrustLevel determines the trust level for a Google ID token.
// Google uses standard authentication, which maps to "substantial" by default.
// Google doesn't provide detailed ACR (Authentication Context Class Reference) in standard flows.
func (p *GoogleProvider) GetTrustLevel(claims *oidc.IDTokenClaims) string {
	// Check if ACR is present in claims
	if claims.ACR != "" {
		// Map Google ACR to AgentAuth trust level
		switch claims.ACR {
		case "http://schemas.openid.net/pape/policies/2007/06/multi-factor":
			return trustLevelHigh // Multi-factor authentication
		case "http://schemas.openid.net/pape/policies/2007/06/phishing-resistant":
			return trustLevelHigh // Phishing-resistant authentication
		default:
			// Unknown ACR, use default
			return GoogleDefaultTrust
		}
	}

	// Check for multi-factor authentication via AMR (Authentication Methods References)
	if len(claims.AMR) > 0 {
		for _, method := range claims.AMR {
			switch method {
			case "mfa", "otp", "sms", "hwk":
				return trustLevelHigh // Multi-factor methods
			}
		}
	}

	// Default to substantial trust level
	return GoogleDefaultTrust
}

// SupportsHostedDomain checks if a hosted domain is configured.
func (p *GoogleProvider) SupportsHostedDomain() bool {
	_, ok := p.config.Metadata["hosted_domain"].(string)
	return ok
}

// GetHostedDomain returns the configured hosted domain, if any.
func (p *GoogleProvider) GetHostedDomain() string {
	if hd, ok := p.config.Metadata["hosted_domain"].(string); ok {
		return hd
	}
	return ""
}

// GetAuthorizationURL constructs the Google OAuth2 authorization URL.
func (p *GoogleProvider) GetAuthorizationURL(ctx context.Context, redirectURI string, state string, nonce string) (string, error) {
	discovery, err := p.GetDiscoveryDocument(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get discovery document: %w", err)
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&nonce=%s",
		discovery.AuthorizationEndpoint,
		p.config.ClientID,
		redirectURI,
		"openid profile email",
		state,
		nonce,
	)

	// Add hosted domain hint if configured
	if hd := p.GetHostedDomain(); hd != "" {
		authURL += fmt.Sprintf("&hd=%s", hd)
	}

	return authURL, nil
}

// GetProviderID returns the provider identifier.
func (p *GoogleProvider) GetProviderID() string {
	return GoogleProviderID
}

// GetProviderName returns the provider name.
func (p *GoogleProvider) GetProviderName() string {
	return GoogleProviderName
}

// IsEnabled checks if the provider is enabled.
func (p *GoogleProvider) IsEnabled() bool {
	return p.config.Enabled
}
