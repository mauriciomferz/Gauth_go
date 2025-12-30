// Package agentauth - JWE Key Registry
// Provides multi-key support for key rotation without service restart
package agentauth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// KeyRegistry manages multiple JWE encryption keys for rotation support
// - Encryption uses the current (newest) key
// - Decryption tries all active keys (supports old tokens during rotation)
// - Keys can be loaded from directory or added dynamically
type KeyRegistry struct {
	publicKeys  map[string]*rsa.PublicKey  // kid -> public key
	privateKeys map[string]*rsa.PrivateKey // kid -> private key
	currentKID  string                     // Key ID of current encryption key
	mu          sync.RWMutex
}

// NewKeyRegistry creates a new key registry
func NewKeyRegistry() *KeyRegistry {
	return &KeyRegistry{
		publicKeys:  make(map[string]*rsa.PublicKey),
		privateKeys: make(map[string]*rsa.PrivateKey),
	}
}

// AddKey adds a public/private key pair to the registry
func (r *KeyRegistry) AddKey(kid string, publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) error {
	if kid == "" {
		return fmt.Errorf("key ID cannot be empty")
	}
	if publicKey == nil {
		return fmt.Errorf("public key cannot be nil")
	}
	if privateKey == nil {
		return fmt.Errorf("private key cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.publicKeys[kid] = publicKey
	r.privateKeys[kid] = privateKey

	// If this is the first key, make it current
	if r.currentKID == "" {
		r.currentKID = kid
	}

	return nil
}

// LoadKeysFromDirectory loads all key pairs from a directory
// Expected file naming convention:
//   - Public keys: {kid}.pub.pem or {kid}-public.pem
//   - Private keys: {kid}.priv.pem or {kid}-private.pem
func (r *KeyRegistry) LoadKeysFromDirectory(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// Check directory exists
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}

	// Find all key files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	keyPairs := make(map[string]struct {
		publicKeyPath  string
		privateKeyPath string
	})

	// Identify key pairs
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".pem") {
			continue
		}

		var kid string
		var isPublic bool

		// Extract key ID and type from filename
		if strings.HasSuffix(name, ".pub.pem") {
			kid = strings.TrimSuffix(name, ".pub.pem")
			isPublic = true
		} else if strings.HasSuffix(name, "-public.pem") {
			kid = strings.TrimSuffix(name, "-public.pem")
			isPublic = true
		} else if strings.HasSuffix(name, ".priv.pem") {
			kid = strings.TrimSuffix(name, ".priv.pem")
			isPublic = false
		} else if strings.HasSuffix(name, "-private.pem") {
			kid = strings.TrimSuffix(name, "-private.pem")
			isPublic = false
		} else {
			continue // Unknown format
		}

		// Initialize key pair entry if needed
		if _, exists := keyPairs[kid]; !exists {
			keyPairs[kid] = struct {
				publicKeyPath  string
				privateKeyPath string
			}{}
		}

		// Update paths
		pair := keyPairs[kid]
		fullPath := filepath.Join(dir, name)
		if isPublic {
			pair.publicKeyPath = fullPath
		} else {
			pair.privateKeyPath = fullPath
		}
		keyPairs[kid] = pair
	}

	if len(keyPairs) == 0 {
		return fmt.Errorf("no key pairs found in directory: %s", dir)
	}

	// Load each key pair
	for kid, paths := range keyPairs {
		if paths.publicKeyPath == "" {
			return fmt.Errorf("missing public key for kid: %s", kid)
		}
		if paths.privateKeyPath == "" {
			return fmt.Errorf("missing private key for kid: %s", kid)
		}

		// Load public key
		publicKey, err := LoadRSAPublicKey(paths.publicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load public key for %s: %w", kid, err)
		}

		// Load private key
		privateKey, err := LoadRSAPrivateKey(paths.privateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load private key for %s: %w", kid, err)
		}

		// Add to registry
		if err := r.AddKey(kid, publicKey, privateKey); err != nil {
			return fmt.Errorf("failed to add key %s: %w", kid, err)
		}
	}

	// Set current key to newest (by kid lexicographic order)
	if err := r.SetCurrentKeyByNewest(); err != nil {
		return fmt.Errorf("failed to set current key: %w", err)
	}

	return nil
}

// LoadKeysFromEnvironment loads keys from environment variables
// Supports AGENTAUTH_JWE_KEY_DIR for multi-key setup or single key pair
func (r *KeyRegistry) LoadKeysFromEnvironment() error {
	// Check for key directory first
	if keyDir := os.Getenv("AGENTAUTH_JWE_KEY_DIR"); keyDir != "" {
		return r.LoadKeysFromDirectory(keyDir)
	}

	// Fall back to single key pair
	publicKeyPath := os.Getenv("AGENTAUTH_JWE_PUBLIC_KEY")
	privateKeyPath := os.Getenv("AGENTAUTH_JWE_PRIVATE_KEY")
	keyID := os.Getenv("AGENTAUTH_JWE_KEY_ID")

	if publicKeyPath == "" || privateKeyPath == "" {
		return fmt.Errorf("AGENTAUTH_JWE_PUBLIC_KEY and AGENTAUTH_JWE_PRIVATE_KEY must be set")
	}

	if keyID == "" {
		keyID = fmt.Sprintf("agentauth-default-%s", time.Now().Format("2006-01"))
	}

	// Load keys
	publicKey, err := LoadRSAPublicKey(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load public key: %w", err)
	}

	privateKey, err := LoadRSAPrivateKey(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	return r.AddKey(keyID, publicKey, privateKey)
}

// GetCurrentKey returns the current encryption key
func (r *KeyRegistry) GetCurrentKey() (kid string, publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentKID == "" {
		return "", nil, nil, fmt.Errorf("no current key set")
	}

	publicKey, pubExists := r.publicKeys[r.currentKID]
	privateKey, privExists := r.privateKeys[r.currentKID]

	if !pubExists || !privExists {
		return "", nil, nil, fmt.Errorf("current key not found: %s", r.currentKID)
	}

	return r.currentKID, publicKey, privateKey, nil
}

// GetPublicKey retrieves a public key by key ID
func (r *KeyRegistry) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	publicKey, exists := r.publicKeys[kid]
	if !exists {
		return nil, fmt.Errorf("public key not found: %s", kid)
	}

	return publicKey, nil
}

// GetPrivateKey retrieves a private key by key ID
func (r *KeyRegistry) GetPrivateKey(kid string) (*rsa.PrivateKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	privateKey, exists := r.privateKeys[kid]
	if !exists {
		return nil, fmt.Errorf("private key not found: %s", kid)
	}

	return privateKey, nil
}

// SetCurrentKey sets the current encryption key by key ID
func (r *KeyRegistry) SetCurrentKey(kid string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify key exists
	if _, exists := r.publicKeys[kid]; !exists {
		return fmt.Errorf("key not found: %s", kid)
	}

	r.currentKID = kid
	return nil
}

// SetCurrentKeyByNewest sets the current key to the newest (by lexicographic order)
// Assumes key IDs follow a sortable naming convention (e.g., agentauth-prod-2025-11)
func (r *KeyRegistry) SetCurrentKeyByNewest() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.publicKeys) == 0 {
		return fmt.Errorf("no keys in registry")
	}

	// Get all key IDs and sort
	kids := make([]string, 0, len(r.publicKeys))
	for kid := range r.publicKeys {
		kids = append(kids, kid)
	}
	sort.Strings(kids)

	// Set newest (last in sorted order)
	r.currentKID = kids[len(kids)-1]
	return nil
}

// ListKeys returns all key IDs in the registry
func (r *KeyRegistry) ListKeys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	kids := make([]string, 0, len(r.publicKeys))
	for kid := range r.publicKeys {
		kids = append(kids, kid)
	}
	sort.Strings(kids)
	return kids
}

// RemoveKey removes a key pair from the registry
func (r *KeyRegistry) RemoveKey(kid string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if kid == r.currentKID {
		return fmt.Errorf("cannot remove current key: %s", kid)
	}

	delete(r.publicKeys, kid)
	delete(r.privateKeys, kid)
	return nil
}

// KeyCount returns the number of key pairs in the registry
func (r *KeyRegistry) KeyCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.publicKeys)
}

// JWEServiceWithRegistry is a JWE service that uses a key registry
type JWEServiceWithRegistry struct {
	config   *JWEConfig
	registry *KeyRegistry
}

// NewJWEServiceWithRegistry creates a JWE service with a key registry
func NewJWEServiceWithRegistry(config *JWEConfig, registry *KeyRegistry) (*JWEServiceWithRegistry, error) {
	if config == nil {
		return nil, fmt.Errorf("JWE config cannot be nil")
	}
	if registry == nil {
		return nil, fmt.Errorf("key registry cannot be nil")
	}

	// Validate registry has at least one key
	if registry.KeyCount() == 0 {
		return nil, fmt.Errorf("key registry is empty")
	}

	return &JWEServiceWithRegistry{
		config:   config,
		registry: registry,
	}, nil
}

// EncryptToken encrypts using the current key from the registry
func (s *JWEServiceWithRegistry) EncryptToken(ctx context.Context, jwtString string) (string, error) {
	if !s.config.Enabled {
		return jwtString, nil // Pass-through if disabled
	}

	// Get current key
	kid, publicKey, _, err := s.registry.GetCurrentKey()
	if err != nil {
		return "", fmt.Errorf("failed to get current key: %w", err)
	}

	// Create temporary config for this key
	keyConfig := *s.config
	keyConfig.KeyID = kid

	// Create temporary service (will be optimized in production)
	tmpService, err := NewJWEService(&keyConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create JWE service: %w", err)
	}

	// Override with registry key
	tmpService.publicKeys[kid] = publicKey

	return tmpService.EncryptToken(ctx, jwtString)
}

// DecryptToken tries all keys in the registry until one succeeds
func (s *JWEServiceWithRegistry) DecryptToken(ctx context.Context, jweString string) (string, error) {
	if !s.config.Enabled {
		return jweString, nil // Pass-through if disabled
	}

	// Extract key ID from JWE (if available)
	// For now, try all keys sequentially

	kids := s.registry.ListKeys()
	var lastErr error

	for _, kid := range kids {
		privateKey, err := s.registry.GetPrivateKey(kid)
		if err != nil {
			lastErr = err
			continue
		}

		// Create temporary config for this key
		keyConfig := *s.config
		keyConfig.KeyID = kid

		// Create temporary service
		tmpService, err := NewJWEService(&keyConfig)
		if err != nil {
			lastErr = err
			continue
		}

		// Override with registry key
		tmpService.privateKeys[kid] = privateKey

		// Try decryption
		jwtString, err := tmpService.DecryptToken(ctx, jweString)
		if err == nil {
			return jwtString, nil // Success!
		}
		lastErr = err
	}

	return "", fmt.Errorf("failed to decrypt with any key in registry (tried %d keys): %w", len(kids), lastErr)
}

// IsEnabled returns whether JWE encryption is enabled
func (s *JWEServiceWithRegistry) IsEnabled() bool {
	return s.config.Enabled
}

// RotateKeys reloads keys from environment/directory
func (s *JWEServiceWithRegistry) RotateKeys(ctx context.Context) error {
	return s.registry.LoadKeysFromEnvironment()
}

// GetPublicKey retrieves a public key by key ID from the registry
func (s *JWEServiceWithRegistry) GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	return s.registry.GetPublicKey(kid)
}
