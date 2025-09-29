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

// MemoryStore provides an in-memory token storage implementation
type MemoryStore struct {
	tokens    map[string]*Token
	mu        sync.RWMutex
	maxTokens int
}

// NewMemoryStore creates a new memory-based token store
// An optional TTL parameter can be provided for automatic token cleanup
func NewMemoryStore(ttl ...time.Duration) *MemoryStore {
	store := &MemoryStore{
		tokens: make(map[string]*Token),
	}

	// If TTL is provided, start a cleanup goroutine
	if len(ttl) > 0 && ttl[0] > 0 {
		cleanupInterval := ttl[0] / 10
		if cleanupInterval < time.Minute {
			cleanupInterval = time.Minute
		}

		go func() {
			ticker := time.NewTicker(cleanupInterval)
			defer ticker.Stop()

			for range ticker.C {
				_ = store.Cleanup(context.Background())
			}
		}()
	}

	return store
}

// Save stores a token with the given key
func (s *MemoryStore) Save(ctx context.Context, key string, token *Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxTokens > 0 && len(s.tokens) >= s.maxTokens {
		return ErrStorageFailure
	}

	s.tokens[key] = token
	return nil
}

// Get retrieves a token by key
func (s *MemoryStore) Get(ctx context.Context, key string) (*Token, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.tokens[key]
	if !exists {
		return nil, ErrTokenNotFound
	}

	if time.Now().After(token.ExpiresAt) {
		delete(s.tokens, key)
		return nil, ErrTokenExpired
	}

	// Return a copy to prevent modification of stored token
	return copyToken(token), nil
}

// Delete removes a token
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tokens[key]; !exists {
		return ErrTokenNotFound
	}

	delete(s.tokens, key)
	return nil
}

// List returns all tokens matching the filter
func (s *MemoryStore) List(ctx context.Context, filter Filter) ([]*Token, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*Token

	for _, token := range s.tokens {
		if !matchesFilter(token, filter) {
			continue
		}

		// Return copies to prevent modification of stored tokens
		matches = append(matches, copyToken(token))
	}

	return matches, nil
}

// Rotate replaces an existing token with a new one
func (s *MemoryStore) Rotate(ctx context.Context, old, new *Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tokens[old.ID]; !exists {
		return ErrTokenNotFound
	}

	// Save new token first
	s.tokens[new.ID] = new

	// Then delete old token
	delete(s.tokens, old.ID)

	return nil
}

// Revoke invalidates a token
func (s *MemoryStore) Revoke(ctx context.Context, token *Token) error {
	return s.Delete(ctx, token.ID)
}

// Validate checks if a token is valid
func (s *MemoryStore) Validate(ctx context.Context, token *Token) error {
	stored, err := s.Get(ctx, token.ID)
	if err != nil {
		return err
	}

	if stored.Value != token.Value {
		return ErrInvalidToken
	}

	return nil
}

// Refresh generates a new access token from a refresh token
func (s *MemoryStore) Refresh(ctx context.Context, refreshToken *Token) (*Token, error) {
	if err := s.Validate(ctx, refreshToken); err != nil {
		return nil, err
	}

	if refreshToken.Type != Refresh {
		return nil, ErrInvalidType
	}

	return nil, ErrInvalidConfig // Actual refresh should be handled by Service
}

// matchesFilter checks if a token matches the given filter criteria
func matchesFilter(token *Token, filter Filter) bool {
	return matchesTimeFilter(token, filter) &&
		matchesIdentityFilter(token, filter) &&
		matchesTypeFilter(token, filter) &&
		matchesScopeFilter(token, filter) &&
		matchesActiveFilter(token, filter)
}

func matchesTimeFilter(token *Token, filter Filter) bool {
	if !filter.ExpiresBefore.IsZero() && token.ExpiresAt.After(filter.ExpiresBefore) {
		return false
	}
	if !filter.ExpiresAfter.IsZero() && token.ExpiresAt.Before(filter.ExpiresAfter) {
		return false
	}
	if !filter.IssuedBefore.IsZero() && token.IssuedAt.After(filter.IssuedBefore) {
		return false
	}
	if !filter.IssuedAfter.IsZero() && token.IssuedAt.Before(filter.IssuedAfter) {
		return false
	}
	return true
}

func matchesIdentityFilter(token *Token, filter Filter) bool {
	if filter.Subject != "" && token.Subject != filter.Subject {
		return false
	}
	if filter.Issuer != "" && token.Issuer != filter.Issuer {
		return false
	}
	return true
}

func matchesTypeFilter(token *Token, filter Filter) bool {
	if len(filter.Types) == 0 {
		return true
	}
	for _, t := range filter.Types {
		if token.Type == t {
			return true
		}
	}
	return false
}

func matchesScopeFilter(token *Token, filter Filter) bool {
	if len(filter.Scopes) == 0 {
		return true
	}
	if filter.RequireAllScopes {
		return hasAllScopes(token.Scopes, filter.Scopes)
	}
	return hasAnyScope(token.Scopes, filter.Scopes)
}

func hasAllScopes(tokenScopes, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		found := false
		for _, scope := range tokenScopes {
			if scope == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func hasAnyScope(tokenScopes, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		for _, scope := range tokenScopes {
			if scope == required {
				return true
			}
		}
	}
	return false
}

func matchesActiveFilter(token *Token, filter Filter) bool {
	if !filter.Active {
		return true
	}
	now := time.Now()
	return now.Before(token.ExpiresAt) && now.After(token.NotBefore)
}

// Count returns the number of tokens matching the filter
func (s *MemoryStore) Count(ctx context.Context, filter Filter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	var count int64
	for _, token := range s.tokens {
		if matchesFilter(token, filter) {
			count++
		}
	}

	return count, nil
}

// Cleanup removes expired tokens
func (s *MemoryStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	now := time.Now()
	for key, token := range s.tokens {
		if token.ExpiresAt.Before(now) {
			delete(s.tokens, key)
		}
	}

	return nil
}

// Close releases resources used by the store
// For MemoryStore, this is a no-op as there are no resources to release
func (s *MemoryStore) Close() error {
	return nil
}

// Initialize is a no-op for MemoryStore (EnhancedStore compatibility)
func (s *MemoryStore) Initialize(ctx context.Context) error { return nil }

// Store stores a token (EnhancedStore compatibility)
func (s *MemoryStore) Store(ctx context.Context, token interface{}) error { return nil }

// Remove removes a token (EnhancedStore compatibility)
func (s *MemoryStore) Remove(ctx context.Context, key string) error { return nil }

// StoreAuthorizer is a no-op for MemoryStore (EnhancedStore compatibility)
func (s *MemoryStore) StoreAuthorizer(ctx context.Context, authorizer interface{}) error { return nil }

// StoreOwner is a no-op for MemoryStore (EnhancedStore compatibility)
func (s *MemoryStore) StoreOwner(ctx context.Context, owner interface{}) error { return nil }

// StoreToken is a no-op for MemoryStore (EnhancedStore compatibility)
func (s *MemoryStore) StoreToken(ctx context.Context, token interface{}) error { return nil }

// copyToken creates a deep copy of a token
func copyToken(t *Token) *Token {
	tokCopy := *t

	// Deep copy slices
	if t.Scopes != nil {
		tokCopy.Scopes = make([]string, len(t.Scopes))
		copy(tokCopy.Scopes, t.Scopes)
	}
	if t.Audience != nil {
		tokCopy.Audience = make([]string, len(t.Audience))
		copy(tokCopy.Audience, t.Audience)
	}

	// Deep copy Metadata struct if present
	if t.Metadata != nil {
		metaCopy := *t.Metadata
		// Deep copy maps inside Metadata
		if t.Metadata.AppData != nil {
			metaCopy.AppData = make(map[string]string, len(t.Metadata.AppData))
			for k, v := range t.Metadata.AppData {
				metaCopy.AppData[k] = v
			}
		}
		if t.Metadata.Labels != nil {
			metaCopy.Labels = make(map[string]string, len(t.Metadata.Labels))
			for k, v := range t.Metadata.Labels {
				metaCopy.Labels[k] = v
			}
		}
		if t.Metadata.Attributes != nil {
			metaCopy.Attributes = make(map[string][]string, len(t.Metadata.Attributes))
			for k, v := range t.Metadata.Attributes {
				arrCopy := make([]string, len(v))
				copy(arrCopy, v)
				metaCopy.Attributes[k] = arrCopy
			}
		}
		if t.Metadata.Tags != nil {
			metaCopy.Tags = make([]string, len(t.Metadata.Tags))
			copy(metaCopy.Tags, t.Metadata.Tags)
		}
		tokCopy.Metadata = &metaCopy
	}

	return &tokCopy
}
