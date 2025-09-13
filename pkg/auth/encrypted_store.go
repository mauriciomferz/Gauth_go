package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// EncryptedStoreConfig contains configuration for the encrypted token store
type EncryptedStoreConfig struct {
	// EncryptionKey is the 32-byte key used for AES-256 encryption
	EncryptionKey []byte
	// BackingStore is the underlying store for encrypted tokens
	BackingStore TokenStore
	// TokenTTL is the time-to-live for stored tokens
	TokenTTL time.Duration
}

// encryptedTokenStore implements TokenStore with encryption
type encryptedTokenStore struct {
	config EncryptedStoreConfig
	gcm    cipher.AEAD
	cache  sync.Map
}

// NewEncryptedTokenStore creates a new token store with encryption
func NewEncryptedTokenStore(config EncryptedStoreConfig) (TokenStore, error) {
	if len(config.EncryptionKey) != 32 {
		return nil, errors.New("encryption key must be 32 bytes for AES-256")
	}

	if config.BackingStore == nil {
		return nil, errors.New("backing store is required")
	}

	block, err := aes.NewCipher(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &encryptedTokenStore{
		config: config,
		gcm:    gcm,
	}, nil
}

func (s *encryptedTokenStore) Initialize(ctx context.Context) error {
	return s.config.BackingStore.Initialize(ctx)
}

func (s *encryptedTokenStore) Close() error {
	return s.config.BackingStore.Close()
}

func (s *encryptedTokenStore) encrypt(data []byte) (string, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := s.gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (s *encryptedTokenStore) decrypt(encryptedStr string) ([]byte, error) {
	encrypted, err := base64.RawURLEncoding.DecodeString(encryptedStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(encrypted) < s.gcm.NonceSize() {
		return nil, errors.New("encrypted data too short")
	}

	nonce := encrypted[:s.gcm.NonceSize()]
	ciphertext := encrypted[s.gcm.NonceSize():]

	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *encryptedTokenStore) Store(ctx context.Context, token *TokenResponse) error {
	data := struct {
		Token     *TokenResponse
		ExpiresAt time.Time
	}{
		Token:     token,
		ExpiresAt: time.Now().Add(s.config.TokenTTL),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	encrypted, err := s.encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt token data: %w", err)
	}

	if err := s.config.BackingStore.Store(ctx, &TokenResponse{
		Token: encrypted,
	}); err != nil {
		return fmt.Errorf("failed to store encrypted token: %w", err)
	}

	// Cache the token data
	s.cache.Store(token.Token, data)
	return nil
}

func (s *encryptedTokenStore) Get(ctx context.Context, tokenStr string) (*TokenData, error) {
	// Check cache first
	if cached, ok := s.cache.Load(tokenStr); ok {
		data := cached.(struct {
			Token     *TokenResponse
			ExpiresAt time.Time
		})
		if time.Now().After(data.ExpiresAt) {
			s.cache.Delete(tokenStr)
			return nil, ErrTokenExpired
		}
		return &TokenData{
			Valid:     true,
			Subject:   data.Token.Claims["sub"].(string),
			Scope:     data.Token.Scope,
			ExpiresAt: data.ExpiresAt,
		}, nil
	}

	// Get from backing store
	encryptedToken, err := s.config.BackingStore.Get(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	decrypted, err := s.decrypt(encryptedToken.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token data: %w", err)
	}

	var data struct {
		Token     *TokenResponse
		ExpiresAt time.Time
	}
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	if time.Now().After(data.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Cache the decrypted data
	s.cache.Store(tokenStr, data)

	return &TokenData{
		Valid:     true,
		Subject:   data.Token.Claims["sub"].(string),
		Scope:     data.Token.Scope,
		ExpiresAt: data.ExpiresAt,
	}, nil
}

func (s *encryptedTokenStore) Remove(ctx context.Context, tokenStr string) error {
	s.cache.Delete(tokenStr)
	return s.config.BackingStore.Remove(ctx, tokenStr)
}

func (s *encryptedTokenStore) Cleanup(ctx context.Context) error {
	// Clear cache
	s.cache.Range(func(key, value interface{}) bool {
		data := value.(struct {
			Token     *TokenResponse
			ExpiresAt time.Time
		})
		if time.Now().After(data.ExpiresAt) {
			s.cache.Delete(key)
		}
		return true
	})

	return s.config.BackingStore.Cleanup(ctx)
}
