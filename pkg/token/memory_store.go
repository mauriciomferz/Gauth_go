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
	"errors"
	"sync"
	"time"
)

// memoryStoreV1 implements Store using in-memory storage with TTL support
// DEPRECATED: Use MemoryStore from memory.go instead
type memoryStoreV1 struct {
	mu      sync.RWMutex
	tokens  map[string]tokenEntry
	ttl     time.Duration
	cleaner *time.Ticker
	done    chan struct{}
}

type tokenEntry struct {
	token     *Token
	createdAt time.Time
}

// NewMemoryStoreV1 creates a new in-memory token store with TTL-based cleanup
// DEPRECATED: Use NewMemoryStore() from memory.go instead
func NewMemoryStoreV1(ttl time.Duration) Store {
	store := &memoryStoreV1{
		tokens: make(map[string]tokenEntry),
		ttl:    ttl,
		done:   make(chan struct{}),
	}
	store.startCleaner()
	return store
}

// Save stores a token with the given key (Store interface compatibility)
func (s *memoryStoreV1) Save(ctx context.Context, key string, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[key] = tokenEntry{
		token:     token,
		createdAt: time.Now(),
	}
	return nil
}

func (s *memoryStoreV1) Get(ctx context.Context, key string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.tokens[key]
	if !exists {
		return nil, ErrTokenNotFound
	}

	// Check if token has expired
	if time.Since(entry.createdAt) > s.ttl {
		return nil, ErrTokenExpired
	}

	return entry.token, nil
}

func (s *memoryStoreV1) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, key)
	return nil
}

func (s *memoryStoreV1) List(ctx context.Context, filter Filter) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Token
	now := time.Now()

	for _, entry := range s.tokens {
		if s.matchesFilter(entry.token, filter) && now.Before(entry.token.ExpiresAt) {
			result = append(result, entry.token)
		}
	}

	return result, nil
}

func (s *memoryStoreV1) matchesFilter(token *Token, filter Filter) bool {
	// Check token type
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if token.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check expiration
	if !filter.ExpiresAfter.IsZero() && token.ExpiresAt.Before(filter.ExpiresAfter) {
		return false
	}

	// Check scopes
	if len(filter.Scopes) > 0 {
		for _, requiredScope := range filter.Scopes {
			found := false
			for _, tokenScope := range token.Scopes {
				if tokenScope == requiredScope {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// startCleaner begins the background cleanup process
func (s *memoryStoreV1) startCleaner() {
	if s.ttl <= 0 {
		return // No cleanup needed for unlimited TTL
	}

	s.cleaner = time.NewTicker(s.ttl / 2)

	go func() {
		for {
			select {
			case <-s.cleaner.C:
				s.cleanup()
			case <-s.done:
				s.cleaner.Stop()
				return
			}
		}
	}()
}

func (s *memoryStoreV1) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := time.Now().Add(-s.ttl)
	for key, entry := range s.tokens {
		if entry.createdAt.Before(threshold) {
			delete(s.tokens, key)
		}
	}
}

// Close stops the cleanup goroutine
func (s *memoryStoreV1) Close() error {
	if s.cleaner != nil {
		s.cleaner.Stop()
	}
	close(s.done)
	return nil
}

// Count returns the number of tokens matching the filter
func (s *memoryStoreV1) Count(ctx context.Context, filter Filter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	var count int64
	for _, token := range s.tokens {
		if s.matchesFilter(token.token, filter) {
			count++
		}
	}

	return count, nil
}

// Cleanup removes expired tokens
func (s *memoryStoreV1) Cleanup(ctx context.Context) error {
	// This function is intentionally empty because cleanup is handled
	// automatically by the cleanup goroutine started in NewMemoryStoreV1
	return nil
}

// Revoke marks a token as revoked
func (s *memoryStoreV1) Revoke(ctx context.Context, token *Token) error {
	return s.Delete(ctx, token.ID)
}

// Validate checks if a token is valid and active
func (s *memoryStoreV1) Validate(ctx context.Context, token *Token) error {
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
func (s *memoryStoreV1) Refresh(ctx context.Context, refreshToken *Token) (*Token, error) {
	// This is a deprecated implementation
	return nil, errors.New("not implemented")
}

// Rotate replaces an existing token with a new one
func (s *memoryStoreV1) Rotate(ctx context.Context, old, new *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete old token
	delete(s.tokens, old.ID)

	// Store new token
	s.tokens[new.ID] = tokenEntry{
		token:     new,
		createdAt: time.Now(),
	}

	return nil
}
