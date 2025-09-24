package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// EncryptedStoreConfig contains configuration for the encrypted token store
type EncryptedStoreConfig struct {
	// EncryptionKey is the 32-byte key used for AES-256 encryption
	EncryptionKey []byte
	// BackingStore is the underlying store for encrypted tokens
	BackingStore token.EnhancedStore
	// TokenTTL is the time-to-live for stored tokens
	TokenTTL time.Duration
}

// encryptedTokenStore implements token.EnhancedStore with encryption
type encryptedTokenStore struct {
	config EncryptedStoreConfig
	gcm    cipher.AEAD
	cache  sync.Map
}

// NewEncryptedTokenStore creates a new token store with encryption
func NewEncryptedTokenStore(config EncryptedStoreConfig) (token.EnhancedStore, error) {
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

// Initialize and Close are not required by EnhancedStore, so stub them out.
func (s *encryptedTokenStore) Initialize(ctx context.Context) error {
	return nil
}

func (s *encryptedTokenStore) Close() error {
	return nil
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

// Store implements EnhancedStore interface (accepts interface{})
func (s *encryptedTokenStore) Store(ctx context.Context, token interface{}) error {
	// Stub: not implemented
	return errors.New("not implemented")
}

// Get retrieves a token by key (not implemented)
func (s *encryptedTokenStore) Get(ctx context.Context, key string) (*token.Token, error) {
	return nil, errors.New("not implemented")
}

func (s *encryptedTokenStore) Remove(ctx context.Context, tokenStr string) error {
	// Stub: not implemented
	return errors.New("not implemented")
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

// Count returns the number of tokens matching the filter (not implemented)
func (s *encryptedTokenStore) Count(ctx context.Context, filter token.Filter) (int64, error) {
	return 0, errors.New("not implemented")
}

// Delete removes a token (not implemented)
func (s *encryptedTokenStore) Delete(ctx context.Context, key string) error {
	return errors.New("not implemented")
}

// List returns all tokens matching the filter (not implemented)
func (s *encryptedTokenStore) List(ctx context.Context, filter token.Filter) ([]*token.Token, error) {
	return nil, errors.New("not implemented")
}

// Refresh generates a new access token from a refresh token (not implemented)
func (s *encryptedTokenStore) Refresh(ctx context.Context, refreshToken *token.Token) (*token.Token, error) {
	return nil, errors.New("not implemented")
}

// Revoke invalidates a token before its natural expiration (not implemented)
func (s *encryptedTokenStore) Revoke(ctx context.Context, token *token.Token) error {
	return errors.New("not implemented")
}

// Rotate replaces an existing token with a new one (not implemented)
func (s *encryptedTokenStore) Rotate(ctx context.Context, old, new *token.Token) error {
	return errors.New("not implemented")
}

// Save stores a token with the given key (not implemented)
func (s *encryptedTokenStore) Save(ctx context.Context, key string, token *token.Token) error {
	return errors.New("not implemented")
}

// TrackVersionHistory is not implemented for encryptedTokenStore
func (s *encryptedTokenStore) TrackVersionHistory(ctx context.Context, token *token.EnhancedToken) error {
	return errors.New("not implemented")
}

// Validate checks if a token is valid and active (not implemented)
func (s *encryptedTokenStore) Validate(ctx context.Context, token *token.Token) error {
	return errors.New("not implemented")
}

// ValidateAuthorization is not implemented for encryptedTokenStore
func (s *encryptedTokenStore) ValidateAuthorization(ctx context.Context, token *token.EnhancedToken) error {
	return errors.New("not implemented")
}

// VerifyAttestation is not implemented for encryptedTokenStore
func (s *encryptedTokenStore) VerifyAttestation(ctx context.Context, attestation *token.Attestation) error {
	return errors.New("not implemented")
}

func (s *encryptedTokenStore) GetHumanVerification(ctx context.Context, token *token.EnhancedToken) (*common.HumanVerification, error) {
	return &common.HumanVerification{
		UltimateHumanID:          "stub-human",
		Role:                     "stub-role",
		LegalCapacityVerified:    true,
		CapacityVerificationTime: time.Now(),
		CapacityVerifier:         "stub-verifier",
		DelegationChain:          []common.DelegationLink{},
	}, nil
}

func (s *encryptedTokenStore) GetSecondLevelApproval(ctx context.Context, token *token.EnhancedToken) (*common.SecondLevelApproval, error) {
	return &common.SecondLevelApproval{
		PrimaryApprover:       "stub-primary",
		PrimaryApprovalTime:   time.Now(),
		PrimaryRole:           "stub-role",
		SecondaryApprover:     "stub-secondary",
		SecondaryApprovalTime: time.Now(),
		SecondaryRole:         "stub-role",
		ApprovalLevel:         1,
		ApprovalScope:         []string{"stub-scope"},
		ApprovalDuration:      0,
		JurisdictionRules:     nil,
	}, nil
}

// StoreAuthorizer is a no-op for encryptedTokenStore (EnhancedStore compatibility)
func (s *encryptedTokenStore) StoreAuthorizer(ctx context.Context, authorizer interface{}) error {
	return nil
}

// StoreOwner is a no-op for encryptedTokenStore (EnhancedStore compatibility)
func (s *encryptedTokenStore) StoreOwner(ctx context.Context, owner interface{}) error { return nil }

// StoreToken is a no-op for encryptedTokenStore (EnhancedStore compatibility)
func (s *encryptedTokenStore) StoreToken(ctx context.Context, token interface{}) error { return nil }
