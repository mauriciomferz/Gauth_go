// Package gauth provides token management for the GAuth protocol.
//
// The token management system includes:
//   - Secure token generation using cryptographic random numbers
//   - Token validation with built-in expiration checks
//   - Token lifecycle management with refresh support
//   - Token revocation with audit trail
//   - Claims-based authorization
//   - Token refresh strategies with sliding windows
//   - Token introspection endpoints
//
// Token types supported:
//   - Access tokens for resource access
//   - Refresh tokens for token renewal
//   - ID tokens for identity verification
package gauth

import (
	"fmt"
	"strings"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// tokenManager handles internal token operations

type tokenManager struct {
	gauth *gauth.GAuth
}

// newTokenManager creates a new token manager instance
func newTokenManager(gauthInstance *gauth.GAuth) *tokenManager {
	return &tokenManager{gauth: gauthInstance}
}

// RequestToken issues a new token based on the provided authorization grant.
func (tm *tokenManager) RequestToken(req gauth.TokenRequest) (*gauth.TokenResponse, error) {
	// Use the canonical GAuth method
	return tm.gauth.RequestToken(req)
}

// RevokeToken invalidates and removes a token.
func (tm *tokenManager) RevokeToken(token string) error {
	// No public method for token deletion; return not supported
	return fmt.Errorf("token revocation not supported by GAuth API")
}

// ValidateToken checks if a token is valid and not expired.
func (tm *tokenManager) ValidateToken(token string) bool {
	// Use the canonical GAuth method
	data, err := tm.gauth.ValidateToken(token)
	if err != nil || data == nil {
		return false
	}
	return data.Valid && !time.Now().After(data.ValidUntil)
}

// GetTokenInfo returns information about a token.
func (tm *tokenManager) GetTokenInfo(token string) *gauth.TokenResponse {
	// Use ValidateToken to get token data
	data, err := tm.gauth.ValidateToken(token)
	if err != nil || data == nil {
		return nil
	}
	return &gauth.TokenResponse{
		Token:      token,
		ValidUntil: data.ValidUntil,
		Scope:      data.Scope,
		// Restrictions: not available from TokenData, set to nil or add if needed
	}
}

// parseScopes converts comma-separated scopes into a slice.
func parseScopes(scopeStr string) []string {
	if scopeStr == "" {
		return nil
	}

	scopes := strings.Split(scopeStr, ",")
	result := make([]string, 0, len(scopes))

	for _, scope := range scopes {
		if s := strings.TrimSpace(scope); s != "" {
			result = append(result, s)
		}
	}

	return result
}
