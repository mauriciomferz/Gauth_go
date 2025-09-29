//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package token

import (
	"context"
	"sync"
	"time"
)

// BlacklistedToken represents a revoked token
type BlacklistedToken struct {
	// ID is the token's unique identifier
	ID string

	// ExpiresAt is when the token would have expired
	ExpiresAt time.Time

	// RevokedAt is when the token was blacklisted
	RevokedAt time.Time

	// Reason explains why the token was revoked
	Reason string
}

// Blacklist manages revoked tokens
type Blacklist struct {
	mu      sync.RWMutex
	tokens  map[string]BlacklistedToken
	cleaner *time.Ticker
	done    chan struct{}
}

// NewBlacklist creates a new token blacklist with automatic cleanup
func NewBlacklist() *Blacklist {
	bl := &Blacklist{
		tokens: make(map[string]BlacklistedToken),
		done:   make(chan struct{}),
	}
	bl.startCleaner()
	return bl
}

// Add blacklists a token
func (bl *Blacklist) Add(_ context.Context, token *Token, reason string) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	bl.tokens[token.ID] = BlacklistedToken{
		ID:        token.ID,
		ExpiresAt: token.ExpiresAt,
		RevokedAt: time.Now(),
		Reason:    reason,
	}
	return nil
}

// IsBlacklisted checks if a token is revoked
func (bl *Blacklist) IsBlacklisted(_ context.Context, tokenID string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	_, exists := bl.tokens[tokenID]
	return exists
}

// GetBlacklistedToken retrieves blacklist details for a token
func (bl *Blacklist) GetBlacklistedToken(_ context.Context, tokenID string) (*BlacklistedToken, bool) {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	token, exists := bl.tokens[tokenID]
	if !exists {
		return nil, false
	}
	return &token, true
}

func (bl *Blacklist) startCleaner() {
	bl.cleaner = time.NewTicker(time.Hour)
	go func() {
		for {
			select {
			case <-bl.cleaner.C:
				bl.cleanup()
			case <-bl.done:
				bl.cleaner.Stop()
				return
			}
		}
	}()
}

func (bl *Blacklist) cleanup() {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	now := time.Now()
	for id, token := range bl.tokens {
		if token.ExpiresAt.Before(now) {
			delete(bl.tokens, id)
		}
	}
}

// Close stops the cleanup goroutine
func (bl *Blacklist) Close() error {
	close(bl.done)
	return nil
}

// Rotator handles token rotation
type Rotator struct {
	store     Store
	blacklist *Blacklist
	config    Config
}

// NewRotator creates a token rotator
func NewRotator(store Store, blacklist *Blacklist, config Config) *Rotator {
	return &Rotator{
		store:     store,
		blacklist: blacklist,
		config:    config,
	}
}

// RotateToken creates a new token and revokes the old one
func (r *Rotator) RotateToken(ctx context.Context, oldToken *Token) (*Token, error) {
	// Create new token
	newToken := &Token{
		ID:        NewID(), // Implement NewID() helper
		Type:      oldToken.Type,
		Subject:   oldToken.Subject,
		Issuer:    oldToken.Issuer,
		IssuedAt:  time.Now(),
		NotBefore: time.Now(),
		ExpiresAt: time.Now().Add(r.config.ValidityPeriod),
		Scopes:    oldToken.Scopes,
		Metadata:  oldToken.Metadata,
	}

	// Store new token
	if err := r.store.Save(ctx, newToken.ID, newToken); err != nil {
		return nil, err
	}

	// Blacklist old token
	if err := r.blacklist.Add(ctx, oldToken, "rotated"); err != nil {
		return nil, err
	}

	return newToken, nil
}
