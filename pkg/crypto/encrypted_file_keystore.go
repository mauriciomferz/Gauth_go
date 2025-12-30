package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptedStoreConfig holds configuration for encrypted storage.
type EncryptedStoreConfig struct {
	FilePath      string
	MasterKeyPath string // Path to master key file
	MasterKeyEnv  string // Environment variable containing master key
	KMSKeyID      string // KMS key ID for envelope encryption (future)
}

// NewEncryptedFileKeyStore creates an encrypted file-based keystore.
// This enhances FileKeyStore with encryption at rest using AES-256-GCM.
func NewEncryptedFileKeyStore(config EncryptedStoreConfig) (*FileKeyStore, error) {
	// Load or derive master key
	masterKey, err := loadMasterKey(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load master key: %w", err)
	}

	// FileKeyStore already has master key field, just use it directly
	// with the base path from config
	store, err := NewFileKeyStore(config.FilePath, 0) // 0 TTL means no expiration
	if err != nil {
		return nil, fmt.Errorf("failed to create file keystore: %w", err)
	}

	// Set the master key for encryption
	store.masterKey = masterKey

	return store, nil
}

// loadMasterKey loads or derives the master encryption key.
func loadMasterKey(config EncryptedStoreConfig) ([]byte, error) {
	// Priority: KMS > File > Environment variable

	// TODO: Future - load from KMS
	if config.KMSKeyID != "" {
		return nil, errors.New("KMS master key loading not yet implemented")
	}

	// Load from file
	if config.MasterKeyPath != "" {
		key, err := os.ReadFile(config.MasterKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read master key file: %w", err)
		}

		// Decode if base64
		if decoded, err := base64.StdEncoding.DecodeString(string(key)); err == nil {
			if len(decoded) == 32 {
				return decoded, nil
			}
		}

		// Use as-is if 32 bytes
		if len(key) == 32 {
			return key, nil
		}

		// Derive 32-byte key using PBKDF2
		return pbkdf2.Key(key, []byte("agentauth-encryption-salt"), 100000, 32, sha256.New), nil
	}

	// Load from environment variable
	if config.MasterKeyEnv != "" {
		keyStr := os.Getenv(config.MasterKeyEnv)
		if keyStr == "" {
			return nil, fmt.Errorf("environment variable %s not set", config.MasterKeyEnv)
		}

		// Try base64 decode
		if decoded, err := base64.StdEncoding.DecodeString(keyStr); err == nil {
			if len(decoded) == 32 {
				return decoded, nil
			}
		}

		// Derive key from passphrase
		return pbkdf2.Key([]byte(keyStr), []byte("agentauth-encryption-salt"), 100000, 32, sha256.New), nil
	}

	return nil, errors.New("no master key source configured (set MasterKeyPath or MasterKeyEnv)")
}

// GenerateMasterKey generates a new random 256-bit master key.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	return key, nil
}

// EncodeMasterKey encodes a master key as base64 for storage.
func EncodeMasterKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// encryptBytes encrypts plaintext using AES-256-GCM.
func encryptBytes(masterKey, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends nonce + ciphertext + tag
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptBytes decrypts ciphertext using AES-256-GCM.
func decryptBytes(masterKey, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
