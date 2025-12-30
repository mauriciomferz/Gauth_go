// Moved from encrypted_store.go
package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	token "github.com/mauriciomferz/AgentAuth/pkg/token"
)

// --- EncryptedStore Definition ---
type EncryptedStore struct {
	store    token.Store
	gcm      cipher.AEAD
	nonceLen int
}

// --- EncryptedStore Constructor ---
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

// --- Encryption Helper ---
func (s *EncryptedStore) encrypt(data []byte) (string, error) {
	nonce := make([]byte, s.nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := s.gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString(encrypted), nil
}

// --- Decryption Helper ---
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

// --- Save Method: Encrypts Token Value ---
func (s *EncryptedStore) Save(ctx context.Context, t *token.Token) error {
	if encrypted, err := s.encrypt([]byte(t.Value)); err != nil {
		return fmt.Errorf("failed to encrypt token value: %w", err)
	} else {
		t.Value = encrypted
	}

	// Metadata encryption not implemented for strongly-typed Metadata struct

	return s.store.Save(ctx, t.ID, t)
}

// --- Get Method: Retrieves Token ---
func (s *EncryptedStore) Get(ctx context.Context, id string) (*token.Token, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// ... (rest of the code remains unchanged)
	return t, nil
}

// --- Example Usage ---
func main() {
	// Example usage of EncryptedStore
	fmt.Println("EncryptedStore example loaded.")
}
