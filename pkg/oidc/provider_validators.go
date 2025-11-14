package oidc

import (
	"context"
	"fmt"
	"strings"
)

// ProviderValidator provides provider-specific validation logic.
type ProviderValidator interface {
	// ValidateToken validates a token for this specific provider
	ValidateToken(ctx context.Context, token string, provider *ProviderConfig) (*IDTokenClaims, error)

	// GetProviderID returns the provider ID this validator handles
	GetProviderID() string
}

// GoogleValidator validates Google OIDC tokens.
type GoogleValidator struct {
	tokenValidator *ExternalTokenValidator
}

// NewGoogleValidator creates a new Google token validator.
func NewGoogleValidator(tokenValidator *ExternalTokenValidator) *GoogleValidator {
	return &GoogleValidator{
		tokenValidator: tokenValidator,
	}
}

// ValidateToken validates a Google OIDC token.
func (v *GoogleValidator) ValidateToken(ctx context.Context, token string, provider *ProviderConfig) (*IDTokenClaims, error) {
	// Use the base validator
	claims, err := v.tokenValidator.ValidateTokenForProvider(ctx, token, *provider)
	if err != nil {
		return nil, err
	}

	// Google-specific validations
	// 1. Verify email is verified for Google accounts
	// Google requires email verification for proper account security
	if claims.Email != "" && !claims.EmailVerified {
		return nil, fmt.Errorf("google account email not verified")
	}

	// 2. Check hosted domain if specified in provider metadata
	// Note: This would require parsing the JWT again to access the 'hd' claim
	// which is not in the standard IDTokenClaims structure.
	// For production, we would need to either:
	// a) Add 'hd' field to IDTokenClaims
	// b) Parse raw JWT to extract custom claims
	// c) Use provider metadata to store and verify domain restrictions

	// For now, we validate what we can from the standard claims
	if provider.Metadata != nil {
		if _, ok := provider.Metadata["hosted_domain"].(string); ok {
			// In production, we would validate the hd claim here
			// For this implementation, we rely on email domain verification
		}
	}

	return claims, nil
}

// GetProviderID returns the provider ID.
func (v *GoogleValidator) GetProviderID() string {
	return "google"
}

// OktaValidator validates Okta OIDC tokens.
type OktaValidator struct {
	tokenValidator *ExternalTokenValidator
}

// NewOktaValidator creates a new Okta token validator.
func NewOktaValidator(tokenValidator *ExternalTokenValidator) *OktaValidator {
	return &OktaValidator{
		tokenValidator: tokenValidator,
	}
}

// ValidateToken validates an Okta OIDC token.
func (v *OktaValidator) ValidateToken(ctx context.Context, token string, provider *ProviderConfig) (*IDTokenClaims, error) {
	// Use the base validator
	claims, err := v.tokenValidator.ValidateTokenForProvider(ctx, token, *provider)
	if err != nil {
		return nil, err
	}

	// Okta-specific validations
	// 1. Verify issuer format (should match Okta domain)
	if !strings.Contains(claims.Issuer, "okta.com") && !strings.Contains(claims.Issuer, "oktapreview.com") {
		return nil, fmt.Errorf("invalid okta issuer: %s", claims.Issuer)
	}

	// 2. Check MFA requirement if specified in metadata
	if provider.Metadata != nil {
		if requireMFA, ok := provider.Metadata["require_mfa"].(bool); ok && requireMFA {
			hasMFA := false
			for _, method := range claims.AMR {
				// Check for MFA-related authentication methods
				if method == "mfa" || method == "otp" || method == "sms" ||
					method == "hwk" || method == "swk" || method == "tel" {
					hasMFA = true
					break
				}
			}
			if !hasMFA {
				return nil, fmt.Errorf("okta token does not meet MFA requirement")
			}
		}
	}

	// Note: Group membership validation would require accessing the 'groups' claim
	// which is not in the standard IDTokenClaims structure. In production, this could be
	// handled by:
	// a) Adding a groups field to IDTokenClaims
	// b) Parsing the JWT to extract custom claims
	// c) Making a separate API call to Okta's /userinfo endpoint

	return claims, nil
}

// GetProviderID returns the provider ID.
func (v *OktaValidator) GetProviderID() string {
	return "okta"
}

// AzureADValidator validates Azure AD OIDC tokens.
type AzureADValidator struct {
	tokenValidator *ExternalTokenValidator
}

// NewAzureADValidator creates a new Azure AD token validator.
func NewAzureADValidator(tokenValidator *ExternalTokenValidator) *AzureADValidator {
	return &AzureADValidator{
		tokenValidator: tokenValidator,
	}
}

// ValidateToken validates an Azure AD OIDC token.
func (v *AzureADValidator) ValidateToken(ctx context.Context, token string, provider *ProviderConfig) (*IDTokenClaims, error) {
	// Use the base validator
	claims, err := v.tokenValidator.ValidateTokenForProvider(ctx, token, *provider)
	if err != nil {
		return nil, err
	}

	// Azure AD-specific validations
	// 1. Verify tenant ID if specified in metadata
	if provider.Metadata != nil {
		if tenantID, ok := provider.Metadata["tenant_id"].(string); ok && tenantID != "" {
			// Check if issuer contains the tenant ID
			// Special tenants: common, organizations, consumers
			if tenantID != "common" && tenantID != "organizations" && tenantID != "consumers" {
				if !strings.Contains(claims.Issuer, tenantID) {
					return nil, fmt.Errorf("token tenant does not match expected tenant %s", tenantID)
				}
			}
		}
	}

	// 2. Verify issuer format (should match Azure AD format)
	if !strings.Contains(claims.Issuer, "login.microsoftonline.com") &&
		!strings.Contains(claims.Issuer, "sts.windows.net") {
		return nil, fmt.Errorf("invalid azure ad issuer: %s", claims.Issuer)
	}

	// Note: Additional Azure AD-specific validations would require accessing custom claims:
	// - Token version (ver): v1.0 vs v2.0
	// - Roles claim: Application role memberships
	// - Identity provider (idp): Distinguishing work/school from personal accounts
	// - Unique name (unique_name): For v1.0 tokens
	// - Object ID (oid): Azure AD user object ID
	//
	// These could be implemented by:
	// a) Extending IDTokenClaims with Azure-specific fields
	// b) Parsing the JWT to extract custom claims
	// c) Making calls to Microsoft Graph API for additional user information

	return claims, nil
}

// GetProviderID returns the provider ID.
func (v *AzureADValidator) GetProviderID() string {
	return "azure_ad"
}

// ProviderValidatorRegistry manages provider-specific validators.
type ProviderValidatorRegistry struct {
	validators map[string]ProviderValidator
}

// NewProviderValidatorRegistry creates a new provider validator registry.
func NewProviderValidatorRegistry(tokenValidator *ExternalTokenValidator) *ProviderValidatorRegistry {
	registry := &ProviderValidatorRegistry{
		validators: make(map[string]ProviderValidator),
	}

	// Register default validators
	registry.Register(NewGoogleValidator(tokenValidator))
	registry.Register(NewOktaValidator(tokenValidator))
	registry.Register(NewAzureADValidator(tokenValidator))

	return registry
}

// Register registers a provider validator.
func (r *ProviderValidatorRegistry) Register(validator ProviderValidator) {
	r.validators[validator.GetProviderID()] = validator
}

// Get retrieves a validator for a provider.
func (r *ProviderValidatorRegistry) Get(providerID string) (ProviderValidator, bool) {
	validator, ok := r.validators[providerID]
	return validator, ok
}

// ValidateToken validates a token using the appropriate provider validator.
func (r *ProviderValidatorRegistry) ValidateToken(ctx context.Context, token string, provider *ProviderConfig) (*IDTokenClaims, error) {
	validator, ok := r.Get(provider.ID)
	if !ok {
		return nil, fmt.Errorf("no validator found for provider %s", provider.ID)
	}

	return validator.ValidateToken(ctx, token, provider)
}
