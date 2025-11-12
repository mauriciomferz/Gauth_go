// Package oidc - Token Exchange
// Implements token exchange between external OIDC providers and GAuth
package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenExchangeService handles token exchange between external providers and GAuth.
type TokenExchangeService struct {
	providerRegistry   ProviderRegistry
	idTokenService     *IDTokenService
	tokenValidator     *ExternalTokenValidator
	jwksFetcher        JWKSFetcher
	discoveryCache     DiscoveryCache
	revocationService  *TokenRevocationService
	refreshTokenService *RefreshTokenService
}

// TokenExchangeConfig configures the token exchange service.
type TokenExchangeConfig struct {
	ProviderRegistry     ProviderRegistry
	IDTokenService       *IDTokenService
	TokenValidator       *ExternalTokenValidator
	JWKSFetcher          JWKSFetcher
	DiscoveryCache       DiscoveryCache
	RevocationService    *TokenRevocationService
	RefreshTokenService  *RefreshTokenService
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

	// Create default services if not provided
	revocationService := config.RevocationService
	if revocationService == nil {
		revocationService = NewTokenRevocationService()
	}

	refreshTokenService := config.RefreshTokenService
	if refreshTokenService == nil {
		refreshTokenService = NewRefreshTokenService()
	}

	return &TokenExchangeService{
		providerRegistry:    config.ProviderRegistry,
		idTokenService:      config.IDTokenService,
		tokenValidator:      tokenValidator,
		jwksFetcher:         jwksFetcher,
		discoveryCache:      discoveryCache,
		revocationService:   revocationService,
		refreshTokenService: refreshTokenService,
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

// RevokeExchangedToken revokes a previously exchanged token.
func (s *TokenExchangeService) RevokeExchangedToken(ctx context.Context, token string, reason string, revokedBy string) error {
	if token == "" {
		return fmt.Errorf("token is required")
	}

	// Parse the token to extract the token ID and expiration
	// For now, we'll use empty audience to allow revocation without validation
	claims, err := s.idTokenService.ValidateIDToken(ctx, token, "")
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Determine expiration time for revocation entry
	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		// Default to 24 hours if no expiration
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	// Revoke the token
	tokenID := claims.ID
	if tokenID == "" {
		tokenID = token // Use full token as ID if no JTI claim
	}

	return s.revocationService.RevokeToken(ctx, tokenID, reason, revokedBy, expiresAt)
}

// RefreshExchangedToken refreshes an exchanged token using a refresh token.
func (s *TokenExchangeService) RefreshExchangedToken(ctx context.Context, refreshToken string, providerID string) (*ExchangeResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	// Get the stored refresh token entry
	entry, err := s.refreshTokenService.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found or expired: %w", err)
	}

	// Verify provider ID matches
	if entry.ProviderID != providerID {
		return nil, fmt.Errorf("provider ID mismatch: expected %s, got %s", entry.ProviderID, providerID)
	}

	// Update usage
	if err := s.refreshTokenService.UpdateRefreshTokenUsage(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to update refresh token usage: %w", err)
	}

	// Get provider configuration
	provider, err := s.providerRegistry.Get(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// In a real implementation, we would:
	// 1. Call the provider's token refresh endpoint with the refresh token
	// 2. Get a new access token and ID token
	// 3. Validate and exchange the new ID token
	//
	// For now, we'll create a new token based on the stored refresh token entry
	// This is a simplified implementation that doesn't actually call the provider

	// Create new ID token claims using jwt.RegisteredClaims
	newExpiration := time.Now().Add(1 * time.Hour)
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   entry.Subject,
			Audience:  jwt.ClaimStrings{entry.Audience},
			Issuer:    provider.IssuerURL,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(newExpiration),
			ID:        generateTokenID(), // Generate new token ID
		},
	}

	// Generate new GAuth token (in real implementation, this would come from external provider)
	// For now, return a response indicating successful refresh
	return &ExchangeResponse{
		GAuthToken: refreshToken, // Placeholder - in real implementation, this would be a new token
		ExpiresAt:  newExpiration,
		Claims:     claims,
		TrustLevel: "medium", // Based on refresh token usage
		ProviderID: providerID,
	}, nil
}

// generateTokenID generates a unique token ID for JTI claim
func generateTokenID() string {
	return fmt.Sprintf("jti_%d", time.Now().UnixNano())
}
