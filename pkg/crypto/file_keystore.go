package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileKeyStore implements KeyStore using the local filesystem.
// This implementation is suitable for development and single-node deployments.
type FileKeyStore struct {
	basePath  string
	mu        sync.RWMutex
	ttl       time.Duration
	masterKey []byte // AES key for encryption at rest
}

// FileKeyData represents the JSON structure stored in key files.
type FileKeyData struct {
	Algorithm   string     `json:"algorithm"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	PrivateKey  string     `json:"private_key"` // Base64 encoded
	PublicKey   string     `json:"public_key"`  // Base64 encoded
	Tenant      string     `json:"tenant"`
	Active      bool       `json:"active"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
}

// NewFileKeyStore creates a new file-based key store.
func NewFileKeyStore(basePath string, ttl time.Duration) (*FileKeyStore, error) {
	if basePath == "" {
		return nil, fmt.Errorf("basePath cannot be empty")
	}

	if ttl == 0 {
		ttl = 24 * time.Hour // Default TTL
	}

	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	var masterKey []byte
	keyStr := os.Getenv("GAUTH_FILEKEYSTORE_MASTER_KEY")
	if keyStr != "" {
		mk, err := base64.StdEncoding.DecodeString(keyStr)
		if err == nil && (len(mk) == 32) {
			masterKey = mk
		}
	}

	return &FileKeyStore{
		basePath:  basePath,
		ttl:       ttl,
		masterKey: masterKey,
	}, nil
}

// Generate creates a new Ed25519 key pair and stores it to disk.
func (f *FileKeyStore) Generate(ctx context.Context, tenant string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Generate Ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("key generation failed: %w", err)
	}

	// Create key ID from first 8 bytes of public key
	keyID := base64.RawURLEncoding.EncodeToString(pub[:8])

	// Prepare key data
	now := time.Now().UTC()
	keyData := FileKeyData{
		Algorithm:  "Ed25519",
		CreatedAt:  now,
		ExpiresAt:  now.Add(f.ttl),
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		Tenant:     tenant,
		Active:     false,
	}

	// Ensure tenant directory exists
	tenantDir := filepath.Join(f.basePath, tenant)
	if err := os.MkdirAll(tenantDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create tenant directory: %w", err)
	}

	// Write key to file
	keyPath := filepath.Join(tenantDir, keyID+".json")
	if err := f.writeKeyFile(keyPath, keyData); err != nil {
		return "", fmt.Errorf("failed to write key file: %w", err)
	}

	return keyID, nil
}

// Activate marks a key as active for a tenant.
func (f *FileKeyStore) Activate(ctx context.Context, tenant, keyID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// First, deactivate all existing keys for the tenant
	if err := f.deactivateAllKeysLocked(tenant); err != nil {
		return fmt.Errorf("failed to deactivate existing keys: %w", err)
	}

	// Load and activate the specified key
	keyPath := filepath.Join(f.basePath, tenant, keyID+".json")
	keyData, err := f.readKeyFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	now := time.Now().UTC()
	keyData.Active = true
	keyData.ActivatedAt = &now

	if err := f.writeKeyFile(keyPath, *keyData); err != nil {
		return fmt.Errorf("failed to update key file: %w", err)
	}

	return nil
}

// Archive marks a key as archived (inactive but retained).
func (f *FileKeyStore) Archive(ctx context.Context, tenant, keyID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	keyPath := filepath.Join(f.basePath, tenant, keyID+".json")
	keyData, err := f.readKeyFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	now := time.Now().UTC()
	keyData.Active = false
	keyData.ArchivedAt = &now

	if err := f.writeKeyFile(keyPath, *keyData); err != nil {
		return fmt.Errorf("failed to update key file: %w", err)
	}

	return nil
}

// GetActive retrieves the currently active key for a tenant.
func (f *FileKeyStore) GetActive(ctx context.Context, tenant string) (*Key, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	tenantDir := filepath.Join(f.basePath, tenant)
	entries, err := os.ReadDir(tenantDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no keys found for tenant %s", tenant)
		}
		return nil, fmt.Errorf("failed to read tenant directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			keyPath := filepath.Join(tenantDir, entry.Name())
			keyData, err := f.readKeyFile(keyPath)
			if err != nil {
				continue
			}

			if keyData.Active {
				keyID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
				return f.parseFileKey(keyID, *keyData)
			}
		}
	}

	return nil, fmt.Errorf("no active key found for tenant %s", tenant)
}

// GetKey retrieves a specific key by ID.
func (f *FileKeyStore) GetKey(ctx context.Context, tenant, keyID string) (*Key, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keyPath := filepath.Join(f.basePath, tenant, keyID+".json")
	keyData, err := f.readKeyFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	return f.parseFileKey(keyID, *keyData)
}

// ListKeys returns all keys for a tenant.
func (f *FileKeyStore) ListKeys(ctx context.Context, tenant string) ([]*Key, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	tenantDir := filepath.Join(f.basePath, tenant)
	entries, err := os.ReadDir(tenantDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Key{}, nil // No keys found
		}
		return nil, fmt.Errorf("failed to read tenant directory: %w", err)
	}

	var keys []*Key
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			keyID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			keyPath := filepath.Join(tenantDir, entry.Name())

			keyData, err := f.readKeyFile(keyPath)
			if err != nil {
				continue // Skip invalid key files
			}

			key, err := f.parseFileKey(keyID, *keyData)
			if err != nil {
				continue // Skip unparseable keys
			}

			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Delete permanently removes a key from disk.
func (f *FileKeyStore) Delete(ctx context.Context, tenant, keyID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	keyPath := filepath.Join(f.basePath, tenant, keyID+".json")
	if err := os.Remove(keyPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key not found: %s", keyID)
		}
		return fmt.Errorf("failed to delete key file: %w", err)
	}

	return nil
}

// Health checks if the file system is accessible.
func (f *FileKeyStore) Health(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Test write access to base directory
	testFile := filepath.Join(f.basePath, ".health_check")
	if err := os.WriteFile(testFile, []byte("health"), 0600); err != nil {
		return fmt.Errorf("filesystem health check failed: %w", err)
	}

	// Clean up test file
	os.Remove(testFile)
	return nil
}

// Helper methods

// deactivateAllKeysLocked deactivates all keys for a tenant (must be called with lock held).
func (f *FileKeyStore) deactivateAllKeysLocked(tenant string) error {
	tenantDir := filepath.Join(f.basePath, tenant)
	entries, err := os.ReadDir(tenantDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No keys to deactivate
		}
		return fmt.Errorf("failed to read tenant directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			keyPath := filepath.Join(tenantDir, entry.Name())
			keyData, err := f.readKeyFile(keyPath)
			if err != nil {
				continue
			}

			if keyData.Active {
				keyData.Active = false
				if err := f.writeKeyFile(keyPath, *keyData); err != nil {
					// Log error but continue deactivating other keys
					continue
				}
			}
		}
	}

	return nil
}

// readKeyFile reads and parses a key file.
func (f *FileKeyStore) readKeyFile(path string) (*FileKeyData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var keyData FileKeyData
	if err := json.Unmarshal(data, &keyData); err != nil {
		return nil, fmt.Errorf("failed to parse key file: %w", err)
	}

	return &keyData, nil
}

// writeKeyFile writes key data to a file with proper permissions.
func (f *FileKeyStore) writeKeyFile(path string, keyData FileKeyData) error {
	data, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// parseFileKey converts file key data to Key struct.
func (f *FileKeyStore) parseFileKey(keyID string, keyData FileKeyData) (*Key, error) {
	privBytes, err := base64.StdEncoding.DecodeString(keyData.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	if len(f.masterKey) == 32 {
		block, err2 := aes.NewCipher(f.masterKey)
		if err2 != nil {
			return nil, fmt.Errorf("AES cipher error: %w", err2)
		}
		gcm, err3 := cipher.NewGCM(block)
		if err3 != nil {
			return nil, fmt.Errorf("AES-GCM error: %w", err3)
		}
		nonceSize := gcm.NonceSize()
		if len(privBytes) < nonceSize {
			return nil, fmt.Errorf("encrypted private key too short")
		}
		nonce, ciphertext := privBytes[:nonceSize], privBytes[nonceSize:]
		privBytes, err = gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("AES-GCM decryption failed: %w", err)
		}
	}
	publicKey, err := base64.StdEncoding.DecodeString(keyData.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}
	return &Key{
		ID:        keyID,
		CreatedAt: keyData.CreatedAt,
		ExpiresAt: keyData.ExpiresAt,
		Private:   ed25519.PrivateKey(privBytes),
		Public:    ed25519.PublicKey(publicKey),
		Alg:       keyData.Algorithm,
		Use:       "sig",
	}, nil
}

// Cleanup removes expired keys from disk.
func (f *FileKeyStore) Cleanup(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().UTC()

	return filepath.WalkDir(f.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(d.Name()) == ".json" {
			keyData, err := f.readKeyFile(path)
			if err != nil {
				return nil // Skip invalid files
			}

			// Remove expired and archived keys older than 30 days
			if now.After(keyData.ExpiresAt) ||
				(keyData.ArchivedAt != nil && now.Sub(*keyData.ArchivedAt) > 30*24*time.Hour) {
				os.Remove(path)
			}
		}

		return nil
	})
}
