package secret

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptedProvider wraps any Provider and adds transparent encryption at rest.
// Secrets are encrypted with AES-256-GCM before storage and decrypted on retrieval.
type EncryptedProvider struct {
	backend Provider
	key     []byte // 32-byte encryption key
}

// NewEncrypted creates an encrypted provider wrapping the given backend.
// The key is derived from passphrase using PBKDF2-SHA256 (100,000 iterations).
// For production, use a strong random passphrase (>= 32 bytes entropy).
func NewEncrypted(backend Provider, passphrase string) (*EncryptedProvider, error) {
	if backend == nil {
		return nil, errors.New("backend provider required")
	}
	if len(passphrase) < 16 {
		return nil, errors.New("passphrase too short (minimum 16 characters)")
	}

	// Derive 32-byte key using PBKDF2-SHA256
	// Salt is fixed for deterministic key derivation from passphrase
	// In production, consider per-tenant salts stored separately
	salt := []byte("agentauth-secret-storage-v1")
	key := pbkdf2.Key([]byte(passphrase), salt, 100000, 32, sha256.New)

	return &EncryptedProvider{
		backend: backend,
		key:     key,
	}, nil
}

// Get retrieves and decrypts a secret.
func (e *EncryptedProvider) Get(ctx context.Context, secretKey string) (string, error) {
	encrypted, err := e.backend.Get(ctx, secretKey)
	if err != nil {
		return "", err
	}

	decrypted, err := e.decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}

	return decrypted, nil
}

// Set encrypts and stores a secret.
func (e *EncryptedProvider) Set(ctx context.Context, secretKey, value string, opts ...Option) error {
	encrypted, err := e.encrypt(value)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	return e.backend.Set(ctx, secretKey, encrypted, opts...)
}

// Delete removes a secret.
func (e *EncryptedProvider) Delete(ctx context.Context, secretKey string) error {
	return e.backend.Delete(ctx, secretKey)
}

// List returns keys matching prefix (keys are not encrypted, only values).
func (e *EncryptedProvider) List(ctx context.Context, prefix string) ([]string, error) {
	return e.backend.List(ctx, prefix)
}

// Name returns the backend provider name with encryption indicator.
func (e *EncryptedProvider) Name() string {
	return fmt.Sprintf("encrypted(%s)", e.backend.Name())
}

// encrypt uses AES-256-GCM to encrypt plaintext.
// Returns base64-encoded: nonce(12) + ciphertext + tag(16)
func (e *EncryptedProvider) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce (12 bytes for GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt and append tag: nonce + ciphertext + tag
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decrypt reverses the encryption process.
func (e *EncryptedProvider) decrypt(encoded string) (string, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
