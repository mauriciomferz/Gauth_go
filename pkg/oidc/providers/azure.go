// Package providers implements provider-specific OIDC integrations.
package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/oidc"
)

// Azure AD OIDC provider constants
const (
	AzureADProviderID     = "azure_ad"
	AzureADProviderName   = "Azure AD"
	AzureADDefaultTrust   = "substantial"
	AzureADCommonEndpoint = "https://login.microsoftonline.com/common/v2.0"
)

// AzureADClaimMappings defines how Azure AD OIDC claims map to GAuth claims.
// Azure AD provides enterprise-grade claims with role and group information.
var AzureADClaimMappings = map[string]string{
	"oid":                "user_id",            // Object ID (unique identifier)
	"sub":                "subject",            // Subject (app-specific identifier)
	"email":              "email",              // User's email address
	"upn":                "username",           // User Principal Name
	"name":               "full_name",          // Full name
	"given_name":         "given_name",         // Given/first name
	"family_name":        "family_name",        // Family/last name
	"preferred_username": "preferred_username", // Preferred username
	"roles":              "roles",              // Application roles
	"groups":             "groups",             // Group memberships
	"tid":                "tenant_id",          // Tenant ID
}

// AzureADProvider implements Azure AD-specific OIDC integration.
type AzureADProvider struct {
	config         *oidc.ProviderConfig
	discoveryCache oidc.DiscoveryCache
	idTokenService *oidc.IDTokenService
	tenantID       string // Azure AD tenant ID (or "common", "organizations", "consumers")
}

// AzureADProviderConfig holds configuration for Azure AD provider.
type AzureADProviderConfig struct {
	TenantID       string // Tenant ID (GUID), or "common", "organizations", "consumers"
	ClientID       string
	ClientSecret   string
	DiscoveryCache oidc.DiscoveryCache
	IDTokenService *oidc.IDTokenService
	AllowedTenants []string // Optional: whitelist of allowed tenant IDs
}

// NewAzureADProvider creates a new Azure AD OIDC provider.
func NewAzureADProvider(cfg AzureADProviderConfig) (*AzureADProvider, error) {
	if cfg.TenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
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

	// Validate tenant ID format
	tenantID := strings.TrimSpace(cfg.TenantID)
	if !isValidAzureTenantID(tenantID) {
		return nil, fmt.Errorf("invalid tenant ID: must be a GUID, 'common', 'organizations', or 'consumers'")
	}

	// Construct issuer URL based on tenant
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)

	// Create provider configuration
	providerConfig := oidc.ProviderConfig{
		ID:                AzureADProviderID,
		Name:              AzureADProviderName,
		IssuerURL:         issuerURL,
		ClientID:          cfg.ClientID,
		ClientSecret:      cfg.ClientSecret,
		Scopes:            []string{"openid", "profile", "email", "User.Read"},
		ClaimMappings:     AzureADClaimMappings,
		DefaultTrustLevel: AzureADDefaultTrust,
		Enabled:           true,
		Metadata:          make(map[string]interface{}),
	}

	// Add tenant ID to metadata
	providerConfig.Metadata["tenant_id"] = tenantID

	// Add allowed tenants to metadata if provided
	if len(cfg.AllowedTenants) > 0 {
		providerConfig.Metadata["allowed_tenants"] = cfg.AllowedTenants
	}

	// Validate configuration
	if err := providerConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	return &AzureADProvider{
		config:         &providerConfig,
		discoveryCache: cfg.DiscoveryCache,
		idTokenService: cfg.IDTokenService,
		tenantID:       tenantID,
	}, nil
}

// isValidAzureTenantID checks if the tenant ID is valid.
func isValidAzureTenantID(tenantID string) bool {
	// Check for special tenant values
	if tenantID == "common" || tenantID == "organizations" || tenantID == "consumers" {
		return true
	}

	// Check for GUID format (simple validation)
	// GUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(tenantID) != 36 {
		return false
	}

	// Check for hyphens in correct positions
	if tenantID[8] != '-' || tenantID[13] != '-' || tenantID[18] != '-' || tenantID[23] != '-' {
		return false
	}

	return true
}

// GetConfiguration returns the provider configuration.
func (p *AzureADProvider) GetConfiguration() *oidc.ProviderConfig {
	return p.config
}

// GetDiscoveryDocument fetches Azure AD's OIDC discovery document.
func (p *AzureADProvider) GetDiscoveryDocument(ctx context.Context) (*oidc.OIDCConfiguration, error) {
	return p.discoveryCache.Get(ctx, p.config.IssuerURL)
}

// ValidateIDToken validates an Azure AD ID token.
func (p *AzureADProvider) ValidateIDToken(ctx context.Context, idToken string, audience string) (*oidc.IDTokenClaims, error) {
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

	// Verify issuer matches Azure AD
	if !p.isValidAzureIssuer(claims.Issuer) {
		return nil, fmt.Errorf("invalid issuer: expected Azure AD issuer, got %s", claims.Issuer)
	}

	// If allowed tenants are configured, verify tenant ID
	if allowedTenants, ok := p.config.Metadata["allowed_tenants"].([]string); ok && len(allowedTenants) > 0 {
		// Extract tenant ID from token (tid claim or from issuer)
		tenantID := p.extractTenantID(claims)
		if !p.isTenantAllowed(tenantID, allowedTenants) {
			return nil, fmt.Errorf("tenant %s is not allowed", tenantID)
		}
	}

	return claims, nil
}

// isValidAzureIssuer checks if the issuer is a valid Azure AD issuer.
func (p *AzureADProvider) isValidAzureIssuer(issuer string) bool {
	// Azure AD v2.0 issuer format: https://login.microsoftonline.com/{tenant}/v2.0
	// Also accept: https://sts.windows.net/{tenant}/ (v1.0)
	return strings.HasPrefix(issuer, "https://login.microsoftonline.com/") ||
		strings.HasPrefix(issuer, "https://sts.windows.net/")
}

// extractTenantID extracts the tenant ID from claims or issuer.
func (p *AzureADProvider) extractTenantID(claims *oidc.IDTokenClaims) string {
	// First, try to get tenant ID from issuer URL
	// Format: https://login.microsoftonline.com/{tenant}/v2.0
	issuer := claims.Issuer
	if strings.HasPrefix(issuer, "https://login.microsoftonline.com/") {
		parts := strings.Split(issuer, "/")
		if len(parts) >= 4 {
			return parts[3] // tenant ID is the 4th part
		}
	}

	// Fallback: try to get from issuer (v1.0 format)
	if strings.HasPrefix(issuer, "https://sts.windows.net/") {
		parts := strings.Split(issuer, "/")
		if len(parts) >= 4 {
			return strings.TrimSuffix(parts[3], "/")
		}
	}

	return ""
}

// isTenantAllowed checks if a tenant ID is in the allowed list.
func (p *AzureADProvider) isTenantAllowed(tenantID string, allowedTenants []string) bool {
	for _, allowed := range allowedTenants {
		if tenantID == allowed {
			return true
		}
	}
	return false
}

// MapClaims maps Azure AD OIDC claims to GAuth format.
func (p *AzureADProvider) MapClaims(claims map[string]interface{}) map[string]interface{} {
	mapped := make(map[string]interface{})

	for azureClaim, gauthClaim := range AzureADClaimMappings {
		if value, exists := claims[azureClaim]; exists {
			mapped[gauthClaim] = value
		}
	}

	// Preserve unmapped claims
	for key, value := range claims {
		if _, isMapped := AzureADClaimMappings[key]; !isMapped {
			mapped[key] = value
		}
	}

	return mapped
}

// GetTrustLevel determines the trust level for an Azure AD ID token.
// Azure AD provides authentication context through AMR claims.
func (p *AzureADProvider) GetTrustLevel(claims *oidc.IDTokenClaims) string {
	// Check if ACR is present in claims
	if claims.ACR != "" {
		// Map Azure AD ACR to GAuth trust level
		switch claims.ACR {
		case "0":
			return "low" // No specific authentication context
		case "1":
			return "substantial" // Password authentication
		case "2", "3":
			return "high" // Multi-factor or higher
		case "c1", "c2", "c3":
			return "high" // Conditional access policies (MFA enforced)
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
	return AzureADDefaultTrust
}

// hasMFA checks if the token has MFA authentication.
func (p *AzureADProvider) hasMFA(claims *oidc.IDTokenClaims) bool {
	for _, method := range claims.AMR {
		switch method {
		case "mfa", "otp", "sms", "tel", "hwk", "swk", "wia", "ngcmfa", "rsa":
			return true
		}
	}
	return false
}

// GetTenantID returns the tenant ID.
func (p *AzureADProvider) GetTenantID() string {
	return p.tenantID
}

// IsMultiTenant checks if this provider is configured for multi-tenant use.
func (p *AzureADProvider) IsMultiTenant() bool {
	return p.tenantID == "common" || p.tenantID == "organizations"
}

// GetAuthorizationURL constructs the Azure AD OAuth2 authorization URL.
func (p *AzureADProvider) GetAuthorizationURL(ctx context.Context, redirectURI string, state string, nonce string) (string, error) {
	discovery, err := p.GetDiscoveryDocument(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get discovery document: %w", err)
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&nonce=%s",
		discovery.AuthorizationEndpoint,
		p.config.ClientID,
		redirectURI,
		"openid profile email User.Read",
		state,
		nonce,
	)

	return authURL, nil
}

// GetProviderID returns the provider identifier.
func (p *AzureADProvider) GetProviderID() string {
	return AzureADProviderID
}

// GetProviderName returns the provider name.
func (p *AzureADProvider) GetProviderName() string {
	return AzureADProviderName
}

// IsEnabled checks if the provider is enabled.
func (p *AzureADProvider) IsEnabled() bool {
	return p.config.Enabled
}

// SupportsRoles checks if role claims are supported.
func (p *AzureADProvider) SupportsRoles() bool {
	for _, scope := range p.config.Scopes {
		if scope == "User.Read" {
			return true
		}
	}
	return false
}

// SupportsGroups checks if group claims are supported.
func (p *AzureADProvider) SupportsGroups() bool {
	// Azure AD supports groups, but requires additional configuration
	// in the app registration (optional claims or group membership claims)
	return true
}

// GetAllowedTenants returns the list of allowed tenant IDs.
func (p *AzureADProvider) GetAllowedTenants() []string {
	if allowedTenants, ok := p.config.Metadata["allowed_tenants"].([]string); ok {
		return allowedTenants
	}
	return nil
}
