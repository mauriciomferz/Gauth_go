// Package oidc - Token Exchange
// Implements token exchange between external OIDC providers and GAuth
package oidc

import (
	"context"
	"fmt"
	"time"
)

// TokenExchangeService handles token exchange between external providers and GAuth.
type TokenExchangeService struct {
	providerRegistry  ProviderRegistry
	idTokenService    *IDTokenService
	tokenValidator    *ExternalTokenValidator
	jwksFetcher       JWKSFetcher
	discoveryCache    DiscoveryCache
}

// TokenExchangeConfig configures the token exchange service.
type TokenExchangeConfig struct {
	ProviderRegistry  ProviderRegistry
	IDTokenService    *IDTokenService
	TokenValidator    *ExternalTokenValidator
	JWKSFetcher       JWKSFetcher
	DiscoveryCache    DiscoveryCache
}

// NewTokenExchangeService creates a new token exchange service.
func NewTokenExchangeService(config TokenExchangeConfig) (*TokenExchangeService, error) {
	if config.ProviderRegistry == nil {
		return nil, fmt.Errorf("provider registry is required")
	}

	if config.IDTokenService == nil {
		return nil, fmt.Errorf("ID token service is required")
	}

	// If TokenValidator is not provided, create one with JWKS fetcher and discovery cache
	tokenValidator := config.TokenValidator
	jwksFetcher := config.JWKSFetcher
	discoveryCache := config.DiscoveryCache

	if tokenValidator == nil {
		// Create default components if not provided
		if jwksFetcher == nil {
			jwksFetcher = NewInMemoryJWKSFetcher(24 * time.Hour)
		}
		if discoveryCache == nil {
			discoveryCache = NewInMemoryDiscoveryCache(WithDefaultTTL(24 * time.Hour))
		}
		tokenValidator = NewExternalTokenValidator(jwksFetcher, discoveryCache)
	}

	return &TokenExchangeService{
		providerRegistry:  config.ProviderRegistry,
		idTokenService:    config.IDTokenService,
		tokenValidator:    tokenValidator,
		jwksFetcher:       jwksFetcher,
		discoveryCache:    discoveryCache,
	}, nil
}

// ExchangeRequest represents a token exchange request.
type ExchangeRequest struct {
	// ProviderID identifies the external provider (google, okta, azure_ad)
	ProviderID string

	// ExternalToken is the ID token from the external provider
	ExternalToken string

	// Audience is the expected audience (client ID) for validation
	Audience string

	// Subject is the user identifier from the external provider
	Subject string

	// AdditionalClaims are optional additional claims to include
	AdditionalClaims map[string]interface{}
}

// ExchangeResponse represents the result of a token exchange.
type ExchangeResponse struct {
	// GAuthToken is the new GAuth ID token
	GAuthToken string

	// ExpiresAt is when the token expires
	ExpiresAt time.Time

	// Claims are the normalized claims in GAuth format
	Claims *IDTokenClaims

	// TrustLevel is the determined trust level
	TrustLevel string

	// ProviderID is the original provider
	ProviderID string
}

// ExchangeToken exchanges an external OIDC token for a GAuth token.
func (s *TokenExchangeService) ExchangeToken(ctx context.Context, req ExchangeRequest) (*ExchangeResponse, error) {
	// Validate request
	if req.ProviderID == "" {
		return nil, fmt.Errorf("provider ID is required")
	}

	if req.ExternalToken == "" {
		return nil, fmt.Errorf("external token is required")
	}

	if req.Audience == "" {
		return nil, fmt.Errorf("audience is required")
	}

	// Get provider configuration
	provider, err := s.providerRegistry.Get(req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	if !provider.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", req.ProviderID)
	}

	// Validate external token (this would normally call the provider's validation)
	// For now, we'll create a simple validation flow
	externalClaims, err := s.validateExternalToken(ctx, provider, req.ExternalToken, req.Audience)
	if err != nil {
		return nil, fmt.Errorf("failed to validate external token: %w", err)
	}

	// Normalize claims to GAuth format
	gauthClaims := s.normalizeClaims(provider, externalClaims)

	// Determine trust level
	trustLevel := s.mapTrustLevel(provider, externalClaims)

	// Add additional claims if provided
	if req.AdditionalClaims != nil {
		for key, value := range req.AdditionalClaims {
			if key == "acr" {
				if strVal, ok := value.(string); ok {
					gauthClaims.ACR = strVal
				}
			} else if key == "amr" {
				if arrVal, ok := value.([]string); ok {
					gauthClaims.AMR = arrVal
				}
			}
			// Additional claims can be added to a custom field if needed
		}
	}

	// Issue new GAuth token
	gauthToken, err := s.idTokenService.IssueIDToken(ctx, gauthClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to issue GAuth token: %w", err)
	}

	return &ExchangeResponse{
		GAuthToken: gauthToken,
		ExpiresAt:  gauthClaims.ExpiresAt.Time,
		Claims:     gauthClaims,
		TrustLevel: trustLevel,
		ProviderID: req.ProviderID,
	}, nil
}

// validateExternalToken validates an external provider token using JWKS validation.
func (s *TokenExchangeService) validateExternalToken(ctx context.Context, provider *ProviderConfig, token string, audience string) (*IDTokenClaims, error) {
	// Use the ExternalTokenValidator to validate the token
	claims, err := s.tokenValidator.ValidateTokenForProvider(ctx, token, *provider)
	if err != nil {
		// If validation fails due to key not found, try refreshing JWKS and retry
		if s.jwksFetcher != nil && s.discoveryCache != nil {
			// Get discovery document to get JWKS URI
			doc, docErr := s.discoveryCache.Get(ctx, provider.IssuerURL)
			if docErr == nil && doc.JWKSUri != "" {
				// Refresh JWKS keys
				refreshErr := s.jwksFetcher.RefreshKeys(ctx, doc.JWKSUri)
				if refreshErr == nil {
					// Retry validation with refreshed keys
					claims, err = s.tokenValidator.ValidateTokenForProvider(ctx, token, *provider)
					if err == nil {
						return claims, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Additional provider-specific validations can be performed here
	// For example, checking hosted domain for Google, organization for Okta, tenant for Azure AD

	return claims, nil
}

// normalizeClaims converts provider-specific claims to GAuth format.
func (s *TokenExchangeService) normalizeClaims(provider *ProviderConfig, externalClaims *IDTokenClaims) *IDTokenClaims {
	// Create new claims in GAuth format
	gauthClaims := &IDTokenClaims{
		RegisteredClaims: externalClaims.RegisteredClaims,
	}

	// Copy standard OIDC claims
	gauthClaims.Name = externalClaims.Name
	gauthClaims.GivenName = externalClaims.GivenName
	gauthClaims.FamilyName = externalClaims.FamilyName
	gauthClaims.MiddleName = externalClaims.MiddleName
	gauthClaims.Nickname = externalClaims.Nickname
	gauthClaims.PreferredUsername = externalClaims.PreferredUsername
	gauthClaims.Profile = externalClaims.Profile
	gauthClaims.Picture = externalClaims.Picture
	gauthClaims.Website = externalClaims.Website
	gauthClaims.Email = externalClaims.Email
	gauthClaims.EmailVerified = externalClaims.EmailVerified
	gauthClaims.Gender = externalClaims.Gender
	gauthClaims.Birthdate = externalClaims.Birthdate
	gauthClaims.Zoneinfo = externalClaims.Zoneinfo
	gauthClaims.Locale = externalClaims.Locale
	gauthClaims.PhoneNumber = externalClaims.PhoneNumber
	gauthClaims.PhoneNumberVerified = externalClaims.PhoneNumberVerified
	gauthClaims.UpdatedAt = externalClaims.UpdatedAt

	// Copy authentication context
	gauthClaims.ACR = externalClaims.ACR
	gauthClaims.AMR = externalClaims.AMR
	gauthClaims.AuthorizedParty = externalClaims.AuthorizedParty
	gauthClaims.Nonce = externalClaims.Nonce

	// Apply provider-specific claim mappings if needed
	// This is where custom transformations would happen based on provider.ClaimMappings

	return gauthClaims
}

// mapTrustLevel determines the GAuth trust level from external provider claims.
func (s *TokenExchangeService) mapTrustLevel(provider *ProviderConfig, claims *IDTokenClaims) string {
	// Check if provider has custom trust level mapping
	if provider.Metadata != nil {
		if trustMapping, ok := provider.Metadata["trust_mapping"].(map[string]string); ok {
			// Check ACR mapping
			if claims.ACR != "" {
				if mappedLevel, exists := trustMapping[claims.ACR]; exists {
					return mappedLevel
				}
			}
		}
	}

	// Default trust level logic based on ACR and AMR
	if claims.ACR != "" {
		// Common eIDAS levels
		switch claims.ACR {
		case "high", "urn:eidas:loa:high":
			return "high"
		case "substantial", "urn:eidas:loa:substantial":
			return "substantial"
		case "low", "urn:eidas:loa:low":
			return "low"
		}
	}

	// Check AMR for MFA indicators
	for _, method := range claims.AMR {
		switch method {
		case "mfa", "otp", "sms", "hwk", "swk", "tel":
			return "high"
		}
	}

	// Default to provider's default trust level
	if provider.DefaultTrustLevel != "" {
		return provider.DefaultTrustLevel
	}

	return "substantial"
}

// BatchExchangeRequest represents multiple token exchange requests.
type BatchExchangeRequest struct {
	Requests []ExchangeRequest
}

// BatchExchangeResponse represents multiple token exchange responses.
type BatchExchangeResponse struct {
	Responses []*ExchangeResponse
	Errors    []error
}

// BatchExchangeTokens exchanges multiple external tokens in a single operation.
func (s *TokenExchangeService) BatchExchangeTokens(ctx context.Context, req BatchExchangeRequest) (*BatchExchangeResponse, error) {
	responses := make([]*ExchangeResponse, len(req.Requests))
	errors := make([]error, len(req.Requests))

	for i, exchangeReq := range req.Requests {
		resp, err := s.ExchangeToken(ctx, exchangeReq)
		responses[i] = resp
		errors[i] = err
	}

	return &BatchExchangeResponse{
		Responses: responses,
		Errors:    errors,
	}, nil
}

// ValidateProviderToken validates a token from a specific provider without exchange.
func (s *TokenExchangeService) ValidateProviderToken(ctx context.Context, providerID string, token string, audience string) (*IDTokenClaims, error) {
	// Get provider configuration
	provider, err := s.providerRegistry.Get(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	if !provider.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", providerID)
	}

	// Validate external token
	claims, err := s.validateExternalToken(ctx, provider, token, audience)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	return claims, nil
}

// GetSupportedProviders returns a list of enabled provider IDs.
func (s *TokenExchangeService) GetSupportedProviders() []string {
	providers := s.providerRegistry.List()

	var supported []string
	for _, provider := range providers {
		if provider.Enabled {
			supported = append(supported, provider.ID)
		}
	}

	return supported
}

// GetProviderInfo returns configuration information for a specific provider.
func (s *TokenExchangeService) GetProviderInfo(providerID string) (*ProviderConfig, error) {
	provider, err := s.providerRegistry.Get(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

// RevokeExchangedToken revokes a previously exchanged token (placeholder).
// In production, this would interact with a token revocation service.
func (s *TokenExchangeService) RevokeExchangedToken(ctx context.Context, token string) error {
	// Placeholder for token revocation logic
	// This would typically:
	// 1. Parse and validate the token
	// 2. Add token to revocation list
	// 3. Notify downstream services
	// 4. Clean up any associated sessions

	return fmt.Errorf("token revocation not implemented")
}

// RefreshExchangedToken refreshes an exchanged token using a refresh token (placeholder).
// In production, this would interact with the original provider's token endpoint.
func (s *TokenExchangeService) RefreshExchangedToken(ctx context.Context, refreshToken string, providerID string) (*ExchangeResponse, error) {
	// Placeholder for token refresh logic
	// This would typically:
	// 1. Call provider's token refresh endpoint
	// 2. Validate the new token
	// 3. Exchange for new GAuth token
	// 4. Return new tokens

	return nil, fmt.Errorf("token refresh not implemented")
}
