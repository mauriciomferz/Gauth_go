// Package store provides token storage implementations for GAuth
package store

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Common storage errors
var (
	ErrTokenNotFound   = errors.New("token not found")
	ErrTokenExpired    = errors.New("token expired")
	ErrTokenRevoked    = errors.New("token revoked")
	ErrInvalidMetadata = errors.New("invalid token metadata")
)

// Config contains configuration for token stores
type Config struct {
	// CleanupInterval specifies how often cleanup of expired tokens runs
	CleanupInterval time.Duration

	// DefaultTTL is the default time-to-live for tokens that don't specify expiration
	DefaultTTL time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		CleanupInterval: 15 * time.Minute,
		DefaultTTL:      24 * time.Hour,
	}
}

// MemoryStore implements TokenStore using in-memory storage
// This is suitable for development and testing, but not for production environments
// where persistence and distribution are required.
type MemoryStore struct {
	tokens        map[string]TokenMetadata // token string -> metadata
	tokensByID    map[string]string        // token ID -> token string
	revokedTokens map[string]time.Time     // token string -> revocation time
	mu            sync.RWMutex
	config        Config
	stopCleanup   chan struct{}
}

// NewMemoryStore creates a new in-memory token store
func NewMemoryStore(cfg Config) (*MemoryStore, error) {
	store := &MemoryStore{
		tokens:        make(map[string]TokenMetadata),
		tokensByID:    make(map[string]string),
		revokedTokens: make(map[string]time.Time),
		config:        cfg,
		stopCleanup:   make(chan struct{}),
	}

	// Start background cleanup if interval > 0
	if cfg.CleanupInterval > 0 {
		go store.startCleanupRoutine()
	}

	return store, nil
}

// startCleanupRoutine periodically cleans up expired tokens
func (s *MemoryStore) startCleanupRoutine() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.Cleanup(context.Background())
		case <-s.stopCleanup:
			return
		}
	}
}

// Close stops background tasks
func (s *MemoryStore) Close() error {
	close(s.stopCleanup)
	return nil
}

// Store stores a token with its metadata
func (s *MemoryStore) Store(ctx context.Context, token string, metadata TokenMetadata) error {
	if token == "" || metadata.ID == "" {
		return &StorageError{
			Op:     "store",
			Key:    token,
			Err:    ErrInvalidMetadata,
			Detail: "token or ID is empty",
		}
	}

	// Set default expiration if not provided
	if metadata.ExpiresAt.IsZero() {
		metadata.ExpiresAt = time.Now().Add(s.config.DefaultTTL)
	}

	// Set default issuedAt if not provided
	if metadata.IssuedAt.IsZero() {
		metadata.IssuedAt = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store token with its metadata
	s.tokens[token] = metadata
	s.tokensByID[metadata.ID] = token

	return nil
}

// Get retrieves token metadata by token string
func (s *MemoryStore) Get(ctx context.Context, token string) (*TokenMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.tokens[token]
	if !exists {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    ErrTokenNotFound,
			Detail: "token not found in store",
		}
	}

	// Check if token is expired
	if !metadata.ExpiresAt.IsZero() && time.Now().After(metadata.ExpiresAt) {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    ErrTokenExpired,
			Detail: "token expired",
		}
	}

	// Check if token is revoked
	if _, revoked := s.revokedTokens[token]; revoked {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    ErrTokenRevoked,
			Detail: "token has been revoked",
		}
	}

	// Return a copy to prevent modification of internal state
	metadataCopy := metadata
	return &metadataCopy, nil
}

// GetByID retrieves token metadata by token ID
func (s *MemoryStore) GetByID(ctx context.Context, id string) (*TokenMetadata, error) {
	s.mu.RLock()
	token, exists := s.tokensByID[id]
	s.mu.RUnlock()

	if !exists {
		return nil, &StorageError{
			Op:     "getbyid",
			Key:    id,
			Err:    ErrTokenNotFound,
			Detail: "token ID not found",
		}
	}

	return s.Get(ctx, token)
}

// Delete removes a token from storage
func (s *MemoryStore) Delete(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, exists := s.tokens[token]
	if !exists {
		return &StorageError{
			Op:     "delete",
			Key:    token,
			Err:    ErrTokenNotFound,
			Detail: "cannot delete non-existent token",
		}
	}

	// Remove token from maps
	delete(s.tokens, token)
	delete(s.tokensByID, metadata.ID)
	delete(s.revokedTokens, token)

	return nil
}

// List returns all tokens for a subject
func (s *MemoryStore) List(ctx context.Context, subject string) ([]TokenMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tokens []TokenMetadata
	now := time.Now()

	for _, metadata := range s.tokens {
		// Skip if subject doesn't match
		if metadata.Subject != subject {
			continue
		}

		// Skip expired tokens
		if !metadata.ExpiresAt.IsZero() && now.After(metadata.ExpiresAt) {
			continue
		}

		// Skip revoked tokens
		if _, revoked := s.revokedTokens[s.tokensByID[metadata.ID]]; revoked {
			continue
		}

		tokens = append(tokens, metadata)
	}

	return tokens, nil
}

// Revoke marks a token as revoked
func (s *MemoryStore) Revoke(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if token exists
	if _, exists := s.tokens[token]; !exists {
		return &StorageError{
			Op:     "revoke",
			Key:    token,
			Err:    ErrTokenNotFound,
			Detail: "cannot revoke non-existent token",
		}
	}

	// Mark as revoked with current timestamp
	s.revokedTokens[token] = time.Now()

	return nil
}

// IsRevoked checks if a token is revoked
func (s *MemoryStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if token exists
	if _, exists := s.tokens[token]; !exists {
		return false, &StorageError{
			Op:     "isrevoked",
			Key:    token,
			Err:    ErrTokenNotFound,
			Detail: "cannot check revocation status of non-existent token",
		}
	}

	_, revoked := s.revokedTokens[token]
	return revoked, nil
}

// Cleanup removes expired tokens
func (s *MemoryStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expiredTokens := make([]string, 0)

	// Find expired tokens
	for tokenStr, metadata := range s.tokens {
		if !metadata.ExpiresAt.IsZero() && now.After(metadata.ExpiresAt) {
			expiredTokens = append(expiredTokens, tokenStr)
		}
	}

	// Remove expired tokens
	for _, tokenStr := range expiredTokens {
		metadata := s.tokens[tokenStr]
		delete(s.tokens, tokenStr)
		delete(s.tokensByID, metadata.ID)
		delete(s.revokedTokens, tokenStr)
	}

	return nil
}

// Unwrap returns the underlying error
func (e *StorageError) Unwrap() error {
	return e.Err
}
