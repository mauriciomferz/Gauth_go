// Package oidc - Token Exchange
// Implements token exchange between external OIDC providers and AgentAuth
package oidc

import (
	"context"
	"fmt"
	"time"
)

// TokenExchangeService handles token exchange between external providers and AgentAuth.
type TokenExchangeService struct {
	providerRegistry     ProviderRegistry
	idTokenService       *IDTokenService
	tokenValidator       *ExternalTokenValidator
	jwksFetcher          JWKSFetcher
	discoveryCache       DiscoveryCache
	revocationService    *TokenRevocationService
	refreshTokenService  *RefreshTokenService
	refreshTokenManager  *RefreshTokenManager
	revocationHandler    *TokenRevocationHandler
	introspectionHandler *TokenIntrospectionHandler
}

// TokenExchangeConfig configures the token exchange service.
type TokenExchangeConfig struct {
	ProviderRegistry    ProviderRegistry
	IDTokenService      *IDTokenService
	TokenValidator      *ExternalTokenValidator
	JWKSFetcher         JWKSFetcher
	DiscoveryCache      DiscoveryCache
	RevocationService   *TokenRevocationService
	RefreshTokenService *RefreshTokenService
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

	// Create refresh token manager
	refreshTokenManager := NewRefreshTokenManager(
		refreshTokenService,
		revocationService,
		config.IDTokenService,
		config.ProviderRegistry,
	)

	// Create revocation handler
	revocationHandler := NewTokenRevocationHandler(
		revocationService,
		refreshTokenService,
		config.IDTokenService,
		config.ProviderRegistry,
	)

	// Create introspection handler
	introspectionHandler := NewTokenIntrospectionHandler(
		config.IDTokenService,
		refreshTokenService,
		revocationService,
		config.ProviderRegistry,
	)

	return &TokenExchangeService{
		providerRegistry:     config.ProviderRegistry,
		idTokenService:       config.IDTokenService,
		tokenValidator:       tokenValidator,
		jwksFetcher:          jwksFetcher,
		discoveryCache:       discoveryCache,
		revocationService:    revocationService,
		refreshTokenService:  refreshTokenService,
		refreshTokenManager:  refreshTokenManager,
		revocationHandler:    revocationHandler,
		introspectionHandler: introspectionHandler,
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
	// AgentAuthToken is the new AgentAuth ID token
	AgentAuthToken string

	// ExpiresAt is when the token expires
	ExpiresAt time.Time

	// Claims are the normalized claims in AgentAuth format
	Claims *IDTokenClaims

	// TrustLevel is the determined trust level
	TrustLevel string

	// ProviderID is the original provider
	ProviderID string

	// RefreshToken is the new refresh token (if rotated)
	RefreshToken string
}

// ExchangeToken exchanges an external OIDC token for a AgentAuth token.
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

	// Normalize claims to AgentAuth format
	agentauthClaims := s.normalizeClaims(provider, externalClaims)

	// Determine trust level
	trustLevel := s.mapTrustLevel(provider, externalClaims)

	// Add additional claims if provided
	if req.AdditionalClaims != nil {
		for key, value := range req.AdditionalClaims {
			if key == "acr" {
				if strVal, ok := value.(string); ok {
					agentauthClaims.ACR = strVal
				}
			} else if key == "amr" {
				if arrVal, ok := value.([]string); ok {
					agentauthClaims.AMR = arrVal
				}
			}
			// Additional claims can be added to a custom field if needed
		}
	}

	// Issue new AgentAuth token
	agentauthToken, err := s.idTokenService.IssueIDToken(ctx, agentauthClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to issue AgentAuth token: %w", err)
	}

	return &ExchangeResponse{
		AgentAuthToken: agentauthToken,
		ExpiresAt:  agentauthClaims.ExpiresAt.Time,
		Claims:     agentauthClaims,
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

// normalizeClaims converts provider-specific claims to AgentAuth format.
func (s *TokenExchangeService) normalizeClaims(provider *ProviderConfig, externalClaims *IDTokenClaims) *IDTokenClaims {
	// Create new claims in AgentAuth format
	agentauthClaims := &IDTokenClaims{
		RegisteredClaims: externalClaims.RegisteredClaims,
	}

	// Copy standard OIDC claims
	agentauthClaims.Name = externalClaims.Name
	agentauthClaims.GivenName = externalClaims.GivenName
	agentauthClaims.FamilyName = externalClaims.FamilyName
	agentauthClaims.MiddleName = externalClaims.MiddleName
	agentauthClaims.Nickname = externalClaims.Nickname
	agentauthClaims.PreferredUsername = externalClaims.PreferredUsername
	agentauthClaims.Profile = externalClaims.Profile
	agentauthClaims.Picture = externalClaims.Picture
	agentauthClaims.Website = externalClaims.Website
	agentauthClaims.Email = externalClaims.Email
	agentauthClaims.EmailVerified = externalClaims.EmailVerified
	agentauthClaims.Gender = externalClaims.Gender
	agentauthClaims.Birthdate = externalClaims.Birthdate
	agentauthClaims.Zoneinfo = externalClaims.Zoneinfo
	agentauthClaims.Locale = externalClaims.Locale
	agentauthClaims.PhoneNumber = externalClaims.PhoneNumber
	agentauthClaims.PhoneNumberVerified = externalClaims.PhoneNumberVerified
	agentauthClaims.UpdatedAt = externalClaims.UpdatedAt

	// Copy authentication context
	agentauthClaims.ACR = externalClaims.ACR
	agentauthClaims.AMR = externalClaims.AMR
	agentauthClaims.AuthorizedParty = externalClaims.AuthorizedParty
	agentauthClaims.Nonce = externalClaims.Nonce

	// Apply provider-specific claim mappings if needed
	// This is where custom transformations would happen based on provider.ClaimMappings

	return agentauthClaims
}

// mapTrustLevel determines the AgentAuth trust level from external provider claims.
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
			return trustLevelHigh
		case "substantial", "urn:eidas:loa:substantial":
			return trustLevelSubstantial
		case "low", "urn:eidas:loa:low":
			return "low"
		}
	}

	// Check AMR for MFA indicators
	for _, method := range claims.AMR {
		switch method {
		case "mfa", "otp", "sms", "hwk", "swk", "tel":
			return trustLevelHigh
		}
	}

	// Default to provider's default trust level
	if provider.DefaultTrustLevel != "" {
		return provider.DefaultTrustLevel
	}

	return trustLevelSubstantial
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
// Implements RFC 7009 token revocation.
func (s *TokenExchangeService) RevokeExchangedToken(ctx context.Context, token string, reason string, revokedBy string) error {
	// Build revocation request
	req := &RevocationRequest{
		Token:         token,
		TokenTypeHint: "access_token", // Assume access token by default
		Reason:        reason,
		RevokedBy:     revokedBy,
	}

	// Use the revocation handler
	_, err := s.revocationHandler.RevokeToken(ctx, req)
	if err != nil {
		// Check if it's a RevocationError (which means validation failed)
		if revErr, ok := err.(*RevocationError); ok {
			return fmt.Errorf("revocation failed: %s", revErr.ErrorDescription)
		}
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// RefreshExchangedToken refreshes an exchanged token using a refresh token.
// Implements RFC 6749 token refresh flow with token rotation.
func (s *TokenExchangeService) RefreshExchangedToken(ctx context.Context, refreshToken string, providerID string) (*ExchangeResponse, error) {
	// Build refresh request
	req := &RefreshTokenRequest{
		RefreshToken: refreshToken,
		GrantType:    "refresh_token",
		ClientID:     providerID,
	}

	// Use the refresh token manager to handle the refresh
	response, err := s.refreshTokenManager.RefreshToken(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Parse the new ID token to extract claims
	claims, err := s.idTokenService.ValidateIDToken(ctx, response.IDToken, "")
	if err != nil {
		return nil, fmt.Errorf("failed to validate refreshed ID token: %w", err)
	}

	// Build exchange response
	exchangeResponse := &ExchangeResponse{
		AgentAuthToken: response.IDToken,
		ExpiresAt:  claims.ExpiresAt.Time,
		Claims:     claims,
		TrustLevel: determineRefreshTrustLevel(claims),
		ProviderID: providerID,
	}

	// Store the new refresh token if rotated
	if response.RefreshToken != refreshToken {
		// Update refresh token in context (caller should store this)
		exchangeResponse.RefreshToken = response.RefreshToken
	}

	return exchangeResponse, nil
}

// generateTokenID generates a unique token ID for JTI claim
func generateTokenID() string {
	return fmt.Sprintf("jti_%d", time.Now().UnixNano())
}

// determineRefreshTrustLevel determines the trust level for a refreshed token.
// Refreshed tokens generally have lower trust than initial authentication.
func determineRefreshTrustLevel(claims *IDTokenClaims) string {
	// Check for ACR (Authentication Context Class Reference)
	if claims.ACR != "" {
		// If original authentication had high ACR, maintain substantial trust
		if claims.ACR == "2" || claims.ACR == "3" || claims.ACR == "c1" || claims.ACR == "c2" || claims.ACR == "c3" {
			return trustLevelSubstantial
		}
		if claims.ACR == "1" {
			return trustLevelSubstantial
		}
	}

	// Check for AMR (Authentication Methods References)
	if len(claims.AMR) > 0 {
		for _, amr := range claims.AMR {
			// MFA-based authentication maintains substantial trust
			if amr == "mfa" || amr == "otp" || amr == "sms" || amr == "hwk" || amr == "swk" {
				return trustLevelSubstantial
			}
		}
	}

	// Default to low trust for refresh tokens
	// This follows security best practice that refresh tokens have lower assurance
	return "low"
}

// IntrospectToken introspects a token and returns its metadata.
// Implements RFC 7662 token introspection endpoint.
func (s *TokenExchangeService) IntrospectToken(ctx context.Context, token string, tokenTypeHint string) (*IntrospectionResponse, error) {
	// Build introspection request
	req := &IntrospectionRequest{
		Token:         token,
		TokenTypeHint: tokenTypeHint,
	}

	// Use the introspection handler
	response, err := s.introspectionHandler.IntrospectToken(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}

	return response, nil
}

// IntrospectTokenWithValidation introspects a token with full validation.
// This method performs signature verification and full token validation.
func (s *TokenExchangeService) IntrospectTokenWithValidation(ctx context.Context, token string, tokenTypeHint string, clientID string) (*IntrospectionResponse, error) {
	// Build introspection request
	req := &IntrospectionRequest{
		Token:         token,
		TokenTypeHint: tokenTypeHint,
		ClientID:      clientID,
	}

	// Use the introspection handler with validation
	response, err := s.introspectionHandler.IntrospectTokenWithValidation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}

	return response, nil
}
