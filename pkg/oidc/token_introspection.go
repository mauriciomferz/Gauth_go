// Package oidc - Token Introspection
// Implements RFC 7662 OAuth 2.0 Token Introspection
package oidc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IntrospectionRequest represents a token introspection request per RFC 7662.
type IntrospectionRequest struct {
	// Token is the token to introspect (required)
	Token string `json:"token"`

	// TokenTypeHint indicates the type of token (optional)
	// Valid values: "access_token" or "refresh_token"
	TokenTypeHint string `json:"token_type_hint,omitempty"`

	// ClientID is the client identifier (required for authentication)
	ClientID string `json:"client_id,omitempty"`
}

// IntrospectionResponse represents the response to an introspection request per RFC 7662.
type IntrospectionResponse struct {
	// Active indicates whether the token is currently active
	Active bool `json:"active"`

	// Scope is a space-separated list of scopes
	Scope string `json:"scope,omitempty"`

	// ClientID is the client identifier
	ClientID string `json:"client_id,omitempty"`

	// Username is the human-readable identifier for the resource owner
	Username string `json:"username,omitempty"`

	// TokenType is the type of token (e.g., "Bearer")
	TokenType string `json:"token_type,omitempty"`

	// Exp is the timestamp when the token expires
	Exp int64 `json:"exp,omitempty"`

	// Iat is the timestamp when the token was issued
	Iat int64 `json:"iat,omitempty"`

	// Nbf is the timestamp before which the token is not valid
	Nbf int64 `json:"nbf,omitempty"`

	// Sub is the subject of the token
	Sub string `json:"sub,omitempty"`

	// Aud is the intended audience
	Aud string `json:"aud,omitempty"`

	// Iss is the issuer of the token
	Iss string `json:"iss,omitempty"`

	// Jti is the unique identifier for the token
	Jti string `json:"jti,omitempty"`

	// OIDC specific claims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
}

// IntrospectionError represents an error during token introspection.
type IntrospectionError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// Error implements the error interface.
func (e *IntrospectionError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return e.ErrorCode
}

// TokenIntrospectionHandler handles token introspection operations.
type TokenIntrospectionHandler struct {
	idTokenService      *IDTokenService
	refreshTokenService *RefreshTokenService
	revocationService   *TokenRevocationService
	providerRegistry    ProviderRegistry
}

// NewTokenIntrospectionHandler creates a new token introspection handler.
func NewTokenIntrospectionHandler(
	idTokenService *IDTokenService,
	refreshTokenService *RefreshTokenService,
	revocationService *TokenRevocationService,
	providerRegistry ProviderRegistry,
) *TokenIntrospectionHandler {
	return &TokenIntrospectionHandler{
		idTokenService:      idTokenService,
		refreshTokenService: refreshTokenService,
		revocationService:   revocationService,
		providerRegistry:    providerRegistry,
	}
}

// IntrospectToken introspects a token and returns its metadata per RFC 7662.
func (h *TokenIntrospectionHandler) IntrospectToken(ctx context.Context, req *IntrospectionRequest) (*IntrospectionResponse, error) {
	// Validate request
	if err := h.validateIntrospectionRequest(req); err != nil {
		return nil, err
	}

	// Try to introspect as refresh token first if hint suggests it
	if req.TokenTypeHint == "refresh_token" {
		if resp, err := h.introspectRefreshToken(ctx, req.Token); err == nil {
			return resp, nil
		}
	}

	// Try to introspect as access token (ID token)
	if resp, err := h.introspectAccessToken(ctx, req.Token); err == nil {
		return resp, nil
	}

	// If token type hint was "access_token", don't try refresh token
	if req.TokenTypeHint == "access_token" {
		return &IntrospectionResponse{Active: false}, nil
	}

	// Try refresh token if we haven't already
	if req.TokenTypeHint != "refresh_token" {
		if resp, err := h.introspectRefreshToken(ctx, req.Token); err == nil {
			return resp, nil
		}
	}

	// Token not found or invalid - return inactive response
	// Per RFC 7662 Section 2.2, return active=false for unknown tokens
	return &IntrospectionResponse{Active: false}, nil
}

// validateIntrospectionRequest validates the introspection request.
func (h *TokenIntrospectionHandler) validateIntrospectionRequest(req *IntrospectionRequest) error {
	if req.Token == "" {
		return &IntrospectionError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "token parameter is required",
		}
	}

	// Validate token type hint if provided
	if req.TokenTypeHint != "" {
		if req.TokenTypeHint != "access_token" && req.TokenTypeHint != "refresh_token" {
			return &IntrospectionError{
				ErrorCode:        ErrorUnsupportedTokenType,
				ErrorDescription: "token_type_hint must be 'access_token' or 'refresh_token'",
			}
		}
	}

	return nil
}

// introspectAccessToken introspects an access token (ID token).
func (h *TokenIntrospectionHandler) introspectAccessToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	// Parse the token without validation to extract claims
	// We use ParseUnverified to get claims even if signature verification would fail
	parser := jwt.NewParser()
	parsedToken, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is revoked
	tokenID := ""
	if jti, ok := claims["jti"].(string); ok {
		tokenID = jti
	} else {
		tokenID = token // Use full token if no JTI
	}

	isRevoked, err := h.revocationService.IsRevoked(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to check revocation status: %w", err)
	}

	if isRevoked {
		return &IntrospectionResponse{Active: false}, nil
	}

	// Build response from claims
	resp := &IntrospectionResponse{
		Active:    true,
		TokenType: "Bearer",
	}

	// Extract standard claims
	if sub, ok := claims["sub"].(string); ok {
		resp.Sub = sub
		resp.Username = sub // Use sub as username
	}

	if aud, ok := claims["aud"].(string); ok {
		resp.Aud = aud
		resp.ClientID = aud
	} else if audList, ok := claims["aud"].([]interface{}); ok && len(audList) > 0 {
		if audStr, ok := audList[0].(string); ok {
			resp.Aud = audStr
			resp.ClientID = audStr
		}
	}

	if iss, ok := claims["iss"].(string); ok {
		resp.Iss = iss
	}

	if jti, ok := claims["jti"].(string); ok {
		resp.Jti = jti
	}

	// Extract timestamps
	if exp, ok := claims["exp"].(float64); ok {
		resp.Exp = int64(exp)
		// Check if token is expired
		if time.Now().Unix() > int64(exp) {
			resp.Active = false
			return resp, nil
		}
	}

	if iat, ok := claims["iat"].(float64); ok {
		resp.Iat = int64(iat)
	}

	if nbf, ok := claims["nbf"].(float64); ok {
		resp.Nbf = int64(nbf)
		// Check if token is not yet valid
		if time.Now().Unix() < int64(nbf) {
			resp.Active = false
			return resp, nil
		}
	}

	// Extract scope if present
	if scope, ok := claims["scope"].(string); ok {
		resp.Scope = scope
	} else if scopeList, ok := claims["scope"].([]interface{}); ok {
		scopes := make([]string, 0, len(scopeList))
		for _, s := range scopeList {
			if scopeStr, ok := s.(string); ok {
				scopes = append(scopes, scopeStr)
			}
		}
		resp.Scope = strings.Join(scopes, " ")
	}

	// Extract OIDC claims
	if email, ok := claims["email"].(string); ok {
		resp.Email = email
	}

	if emailVerified, ok := claims["email_verified"].(bool); ok {
		resp.EmailVerified = emailVerified
	}

	if name, ok := claims["name"].(string); ok {
		resp.Name = name
	}

	return resp, nil
}

// introspectRefreshToken introspects a refresh token.
func (h *TokenIntrospectionHandler) introspectRefreshToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	// Get refresh token entry
	entry, err := h.refreshTokenService.GetRefreshToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}

	// Check if revoked
	if entry.Revoked {
		return &IntrospectionResponse{Active: false}, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return &IntrospectionResponse{Active: false}, nil
	}

	// Build response
	resp := &IntrospectionResponse{
		Active:    true,
		TokenType: "refresh_token",
		Sub:       entry.Subject,
		Aud:       entry.Audience,
		ClientID:  entry.Audience,
		Exp:       entry.ExpiresAt.Unix(),
		Iat:       entry.IssuedAt.Unix(),
	}

	// Add scopes
	if len(entry.Scopes) > 0 {
		resp.Scope = strings.Join(entry.Scopes, " ")
	}

	// Add OIDC claims if available
	if entry.Email != "" {
		resp.Email = entry.Email
		resp.Username = entry.Email // Use email as username for refresh tokens
	}
	resp.EmailVerified = entry.EmailVerified

	if entry.Name != "" {
		resp.Name = entry.Name
	}

	// Get provider info for issuer
	if entry.ProviderID != "" {
		provider, err := h.providerRegistry.Get(entry.ProviderID)
		if err == nil && provider != nil {
			resp.Iss = provider.IssuerURL
		}
	}

	return resp, nil
}

// IntrospectTokenWithValidation introspects a token with full validation.
// This method performs signature verification and full token validation.
func (h *TokenIntrospectionHandler) IntrospectTokenWithValidation(ctx context.Context, req *IntrospectionRequest) (*IntrospectionResponse, error) {
	// Validate request
	if err := h.validateIntrospectionRequest(req); err != nil {
		return nil, err
	}

	// Try to introspect as refresh token first if hint suggests it
	if req.TokenTypeHint == "refresh_token" {
		if resp, err := h.introspectRefreshToken(ctx, req.Token); err == nil {
			return resp, nil
		}
	}

	// Try to validate as access token (ID token) with full validation
	if req.TokenTypeHint != "refresh_token" {
		// Validate the ID token (this includes signature verification)
		claims, err := h.idTokenService.ValidateIDToken(ctx, req.Token, req.ClientID)
		if err == nil {
			// Token is valid, build response from validated claims
			resp := &IntrospectionResponse{
				Active:    true,
				TokenType: "Bearer",
				Sub:       claims.Subject,
				Aud:       req.ClientID,
				ClientID:  req.ClientID,
				Iss:       claims.Issuer,
				Jti:       claims.ID,
				Username:  claims.Subject,
			}

			if claims.ExpiresAt != nil {
				resp.Exp = claims.ExpiresAt.Unix()
			}
			if claims.IssuedAt != nil {
				resp.Iat = claims.IssuedAt.Unix()
			}
			if claims.NotBefore != nil {
				resp.Nbf = claims.NotBefore.Unix()
			}

			// Add OIDC claims from validated token
			resp.Email = claims.Email
			resp.EmailVerified = claims.EmailVerified
			resp.Name = claims.Name

			return resp, nil
		}
	}

	// Try refresh token if hint allows
	if req.TokenTypeHint == "" || req.TokenTypeHint == "refresh_token" {
		if resp, err := h.introspectRefreshToken(ctx, req.Token); err == nil {
			return resp, nil
		}
	}

	// Token not valid
	return &IntrospectionResponse{Active: false}, nil
}

// BatchIntrospect introspects multiple tokens in a single operation.
func (h *TokenIntrospectionHandler) BatchIntrospect(ctx context.Context, tokens []string) ([]*IntrospectionResponse, error) {
	responses := make([]*IntrospectionResponse, len(tokens))

	for i, token := range tokens {
		req := &IntrospectionRequest{
			Token: token,
		}
		resp, err := h.IntrospectToken(ctx, req)
		if err != nil {
			// For batch operations, return inactive for errors
			responses[i] = &IntrospectionResponse{Active: false}
		} else {
			responses[i] = resp
		}
	}

	return responses, nil
}
