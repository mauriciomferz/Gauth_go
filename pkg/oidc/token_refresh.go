// Package oidc - Token Refresh Operations
// Implements RFC 6749 token refresh flow with token rotation and security best practices
package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RefreshTokenRequest represents a token refresh request per RFC 6749 Section 6.
type RefreshTokenRequest struct {
	RefreshToken string   `json:"refresh_token"`
	GrantType    string   `json:"grant_type"` // Must be "refresh_token"
	Scope        []string `json:"scope,omitempty"`
	ClientID     string   `json:"client_id,omitempty"`
}

// RefreshTokenResponse represents the successful token refresh response.
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"` // Always "Bearer"
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"` // New refresh token if rotated
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenRotationPolicy defines refresh token rotation behavior.
type TokenRotationPolicy struct {
	// RotateOnRefresh indicates whether to issue a new refresh token on each refresh
	RotateOnRefresh bool

	// MaxRefreshCount limits how many times a refresh token can be used (0 = unlimited)
	MaxRefreshCount int

	// RefreshTokenLifetime is the lifetime of the new refresh token
	RefreshTokenLifetime time.Duration

	// RevokeUsedTokens indicates whether to revoke the old refresh token after use
	RevokeUsedTokens bool
}

// DefaultRotationPolicy returns the recommended token rotation policy.
func DefaultRotationPolicy() *TokenRotationPolicy {
	return &TokenRotationPolicy{
		RotateOnRefresh:      true,               // Best practice for security
		MaxRefreshCount:      10,                 // Limit refresh token lifetime
		RefreshTokenLifetime: 7 * 24 * time.Hour, // 7 days
		RevokeUsedTokens:     true,               // One-time use per RFC 6819
	}
}

// RefreshTokenManager handles token refresh operations with rotation support.
type RefreshTokenManager struct {
	refreshTokenService *RefreshTokenService
	revocationService   *TokenRevocationService
	idTokenService      *IDTokenService
	providerRegistry    ProviderRegistry
	rotationPolicy      *TokenRotationPolicy
}

// NewRefreshTokenManager creates a new token refresh manager.
func NewRefreshTokenManager(
	refreshTokenService *RefreshTokenService,
	revocationService *TokenRevocationService,
	idTokenService *IDTokenService,
	providerRegistry ProviderRegistry,
) *RefreshTokenManager {
	return &RefreshTokenManager{
		refreshTokenService: refreshTokenService,
		revocationService:   revocationService,
		idTokenService:      idTokenService,
		providerRegistry:    providerRegistry,
		rotationPolicy:      DefaultRotationPolicy(),
	}
}

// SetRotationPolicy updates the token rotation policy.
func (m *RefreshTokenManager) SetRotationPolicy(policy *TokenRotationPolicy) {
	m.rotationPolicy = policy
}

// RefreshToken performs the complete token refresh flow per RFC 6749 Section 6.
func (m *RefreshTokenManager) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	// 1. Validate request
	if err := m.validateRefreshRequest(req); err != nil {
		return nil, fmt.Errorf("invalid refresh request: %w", err)
	}

	// 2. Retrieve refresh token entry
	entry, err := m.refreshTokenService.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found or expired: %w", err)
	}

	// 3. Validate refresh token status
	if err := m.validateRefreshTokenEntry(entry); err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// 4. Check usage count
	if m.rotationPolicy.MaxRefreshCount > 0 && entry.UseCount >= m.rotationPolicy.MaxRefreshCount {
		// Revoke the refresh token family on excessive use
		_ = m.refreshTokenService.RevokeRefreshToken(ctx, req.RefreshToken)
		return nil, fmt.Errorf("refresh token exceeded maximum usage count")
	}

	// 5. Get provider configuration
	provider, err := m.providerRegistry.Get(entry.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// 6. Generate new ID token
	newIDToken, err := m.generateNewIDToken(ctx, entry, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new ID token: %w", err)
	}

	// 7. Generate new access token (for now, same as ID token)
	newAccessToken := newIDToken

	// 8. Handle refresh token rotation
	var newRefreshToken string
	if m.rotationPolicy.RotateOnRefresh {
		newRefreshToken, err = m.rotateRefreshToken(ctx, req.RefreshToken, entry)
		if err != nil {
			return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
		}
	} else {
		// Update usage count on existing token
		if err := m.refreshTokenService.UpdateRefreshTokenUsage(ctx, req.RefreshToken); err != nil {
			return nil, fmt.Errorf("failed to update refresh token usage: %w", err)
		}
		newRefreshToken = req.RefreshToken // Return same refresh token
	}

	// 9. Build response
	response := &RefreshTokenResponse{
		AccessToken:  newAccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		IDToken:      newIDToken,
		RefreshToken: newRefreshToken,
	}

	// Set scope if requested
	if len(req.Scope) > 0 {
		response.Scope = joinScopes(req.Scope)
	} else if len(entry.Scopes) > 0 {
		response.Scope = joinScopes(entry.Scopes)
	}

	return response, nil
}

// validateRefreshRequest validates the refresh token request.
func (m *RefreshTokenManager) validateRefreshRequest(req *RefreshTokenRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.RefreshToken == "" {
		return fmt.Errorf("refresh_token is required")
	}

	if req.GrantType != "" && req.GrantType != "refresh_token" {
		return fmt.Errorf("invalid grant_type: expected 'refresh_token', got '%s'", req.GrantType)
	}

	return nil
}

// validateRefreshTokenEntry validates the refresh token entry status.
func (m *RefreshTokenManager) validateRefreshTokenEntry(entry *RefreshTokenEntry) error {
	if entry == nil {
		return fmt.Errorf("entry is nil")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return fmt.Errorf("refresh token has expired")
	}

	// Check revocation
	if entry.Revoked {
		return fmt.Errorf("refresh token has been revoked")
	}

	return nil
}

// generateNewIDToken creates a new ID token with refreshed claims.
func (m *RefreshTokenManager) generateNewIDToken(ctx context.Context, entry *RefreshTokenEntry, provider *ProviderConfig) (string, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	// Create new ID token claims
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   entry.Subject,
			Audience:  jwt.ClaimStrings{entry.Audience},
			Issuer:    provider.IssuerURL,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        generateTokenID(),
		},
	}

	// Add standard OIDC claims if available
	if entry.Email != "" {
		claims.Email = entry.Email
		claims.EmailVerified = entry.EmailVerified
	}
	if entry.Name != "" {
		claims.Name = entry.Name
	}

	// Sign the token
	token, err := m.idTokenService.IssueIDToken(ctx, claims)
	if err != nil {
		return "", fmt.Errorf("failed to issue ID token: %w", err)
	}

	return token, nil
}

// rotateRefreshToken creates a new refresh token and revokes the old one.
func (m *RefreshTokenManager) rotateRefreshToken(ctx context.Context, oldToken string, oldEntry *RefreshTokenEntry) (string, error) {
	// Generate new refresh token
	newToken, err := generateSecureToken(32) // 256 bits
	if err != nil {
		return "", fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Create new refresh token entry
	newEntry := &RefreshTokenEntry{
		RefreshToken:  newToken,
		Subject:       oldEntry.Subject,
		ProviderID:    oldEntry.ProviderID,
		Audience:      oldEntry.Audience,
		Scopes:        oldEntry.Scopes,
		Email:         oldEntry.Email,
		EmailVerified: oldEntry.EmailVerified,
		Name:          oldEntry.Name,
		IssuedAt:      time.Now(),
		ExpiresAt:     time.Now().Add(m.rotationPolicy.RefreshTokenLifetime),
		UseCount:      0, // Reset usage count for new token
		LastUsed:      time.Time{},
		Revoked:       false,
	}

	// Store new refresh token
	if err := m.refreshTokenService.StoreRefreshToken(ctx, newToken, newEntry); err != nil {
		return "", fmt.Errorf("failed to store new refresh token: %w", err)
	}

	// Revoke old refresh token if policy requires
	if m.rotationPolicy.RevokeUsedTokens {
		if err := m.refreshTokenService.RevokeRefreshToken(ctx, oldToken); err != nil {
			// Log error but don't fail the operation
			// The new token is already issued
			_ = err
		}
	}

	return newToken, nil
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// joinScopes converts a scope slice to a space-separated string.
func joinScopes(scopes []string) string {
	result := ""
	for i, scope := range scopes {
		if i > 0 {
			result += " "
		}
		result += scope
	}
	return result
}
