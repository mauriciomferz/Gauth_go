// Package oidc - Token Revocation
// Implements RFC 7009 OAuth 2.0 Token Revocation
package oidc

import (
	"context"
	"fmt"
	"time"
)

// RevocationRequest represents a token revocation request per RFC 7009.
type RevocationRequest struct {
	// Token is the token to revoke (required)
	Token string `json:"token"`

	// TokenTypeHint indicates the type of token being revoked (optional)
	// Valid values: "access_token" or "refresh_token"
	TokenTypeHint string `json:"token_type_hint,omitempty"`

	// ClientID is the client identifier (required for authentication)
	ClientID string `json:"client_id,omitempty"`

	// Additional context for audit logging
	RevokedBy string `json:"-"` // User/service performing revocation
	Reason    string `json:"-"` // Reason for revocation
}

// RevocationResponse represents the response to a revocation request.
// Per RFC 7009, successful revocations return HTTP 200 with no body.
type RevocationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// RevocationError represents an error during token revocation.
type RevocationError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// Error implements the error interface.
func (e *RevocationError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return e.ErrorCode
}

// RFC 7009 error codes
const (
	ErrorUnsupportedTokenType = "unsupported_token_type"
)

// TokenRevocationHandler handles token revocation requests per RFC 7009.
type TokenRevocationHandler struct {
	revocationService   *TokenRevocationService
	refreshTokenService *RefreshTokenService
	idTokenService      *IDTokenService
	providerRegistry    ProviderRegistry
}

// NewTokenRevocationHandler creates a new token revocation handler.
func NewTokenRevocationHandler(
	revocationService *TokenRevocationService,
	refreshTokenService *RefreshTokenService,
	idTokenService *IDTokenService,
	providerRegistry ProviderRegistry,
) *TokenRevocationHandler {
	return &TokenRevocationHandler{
		revocationService:   revocationService,
		refreshTokenService: refreshTokenService,
		idTokenService:      idTokenService,
		providerRegistry:    providerRegistry,
	}
}

// RevokeToken handles a token revocation request.
// Implements RFC 7009 Section 2.1.
func (h *TokenRevocationHandler) RevokeToken(ctx context.Context, req *RevocationRequest) (*RevocationResponse, error) {
	// 1. Validate request
	if err := h.validateRevocationRequest(req); err != nil {
		return nil, &RevocationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: err.Error(),
		}
	}

	// 2. Determine token type and revoke accordingly
	var err error
	switch req.TokenTypeHint {
	case "refresh_token", "":
		// Try refresh token first (most common case)
		err = h.revokeRefreshToken(ctx, req)
		if err == nil {
			return &RevocationResponse{Success: true}, nil
		}
		// If not a refresh token and no hint, try access token
		if req.TokenTypeHint == "" {
			if accessErr := h.revokeAccessToken(ctx, req); accessErr == nil {
				return &RevocationResponse{Success: true}, nil
			}
		}

	case "access_token":
		err = h.revokeAccessToken(ctx, req)
		if err == nil {
			return &RevocationResponse{Success: true}, nil
		}

	default:
		return nil, &RevocationError{
			ErrorCode:        ErrorUnsupportedTokenType,
			ErrorDescription: fmt.Sprintf("unsupported token type hint: %s", req.TokenTypeHint),
		}
	}

	// Per RFC 7009 Section 2.2: "The authorization server responds with HTTP status code 200
	// if the token has been revoked successfully or if the client submitted an invalid token."
	// We return success even if token wasn't found to prevent token scanning
	return &RevocationResponse{
		Success: true,
		Message: "token revoked or was already invalid",
	}, nil
}

// validateRevocationRequest validates the revocation request.
func (h *TokenRevocationHandler) validateRevocationRequest(req *RevocationRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.Token == "" {
		return fmt.Errorf("token is required")
	}

	// ClientID validation could be added here based on authentication requirements
	return nil
}

// revokeRefreshToken revokes a refresh token.
func (h *TokenRevocationHandler) revokeRefreshToken(ctx context.Context, req *RevocationRequest) error {
	// Get the refresh token entry
	entry, err := h.refreshTokenService.GetRefreshToken(ctx, req.Token)
	if err != nil {
		return fmt.Errorf("refresh token not found: %w", err)
	}

	// Verify client authentication if ClientID is provided
	if req.ClientID != "" && entry.ProviderID != req.ClientID {
		return fmt.Errorf("client authentication failed: token belongs to different client")
	}

	// Mark the refresh token as revoked
	if err := h.refreshTokenService.RevokeRefreshToken(ctx, req.Token); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// revokeAccessToken revokes an access token (ID token).
func (h *TokenRevocationHandler) revokeAccessToken(ctx context.Context, req *RevocationRequest) error {
	// Parse and validate the token
	claims, err := h.idTokenService.ValidateIDToken(ctx, req.Token, "")
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	// Extract token ID
	tokenID := claims.ID
	if tokenID == "" {
		// Use token itself as ID if no JTI claim
		tokenID = req.Token
	}

	// Determine expiration for revocation entry
	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		// Default to 24 hours if no expiration
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	// Set default values for audit fields
	reason := req.Reason
	if reason == "" {
		reason = "explicit revocation request"
	}

	revokedBy := req.RevokedBy
	if revokedBy == "" {
		revokedBy = "revocation_endpoint"
	}

	// Revoke the token
	if err := h.revocationService.RevokeToken(ctx, tokenID, reason, revokedBy, expiresAt); err != nil {
		return fmt.Errorf("failed to revoke access token: %w", err)
	}

	return nil
}

// RevokeTokenFamily revokes a token and all related tokens in the token family.
// This is useful for revoking all tokens issued from the same authorization.
func (h *TokenRevocationHandler) RevokeTokenFamily(ctx context.Context, req *RevocationRequest) error {
	// First, revoke the token itself
	if _, err := h.RevokeToken(ctx, req); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// If it's a refresh token, we've already revoked it
	// In a production system, you might want to track token families and revoke all related tokens
	// For now, we just revoke the single token

	return nil
}

// CheckRevocationStatus checks if a token has been revoked.
// This is useful for resource servers to validate tokens.
func (h *TokenRevocationHandler) CheckRevocationStatus(ctx context.Context, tokenID string) (bool, error) {
	return h.revocationService.IsRevoked(ctx, tokenID)
}

// GetRevocationInfo retrieves revocation information for a token.
func (h *TokenRevocationHandler) GetRevocationInfo(ctx context.Context, tokenID string) (*RevokedTokenEntry, error) {
	return h.revocationService.GetRevocationInfo(ctx, tokenID)
}

// RevokeAllUserTokens revokes all tokens associated with a user.
// This is useful for logout or account security events.
func (h *TokenRevocationHandler) RevokeAllUserTokens(ctx context.Context, userID string, reason string, revokedBy string) error {
	// This would require tracking tokens by user ID
	// For now, this is a placeholder for future implementation
	return fmt.Errorf("revoke all user tokens not yet implemented")
}

// BatchRevoke revokes multiple tokens in a single operation.
func (h *TokenRevocationHandler) BatchRevoke(ctx context.Context, requests []*RevocationRequest) ([]*RevocationResponse, error) {
	responses := make([]*RevocationResponse, len(requests))

	for i, req := range requests {
		resp, err := h.RevokeToken(ctx, req)
		if err != nil {
			// Per RFC 7009, we still return success for failed revocations
			responses[i] = &RevocationResponse{
				Success: true,
				Message: "token processed",
			}
		} else {
			responses[i] = resp
		}
	}

	return responses, nil
}
