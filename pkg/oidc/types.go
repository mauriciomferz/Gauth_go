// Package oidc implements OpenID Connect integration for AgentAuth
// AAP001 Building Block: OpenID Connect as identity verification mechanism
package oidc

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCConfiguration represents OpenID Connect Discovery metadata
// Spec: OpenID Connect Discovery 1.0
type OIDCConfiguration struct {
	// Core Discovery Fields (REQUIRED per spec)
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSUri               string `json:"jwks_uri"`

	// Optional but Recommended
	UserInfoEndpoint     string `json:"userinfo_endpoint,omitempty"`
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`

	// Supported Values
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`

	// AgentAuth Extensions - ACR/LOA Support
	ACRValuesSupported []string `json:"acr_values_supported"`

	// Metadata
	ServiceDocumentation string `json:"service_documentation,omitempty"`
}

// IDTokenClaims represents OIDC ID Token claims
// Spec: OpenID Connect Core 1.0 Section 2
type IDTokenClaims struct {
	jwt.RegisteredClaims

	// Standard OIDC Claims (Section 5.1)
	Name                string `json:"name,omitempty"`
	GivenName           string `json:"given_name,omitempty"`
	FamilyName          string `json:"family_name,omitempty"`
	MiddleName          string `json:"middle_name,omitempty"`
	Nickname            string `json:"nickname,omitempty"`
	PreferredUsername   string `json:"preferred_username,omitempty"`
	Profile             string `json:"profile,omitempty"`
	Picture             string `json:"picture,omitempty"`
	Website             string `json:"website,omitempty"`
	Email               string `json:"email,omitempty"`
	EmailVerified       bool   `json:"email_verified,omitempty"`
	Gender              string `json:"gender,omitempty"`
	Birthdate           string `json:"birthdate,omitempty"`
	Zoneinfo            string `json:"zoneinfo,omitempty"`
	Locale              string `json:"locale,omitempty"`
	PhoneNumber         string `json:"phone_number,omitempty"`
	PhoneNumberVerified bool   `json:"phone_number_verified,omitempty"`
	UpdatedAt           int64  `json:"updated_at,omitempty"`

	// Authentication Context Class Reference (ACR)
	// Maps to AgentAuth TrustLevel: "substantial", "high", "loa-4", etc.
	ACR string `json:"acr,omitempty"`

	// Authentication Methods References (AMR)
	// Examples: "pwd", "mfa", "otp", "hwk", "bio"
	AMR []string `json:"amr,omitempty"`

	// Authorized Party (azp) - Client ID that requested the ID Token
	AuthorizedParty string `json:"azp,omitempty"`

	// Nonce - For replay attack prevention
	Nonce string `json:"nonce,omitempty"`

	// AgentAuth Extensions - Legal Entity Support
	EntityType      string `json:"entity_type,omitempty"`       // "natural_person", "legal_entity"
	EntityID        string `json:"entity_id,omitempty"`         // Tax ID, registration number
	LegalEntityName string `json:"legal_entity_name,omitempty"` // Company name
	Jurisdiction    string `json:"jurisdiction,omitempty"`      // "DE", "US", "EU", etc.

	// AgentAuth Extensions - Trust Service Provider
	TSPName string `json:"tsp_name,omitempty"` // Trust Service Provider name
	TSPID   string `json:"tsp_id,omitempty"`   // TSP identifier
}

// AuthorizationCodeFlowRequest represents OAuth 2.0/OIDC authorization request
type AuthorizationCodeFlowRequest struct {
	ClientID     string   `json:"client_id"`
	RedirectURI  string   `json:"redirect_uri"`
	ResponseType string   `json:"response_type"` // "code"
	Scope        []string `json:"scope"`         // ["openid", "profile", "email"]
	State        string   `json:"state"`
	Nonce        string   `json:"nonce"`                // For ID token replay prevention
	ACRValues    []string `json:"acr_values,omitempty"` // Requested authentication context
}

// TokenResponse represents OIDC token endpoint response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"` // JWT containing IDTokenClaims
	Scope        string `json:"scope,omitempty"`
}

// ExternalProviderConfig configures external OIDC provider (Google, Okta, Azure AD)
type ExternalProviderConfig struct {
	ProviderID   string            `json:"provider_id"`   // "google", "okta", "azure_ad"
	ProviderName string            `json:"provider_name"` // Display name
	IssuerURL    string            `json:"issuer_url"`    // https://accounts.google.com
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	RedirectURI  string            `json:"redirect_uri"`
	Scopes       []string          `json:"scopes"`      // ["openid", "profile", "email"]
	ACRMapping   map[string]string `json:"acr_mapping"` // Provider ACR → AgentAuth TrustLevel
}

// TrustLevelMapping maps OIDC ACR values to AgentAuth trust levels
type TrustLevelMapping struct {
	ACR                 string `json:"acr"`                   // OIDC ACR value
	AgentAuthTrustLevel string `json:"agentauth_trust_level"` // "low", "substantial", "high"
	MinMFARequired      bool   `json:"min_mfa_required"`
	Description         string `json:"description"`
}

// Default ACR → TrustLevel mappings
var DefaultACRMappings = []TrustLevelMapping{
	{
		ACR:                 "0",
		AgentAuthTrustLevel: "low",
		MinMFARequired:      false,
		Description:         "No specific authentication context",
	},
	{
		ACR:                 "1",
		AgentAuthTrustLevel: "low",
		MinMFARequired:      false,
		Description:         "Basic authentication (password only)",
	},
	{
		ACR:                 "2",
		AgentAuthTrustLevel: "substantial",
		MinMFARequired:      true,
		Description:         "Multi-factor authentication",
	},
	{
		ACR:                 "substantial",
		AgentAuthTrustLevel: "substantial",
		MinMFARequired:      true,
		Description:         "eIDAS Substantial - MFA required",
	},
	{
		ACR:                 "high",
		AgentAuthTrustLevel: "high",
		MinMFARequired:      true,
		Description:         "eIDAS High - Hardware token required",
	},
	{
		ACR:                 "loa-4",
		AgentAuthTrustLevel: "high",
		MinMFARequired:      true,
		Description:         "NIST LOA-4 - Highest assurance",
	},
	{
		ACR:                 "urn:mace:incommon:iap:silver",
		AgentAuthTrustLevel: "substantial",
		MinMFARequired:      true,
		Description:         "InCommon Silver - Research/Education institutions",
	},
	{
		ACR:                 "urn:mace:incommon:iap:bronze",
		AgentAuthTrustLevel: "low",
		MinMFARequired:      false,
		Description:         "InCommon Bronze - Basic authentication",
	},
}

// JWKSKey represents a JSON Web Key from JWKS endpoint
type JWKSKey struct {
	KID     string   `json:"kid"`      // Key ID
	Kty     string   `json:"kty"`      // Key Type (RSA, EC)
	Alg     string   `json:"alg"`      // Algorithm (RS256, ES256)
	Use     string   `json:"use"`      // Usage (sig, enc)
	N       string   `json:"n"`        // RSA modulus
	E       string   `json:"e"`        // RSA exponent
	X       string   `json:"x"`        // EC x coordinate
	Y       string   `json:"y"`        // EC y coordinate
	Crv     string   `json:"crv"`      // EC curve
	X5c     []string `json:"x5c"`      // X.509 certificate chain
	X5t     string   `json:"x5t"`      // X.509 thumbprint (SHA-1)
	X5tS256 string   `json:"x5t#S256"` // X.509 thumbprint (SHA-256)
}

// JWKS represents JSON Web Key Set
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

// OIDCError represents OIDC-specific errors
type OIDCError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// Error implements error interface
func (e *OIDCError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("OIDC error %s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return fmt.Sprintf("OIDC error: %s", e.ErrorCode)
}

// Standard OIDC error codes
const (
	ErrorInvalidRequest          = "invalid_request"
	ErrorInvalidToken            = "invalid_token"
	ErrorInvalidClient           = "invalid_client"
	ErrorInvalidGrant            = "invalid_grant"
	ErrorUnauthorizedClient      = "unauthorized_client"
	ErrorUnsupportedGrantType    = "unsupported_grant_type"
	ErrorUnsupportedResponseType = "unsupported_response_type"
	ErrorInvalidScope            = "invalid_scope"
	ErrorAccessDenied            = "access_denied"
	ErrorServerError             = "server_error"
	ErrorTemporarilyUnavailable  = "temporarily_unavailable"
)

// ProofMethod constants for OIDC
const (
	ProofMethodOIDCIDToken  = "oidc_id_token" // #nosec G101 // AgentAuth-issued ID token
	ProofMethodOIDCExternal = "oidc_external" // External provider (Google, Okta, etc.)
)
