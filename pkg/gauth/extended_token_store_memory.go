package gauth

import (
	"context"
	"sync"
	"time"
)

// MemoryExtendedTokenStore is an in-memory implementation of ExtendedTokenStore.
// This is suitable for testing and development. For production, use PostgresExtendedTokenStore.
type MemoryExtendedTokenStore struct {
	mu              sync.RWMutex
	tokens          map[string]*storedToken // accessToken -> token
	refreshTokens   map[string]*storedToken // refreshToken -> token
	revokedTokens   map[string]time.Time    // accessToken -> revoked time
	clientIndex     map[string][]string     // clientID -> []accessToken
}

// storedToken wraps ExtendedToken with metadata
type storedToken struct {
	Token    *ExtendedToken
	Metadata *TokenMetadata
}

// NewMemoryExtendedTokenStore creates a new in-memory token store
func NewMemoryExtendedTokenStore() *MemoryExtendedTokenStore {
	return &MemoryExtendedTokenStore{
		tokens:        make(map[string]*storedToken),
		refreshTokens: make(map[string]*storedToken),
		revokedTokens: make(map[string]time.Time),
		clientIndex:   make(map[string][]string),
	}
}

// SaveToken stores a new extended token
func (s *MemoryExtendedTokenStore) SaveToken(ctx context.Context, token *ExtendedToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if token already exists
	if _, exists := s.tokens[token.AccessToken]; exists {
		return ErrTokenAlreadyExists
	}

	stored := &storedToken{
		Token: token,
		Metadata: &TokenMetadata{
			CreatedAt:  time.Now(),
			RevokedAt:  nil,
			LastUsedAt: nil,
			UseCount:   0,
		},
	}

	// Store by access token
	s.tokens[token.AccessToken] = stored

	// Store by refresh token if present
	if token.RefreshToken != "" {
		s.refreshTokens[token.RefreshToken] = stored
	}

	// Index by client ID (extracted from authorization chain)
	clientID := extractClientID(token)
	if clientID != "" {
		s.clientIndex[clientID] = append(s.clientIndex[clientID], token.AccessToken)
	}

	return nil
}

// GetToken retrieves a token by its access token value
func (s *MemoryExtendedTokenStore) GetToken(ctx context.Context, accessToken string) (*ExtendedToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.tokens[accessToken]
	if !exists {
		return nil, ErrTokenNotFound
	}

	// Check if revoked
	if _, revoked := s.revokedTokens[accessToken]; revoked {
		return nil, ErrTokenRevoked
	}

	// Check if expired
	if isTokenExpired(stored.Token) {
		return nil, ErrTokenExpired
	}

	// Update last used time (note: this modifies metadata even in read-lock, acceptable for in-memory)
	now := time.Now()
	stored.Metadata.LastUsedAt = &now
	stored.Metadata.UseCount++

	return stored.Token, nil
}

// GetTokenByRefreshToken retrieves a token by its refresh token value
func (s *MemoryExtendedTokenStore) GetTokenByRefreshToken(ctx context.Context, refreshToken string) (*ExtendedToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.refreshTokens[refreshToken]
	if !exists {
		return nil, ErrTokenNotFound
	}

	// Check if revoked
	if _, revoked := s.revokedTokens[stored.Token.AccessToken]; revoked {
		return nil, ErrTokenRevoked
	}

	// Check if expired
	if isTokenExpired(stored.Token) {
		return nil, ErrTokenExpired
	}

	return stored.Token, nil
}

// RevokeToken marks a token as revoked
func (s *MemoryExtendedTokenStore) RevokeToken(ctx context.Context, accessToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, exists := s.tokens[accessToken]
	if !exists {
		// Per RFC 7009, revocation should succeed even if token doesn't exist (idempotent)
		return nil
	}

	// Mark as revoked
	now := time.Now()
	s.revokedTokens[accessToken] = now
	stored.Metadata.RevokedAt = &now

	return nil
}

// IsRevoked checks if a token has been revoked
func (s *MemoryExtendedTokenStore) IsRevoked(ctx context.Context, accessToken string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, revoked := s.revokedTokens[accessToken]
	return revoked, nil
}

// DeleteExpiredTokens removes expired tokens (cleanup operation)
func (s *MemoryExtendedTokenStore) DeleteExpiredTokens(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0

	// Find expired tokens
	expiredTokens := make([]string, 0)
	for accessToken, stored := range s.tokens {
		if isTokenExpired(stored.Token) {
			expiredTokens = append(expiredTokens, accessToken)
		}
	}

	// Delete expired tokens
	for _, accessToken := range expiredTokens {
		stored := s.tokens[accessToken]
		
		// Remove from main storage
		delete(s.tokens, accessToken)
		
		// Remove from refresh token index
		if stored.Token.RefreshToken != "" {
			delete(s.refreshTokens, stored.Token.RefreshToken)
		}
		
		// Remove from revoked index
		delete(s.revokedTokens, accessToken)
		
		// Remove from client index
		clientID := extractClientID(stored.Token)
		if clientID != "" {
			if tokens, exists := s.clientIndex[clientID]; exists {
				filtered := make([]string, 0, len(tokens))
				for _, t := range tokens {
					if t != accessToken {
						filtered = append(filtered, t)
					}
				}
				if len(filtered) > 0 {
					s.clientIndex[clientID] = filtered
				} else {
					delete(s.clientIndex, clientID)
				}
			}
		}
		
		count++
	}

	return count, nil
}

// ListTokensByClient returns all tokens for a specific client
func (s *MemoryExtendedTokenStore) ListTokensByClient(ctx context.Context, clientID string) ([]*ExtendedToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accessTokens, exists := s.clientIndex[clientID]
	if !exists {
		return []*ExtendedToken{}, nil
	}

	tokens := make([]*ExtendedToken, 0, len(accessTokens))
	for _, accessToken := range accessTokens {
		if stored, exists := s.tokens[accessToken]; exists {
			tokens = append(tokens, stored.Token)
		}
	}

	return tokens, nil
}

// ListTokensByResourceOwner returns all active tokens for a specific resource owner
func (s *MemoryExtendedTokenStore) ListTokensByResourceOwner(ctx context.Context, ownerID string) ([]*ExtendedToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ExtendedToken
	
	// Iterate through all tokens and filter by resource owner
	for accessToken, stored := range s.tokens {
		// Skip revoked tokens
		if _, revoked := s.revokedTokens[accessToken]; revoked {
			continue
		}
		
		// Skip expired tokens
		if isTokenExpired(stored.Token) {
			continue
		}
		
		// Check if this token belongs to the resource owner
		if stored.Token.ResourceOwner != nil && stored.Token.ResourceOwner.OwnerID == ownerID {
			result = append(result, stored.Token)
		}
	}
	
	return result, nil
}

// RevokeTokenWithReason marks a token as revoked with a specific reason
func (s *MemoryExtendedTokenStore) RevokeTokenWithReason(ctx context.Context, accessToken string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if token exists
	stored, exists := s.tokens[accessToken]
	if !exists {
		return ErrTokenNotFound
	}

	// Mark as revoked
	now := time.Now()
	s.revokedTokens[accessToken] = now
	stored.Metadata.RevokedAt = &now

	// Add audit entry to token with revocation reason
	if stored.Token.AuditTrail == nil {
		stored.Token.AuditTrail = []AuditEntry{}
	}
	
	stored.Token.AuditTrail = append(stored.Token.AuditTrail, AuditEntry{
		Timestamp: now,
		Action:    "token_revoked",
		Actor:     "resource_owner", // or extract from context
		Result:    "success",
		Details: map[string]interface{}{
			"reason":       reason,
			"access_token": accessToken,
		},
	})

	return nil
}

// GetStats returns statistics about stored tokens (useful for monitoring)
func (s *MemoryExtendedTokenStore) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"total_tokens":   len(s.tokens),
		"revoked_tokens": len(s.revokedTokens),
		"total_clients":  len(s.clientIndex),
	}
}

// Helper functions

// extractClientID extracts the client ID from the authorization chain
func extractClientID(token *ExtendedToken) string {
	if token.AuthorizationChain != nil && token.AuthorizationChain.Client != nil {
		return token.AuthorizationChain.Client.EntityID
	}
	return ""
}

// isTokenExpired checks if a token is expired based on IssuedAt + ExpiresIn
func isTokenExpired(token *ExtendedToken) bool {
	expiresAt := token.IssuedAt.Unix() + token.ExpiresIn
	return time.Now().Unix() > expiresAt
}
