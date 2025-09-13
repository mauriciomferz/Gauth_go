package tokenmanagement

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// EncryptedStore wraps a token store with encryption
type EncryptedStore struct {
	store    token.Store
	gcm      cipher.AEAD
	nonceLen int
}

// NewEncryptedStore creates a new store with AES-GCM encryption
func NewEncryptedStore(baseStore token.Store, key []byte) (*EncryptedStore, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &EncryptedStore{
		store:    baseStore,
		gcm:      gcm,
		nonceLen: gcm.NonceSize(),
	}, nil
}

// encrypt data using AES-GCM
func (s *EncryptedStore) encrypt(data []byte) (string, error) {
	nonce := make([]byte, s.nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := s.gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString(encrypted), nil
}

// decrypt data using AES-GCM
func (s *EncryptedStore) decrypt(data string) ([]byte, error) {
	encrypted, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	if len(encrypted) < s.nonceLen {
		return nil, fmt.Errorf("invalid encrypted data")
	}

	nonce := encrypted[:s.nonceLen]
	ciphertext := encrypted[s.nonceLen:]

	decrypted, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return decrypted, nil
}

// Save stores an encrypted token
func (s *EncryptedStore) Save(ctx context.Context, t *token.Token) error {
	// Encrypt sensitive fields
	if encrypted, err := s.encrypt([]byte(t.Value)); err != nil {
		return fmt.Errorf("failed to encrypt token value: %w", err)
	} else {
		t.Value = encrypted
	}

	// Encrypt metadata
	for k, v := range t.Metadata {
		if encrypted, err := s.encrypt([]byte(v)); err != nil {
			return fmt.Errorf("failed to encrypt metadata: %w", err)
		}
		t.Metadata[k] = encrypted
	}

	return s.store.Save(ctx, t)
}

// Get retrieves and decrypts a token
func (s *EncryptedStore) Get(ctx context.Context, id string) (*token.Token, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Decrypt token value
	if decrypted, err := s.decrypt(t.Value); err != nil {
		return nil, fmt.Errorf("failed to decrypt token value: %w", err)
	} else {
		t.Value = string(decrypted)
	}

	// Decrypt metadata
	for k, v := range t.Metadata {
		if decrypted, err := s.decrypt(v); err != nil {
			return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
		}
		t.Metadata[k] = string(decrypted)
	}

	return t, nil
}

// Example usage
func main() {
	ctx := context.Background()

	// Create base store
	baseStore := token.NewMemoryStore(24 * time.Hour)

	// Create encrypted store with a 32-byte key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	encryptedStore, err := NewEncryptedStore(baseStore, key)
	if err != nil {
		log.Fatalf("Failed to create encrypted store: %v", err)
	}

	// Create a token with sensitive data
	t := &token.Token{
		ID:      token.NewID(),
		Type:    token.Access,
		Subject: "user123",
		Value:   "sensitive-token-value",
		Metadata: map[string]string{
			"user_email": "user@example.com",
			"device_id":  "device123",
		},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Save encrypted token
	if err := encryptedStore.Save(ctx, t); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}
	fmt.Printf("Saved encrypted token: %s\n", t.ID)

	// Retrieve and decrypt token
	retrieved, err := encryptedStore.Get(ctx, t.ID)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}

	fmt.Printf("\nRetrieved token:\n")
	fmt.Printf("Value: %s\n", retrieved.Value)
	fmt.Printf("Email: %s\n", retrieved.Metadata["user_email"])
	fmt.Printf("Device: %s\n", retrieved.Metadata["device_id"])

	// Demonstrate that data is actually encrypted in storage
	raw, _ := baseStore.Get(ctx, t.ID)
	fmt.Printf("\nEncrypted data in storage:\n")
	fmt.Printf("Value: %s\n", raw.Value)
	fmt.Printf("Email: %s\n", raw.Metadata["user_email"])
}
