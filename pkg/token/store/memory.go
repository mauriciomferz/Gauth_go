package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/errors"
)

// DEPRECATED: Use github.com/Gimel-Foundation/gauth/pkg/token.MemoryStore instead
//
// This implementation is kept for backward compatibility with older code.
// New code should use the MemoryStore from the token package directly.

// MemoryStore implements Store using in-memory storage
type MemoryStore struct {
	sync.RWMutex
	tokens    map[string]encryptedMetadata
	config    Config
	gcm       cipher.AEAD
	stopClean chan struct{}
}

type encryptedMetadata struct {
	Data      []byte
	ExpiresAt time.Time
}

// NewMemoryStore creates a new in-memory token store
func NewMemoryStore(config Config) (*MemoryStore, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	store := &MemoryStore{
		tokens:    make(map[string]encryptedMetadata),
		config:    config,
		gcm:       gcm,
		stopClean: make(chan struct{}),
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store, nil
}

// Save implements Store.Save
func (s *MemoryStore) Save(_ context.Context, token string, metadata TokenMetadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	// Check capacity
	s.RLock()
	if int64(len(s.tokens)) >= s.config.MaxTokens {
		s.RUnlock()
		return errors.New(errors.ErrStoreFull, "token store capacity reached")
	}
	s.RUnlock()

	// Encrypt metadata
	data, err := s.encrypt(metadata)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	s.Lock()
	s.tokens[token] = encryptedMetadata{
		Data:      data,
		ExpiresAt: metadata.ExpiresAt,
	}
	s.Unlock()

	return nil
}

// Get implements Store.Get
func (s *MemoryStore) Get(_ context.Context, token string) (*TokenMetadata, error) {
	s.RLock()
	encrypted, ok := s.tokens[token]
	s.RUnlock()

	if !ok {
		return nil, errors.New(errors.ErrTokenNotFound, "token not found in store")
	}

	if time.Now().After(encrypted.ExpiresAt) {
		return nil, errors.New(errors.ErrTokenExpired, "token has expired")
	}

	metadata, err := s.decrypt(encrypted.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return metadata, nil
}

// Delete implements Store.Delete
func (s *MemoryStore) Delete(_ context.Context, token string) error {
	s.Lock()
	delete(s.tokens, token)
	s.Unlock()
	return nil
}

// Revoke implements Store.Revoke
func (s *MemoryStore) Revoke(ctx context.Context, token string, revokedBy string) error {
	metadata, err := s.Get(ctx, token)
	if err != nil {
		return err
	}

	metadata.IsRevoked = true
	metadata.RevokedAt = time.Now()
	metadata.RevokedBy = revokedBy

	return s.Save(ctx, token, *metadata)
}

// IsRevoked implements Store.IsRevoked
func (s *MemoryStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	metadata, err := s.Get(ctx, token)
	if err != nil {
		return false, err
	}
	return metadata.IsRevoked, nil
}

// UpdateLastUsed implements Store.UpdateLastUsed
func (s *MemoryStore) UpdateLastUsed(ctx context.Context, token string) error {
	metadata, err := s.Get(ctx, token)
	if err != nil {
		return err
	}

	metadata.LastUsed = time.Now()
	metadata.UseCount++

	return s.Save(ctx, token, *metadata)
}

// ListExpired implements Store.ListExpired
func (s *MemoryStore) ListExpired(_ context.Context) ([]string, error) {
	var expired []string
	now := time.Now()

	s.RLock()
	for token, meta := range s.tokens {
		if now.After(meta.ExpiresAt) {
			expired = append(expired, token)
		}
	}
	s.RUnlock()

	return expired, nil
}

// Cleanup implements Store.Cleanup
func (s *MemoryStore) Cleanup(ctx context.Context) error {
	expired, err := s.ListExpired(ctx)
	if err != nil {
		return err
	}

	s.Lock()
	for _, token := range expired {
		delete(s.tokens, token)
	}
	s.Unlock()

	return nil
}

// Close implements Store.Close
func (s *MemoryStore) Close() error {
	close(s.stopClean)
	return nil
}

func (s *MemoryStore) cleanupLoop() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.Cleanup(context.Background()); err != nil {
				// Log cleanup error to prevent silent failures
				fmt.Printf("Token cleanup error: %v\n", err)
			}
		case <-s.stopClean:
			return
		}
	}
}

func (s *MemoryStore) encrypt(metadata TokenMetadata) ([]byte, error) {
	// Create nonce
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encode metadata
	data, err := encode(metadata)
	if err != nil {
		return nil, err
	}

	// Encrypt
	encrypted := s.gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

func (s *MemoryStore) decrypt(data []byte) (*TokenMetadata, error) {
	// Extract nonce
	if len(data) < s.gcm.NonceSize() {
		return nil, errors.New(errors.ErrInvalidData, "encrypted data too short")
	}
	nonce := data[:s.gcm.NonceSize()]
	ciphertext := data[s.gcm.NonceSize():]

	// Decrypt
	decrypted, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Decode metadata
	metadata, err := decode(decrypted)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// encode serializes token metadata to bytes (implementation omitted)
func encode(_ TokenMetadata) ([]byte, error) {
	// Implementation would use encoding/gob or similar
	return nil, nil
}

// decode deserializes token metadata from bytes (implementation omitted)
func decode(_ []byte) (*TokenMetadata, error) {
	// Implementation would use encoding/gob or similar
	return nil, nil
}
