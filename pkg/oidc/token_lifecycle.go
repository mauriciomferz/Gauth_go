package oidc

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenRevocationService manages token revocation.
type TokenRevocationService struct {
	revokedTokens map[string]*RevokedTokenEntry
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// RevokedTokenEntry represents a revoked token.
type RevokedTokenEntry struct {
	TokenID    string
	RevokedAt  time.Time
	Reason     string
	RevokedBy  string
	ExpiresAt  time.Time // When this revocation entry can be removed
}

// NewTokenRevocationService creates a new token revocation service.
func NewTokenRevocationService() *TokenRevocationService {
	service := &TokenRevocationService{
		revokedTokens: make(map[string]*RevokedTokenEntry),
		stopCleanup:   make(chan struct{}),
	}

	// Start cleanup goroutine (runs every hour)
	service.cleanupTicker = time.NewTicker(time.Hour)
	go service.cleanupLoop()

	return service
}

// RevokeToken revokes a token.
func (s *TokenRevocationService) RevokeToken(ctx context.Context, tokenID string, reason string, revokedBy string, expiresAt time.Time) error {
	if tokenID == "" {
		return fmt.Errorf("token ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already revoked
	if _, exists := s.revokedTokens[tokenID]; exists {
		return fmt.Errorf("token already revoked")
	}

	// Add to revocation list
	s.revokedTokens[tokenID] = &RevokedTokenEntry{
		TokenID:   tokenID,
		RevokedAt: time.Now(),
		Reason:    reason,
		RevokedBy: revokedBy,
		ExpiresAt: expiresAt,
	}

	return nil
}

// IsRevoked checks if a token is revoked.
func (s *TokenRevocationService) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	if tokenID == "" {
		return false, fmt.Errorf("token ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, revoked := s.revokedTokens[tokenID]
	return revoked, nil
}

// GetRevocationInfo returns revocation information for a token.
func (s *TokenRevocationService) GetRevocationInfo(ctx context.Context, tokenID string) (*RevokedTokenEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.revokedTokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("token not revoked")
	}

	return entry, nil
}

// RevokeTokensBatch revokes multiple tokens in one operation.
func (s *TokenRevocationService) RevokeTokensBatch(ctx context.Context, tokenIDs []string, reason string, revokedBy string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tokenID := range tokenIDs {
		if tokenID == "" {
			continue
		}

		// Skip if already revoked
		if _, exists := s.revokedTokens[tokenID]; exists {
			continue
		}

		s.revokedTokens[tokenID] = &RevokedTokenEntry{
			TokenID:   tokenID,
			RevokedAt: time.Now(),
			Reason:    reason,
			RevokedBy: revokedBy,
			ExpiresAt: expiresAt,
		}
	}

	return nil
}

// GetRevokedCount returns the number of revoked tokens.
func (s *TokenRevocationService) GetRevokedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.revokedTokens)
}

// CleanupExpired removes expired revocation entries.
func (s *TokenRevocationService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0

	for tokenID, entry := range s.revokedTokens {
		if now.After(entry.ExpiresAt) {
			delete(s.revokedTokens, tokenID)
			removed++
		}
	}

	return removed
}

// cleanupLoop runs periodic cleanup of expired entries.
func (s *TokenRevocationService) cleanupLoop() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.CleanupExpired()
		case <-s.stopCleanup:
			s.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (s *TokenRevocationService) Stop() {
	close(s.stopCleanup)
}

// RefreshTokenService manages refresh tokens for exchanged tokens.
type RefreshTokenService struct {
	refreshTokens map[string]*RefreshTokenEntry
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// RefreshTokenEntry represents a stored refresh token.
type RefreshTokenEntry struct {
	RefreshToken string
	ProviderID   string
	Subject      string
	Audience     string
	IssuedAt     time.Time
	ExpiresAt    time.Time
	LastUsed     time.Time
	UseCount     int
}

// NewRefreshTokenService creates a new refresh token service.
func NewRefreshTokenService() *RefreshTokenService {
	service := &RefreshTokenService{
		refreshTokens: make(map[string]*RefreshTokenEntry),
		stopCleanup:   make(chan struct{}),
	}

	// Start cleanup goroutine (runs every hour)
	service.cleanupTicker = time.NewTicker(time.Hour)
	go service.cleanupLoop()

	return service
}

// StoreRefreshToken stores a refresh token.
func (s *RefreshTokenService) StoreRefreshToken(ctx context.Context, tokenID string, entry *RefreshTokenEntry) error {
	if tokenID == "" {
		return fmt.Errorf("token ID is required")
	}

	if entry == nil {
		return fmt.Errorf("refresh token entry is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[tokenID] = entry
	return nil
}

// GetRefreshToken retrieves a refresh token.
func (s *RefreshTokenService) GetRefreshToken(ctx context.Context, tokenID string) (*RefreshTokenEntry, error) {
	if tokenID == "" {
		return nil, fmt.Errorf("token ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.refreshTokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("refresh token not found")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	return entry, nil
}

// UpdateRefreshTokenUsage updates the last used time and use count.
func (s *RefreshTokenService) UpdateRefreshTokenUsage(ctx context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.refreshTokens[tokenID]
	if !exists {
		return fmt.Errorf("refresh token not found")
	}

	entry.LastUsed = time.Now()
	entry.UseCount++

	return nil
}

// RevokeRefreshToken removes a refresh token.
func (s *RefreshTokenService) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.refreshTokens, tokenID)
	return nil
}

// CleanupExpired removes expired refresh tokens.
func (s *RefreshTokenService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0

	for tokenID, entry := range s.refreshTokens {
		if now.After(entry.ExpiresAt) {
			delete(s.refreshTokens, tokenID)
			removed++
		}
	}

	return removed
}

// cleanupLoop runs periodic cleanup of expired entries.
func (s *RefreshTokenService) cleanupLoop() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.CleanupExpired()
		case <-s.stopCleanup:
			s.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (s *RefreshTokenService) Stop() {
	close(s.stopCleanup)
}

// GetRefreshTokenCount returns the number of stored refresh tokens.
func (s *RefreshTokenService) GetRefreshTokenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.refreshTokens)
}

// TokenIntrospectionService provides RFC 7662 token introspection.
type TokenIntrospectionService struct {
	revocationService *TokenRevocationService
	idTokenService    *IDTokenService
}

// TokenIntrospectionRequest represents an introspection request (RFC 7662).
type TokenIntrospectionRequest struct {
	Token         string
	TokenTypeHint string // "access_token" or "refresh_token"
}

// TokenIntrospectionResponse represents an introspection response (RFC 7662).
type TokenIntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Username  string   `json:"username,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
	Exp       int64    `json:"exp,omitempty"`
	Iat       int64    `json:"iat,omitempty"`
	Nbf       int64    `json:"nbf,omitempty"`
	Sub       string   `json:"sub,omitempty"`
	Aud       []string `json:"aud,omitempty"`
	Iss       string   `json:"iss,omitempty"`
	Jti       string   `json:"jti,omitempty"`
}

// NewTokenIntrospectionService creates a new token introspection service.
func NewTokenIntrospectionService(revocationService *TokenRevocationService, idTokenService *IDTokenService) *TokenIntrospectionService {
	return &TokenIntrospectionService{
		revocationService: revocationService,
		idTokenService:    idTokenService,
	}
}

// IntrospectToken introspects a token (RFC 7662).
func (s *TokenIntrospectionService) IntrospectToken(ctx context.Context, req TokenIntrospectionRequest) (*TokenIntrospectionResponse, error) {
	if req.Token == "" {
		return &TokenIntrospectionResponse{Active: false}, nil
	}

	// Parse the token without validation first to extract audience
	// In a real implementation, we'd use a more sophisticated approach
	// For now, we'll accept any audience for introspection
	// The caller is responsible for verifying they have access to introspect this token
	
	// Try to validate with empty audience (will check other claims)
	claims, err := s.idTokenService.ValidateIDToken(ctx, req.Token, "")
	if err != nil {
		// Invalid token - could be expired, wrong signature, etc.
		return &TokenIntrospectionResponse{Active: false}, nil
	}

	// Check if token is revoked
	tokenID := claims.ID
	if tokenID != "" {
		revoked, err := s.revocationService.IsRevoked(ctx, tokenID)
		if err == nil && revoked {
			return &TokenIntrospectionResponse{Active: false}, nil
		}
	}

	// Check expiration
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return &TokenIntrospectionResponse{Active: false}, nil
	}

	// Check not before
	if claims.NotBefore != nil && time.Now().Before(claims.NotBefore.Time) {
		return &TokenIntrospectionResponse{Active: false}, nil
	}

	// Token is active, return details
	response := &TokenIntrospectionResponse{
		Active:    true,
		Sub:       claims.Subject,
		Iss:       claims.Issuer,
		TokenType: "Bearer",
		Username:  claims.PreferredUsername,
	}

	if claims.ExpiresAt != nil {
		response.Exp = claims.ExpiresAt.Unix()
	}

	if claims.IssuedAt != nil {
		response.Iat = claims.IssuedAt.Unix()
	}

	if claims.NotBefore != nil {
		response.Nbf = claims.NotBefore.Unix()
	}

	if len(claims.Audience) > 0 {
		response.Aud = claims.Audience
		response.ClientID = claims.Audience[0]
	}

	if claims.ID != "" {
		response.Jti = claims.ID
	}

	return response, nil
}
