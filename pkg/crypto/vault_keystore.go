package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// VaultKeyStore implements KeyStore using HashiCorp Vault.
type VaultKeyStore struct {
	client      VaultClient
	kvPath      string // KV mount path
	transitPath string // Transit mount path (optional)
	tokenTTL    time.Duration
}

// VaultClient interface for testing and abstraction.
type VaultClient interface {
	Read(ctx context.Context, path string) (*VaultResponse, error)
	Write(ctx context.Context, path string, data map[string]interface{}) (*VaultResponse, error)
	Delete(ctx context.Context, path string) error
	Health(ctx context.Context) error
}

// VaultResponse represents a Vault API response.
type VaultResponse struct {
	Data map[string]interface{} `json:"data"`
}

// VaultConfig holds Vault configuration.
type VaultConfig struct {
	Address     string
	Token       string
	KVPath      string
	TransitPath string
	TokenTTL    time.Duration
}

// NewVaultKeyStore creates a new Vault-backed key store.
func NewVaultKeyStore(config VaultConfig) (*VaultKeyStore, error) {
	// Use official SDK client
	client, err := NewVaultSDKClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Default paths
	if config.KVPath == "" {
		config.KVPath = "secret"
	}
	if config.TransitPath == "" {
		config.TransitPath = "transit"
	}
	if config.TokenTTL == 0 {
		config.TokenTTL = time.Hour
	}

	store := &VaultKeyStore{
		client:      client,
		kvPath:      config.KVPath,
		transitPath: config.TransitPath,
		tokenTTL:    config.TokenTTL,
	}

	// Test connectivity
	if err := store.Health(context.Background()); err != nil {
		return nil, fmt.Errorf("vault connectivity test failed: %w", err)
	}

	return store, nil
}

// Generate creates a new Ed25519 key pair in Vault.
func (v *VaultKeyStore) Generate(ctx context.Context, tenant string) (string, error) {
	// Generate Ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("key generation failed: %w", err)
	}

	// Create key ID
	keyID := base64.RawURLEncoding.EncodeToString(pub[:8])

	// Encrypt private key using Vault transit if configured
	var encryptedPriv string
	if v.transitPath != "" {
		// Use Vault transit encrypt API
		transitKey := "agentauth-key"
		transitEncryptPath := fmt.Sprintf("%s/encrypt/%s", v.transitPath, transitKey)
		req := map[string]interface{}{"plaintext": base64.StdEncoding.EncodeToString(priv)}
		resp, err := v.client.Write(ctx, transitEncryptPath, req)
		if err != nil {
			return "", fmt.Errorf("vault transit encrypt failed: %w", err)
		}
		ciphertext, ok := resp.Data["ciphertext"].(string)
		if !ok {
			return "", fmt.Errorf("vault transit encrypt: missing ciphertext")
		}
		encryptedPriv = ciphertext
	} else {
		encryptedPriv = base64.StdEncoding.EncodeToString(priv)
	}

	// Store key in Vault KV
	keyData := map[string]interface{}{
		"algorithm":   "Ed25519",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"expires_at":  time.Now().Add(v.tokenTTL).UTC().Format(time.RFC3339),
		"private_key": encryptedPriv,
		"public_key":  base64.StdEncoding.EncodeToString(pub),
		"tenant":      tenant,
		"active":      false,
	}

	path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, keyID)
	if _, err := v.client.Write(ctx, path, map[string]interface{}{"data": keyData}); err != nil {
		return "", fmt.Errorf("vault key storage failed: %w", err)
	}

	return keyID, nil
}

// Activate marks a key as active for a tenant.
func (v *VaultKeyStore) Activate(ctx context.Context, tenant, keyID string) error {
	// First, deactivate any existing active key
	if err := v.deactivateAllKeys(ctx, tenant); err != nil {
		return fmt.Errorf("failed to deactivate existing keys: %w", err)
	}

	// Activate the specified key
	path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, keyID)
	resp, err := v.client.Read(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to read key for activation: %w", err)
	}

	keyData, ok := resp.Data["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid key data format")
	}

	keyData["active"] = true
	keyData["activated_at"] = time.Now().UTC().Format(time.RFC3339)

	if _, err := v.client.Write(ctx, path, map[string]interface{}{"data": keyData}); err != nil {
		return fmt.Errorf("vault key activation failed: %w", err)
	}

	return nil
}

// Archive marks a key as archived (inactive but retained).
func (v *VaultKeyStore) Archive(ctx context.Context, tenant, keyID string) error {
	path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, keyID)
	resp, err := v.client.Read(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to read key for archiving: %w", err)
	}

	keyData, ok := resp.Data["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid key data format")
	}

	keyData["active"] = false
	keyData["archived_at"] = time.Now().UTC().Format(time.RFC3339)

	if _, err := v.client.Write(ctx, path, map[string]interface{}{"data": keyData}); err != nil {
		return fmt.Errorf("vault key archiving failed: %w", err)
	}

	return nil
}

// GetActive retrieves the currently active key for a tenant.
func (v *VaultKeyStore) GetActive(ctx context.Context, tenant string) (*Key, error) {
	keys, err := v.ListKeys(ctx, tenant)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.ID != "" { // Check if this is the active key by reading from Vault
			path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, key.ID)
			resp, err := v.client.Read(ctx, path)
			if err != nil {
				continue
			}

			keyData, ok := resp.Data["data"].(map[string]interface{})
			if !ok {
				continue
			}

			if active, ok := keyData["active"].(bool); ok && active {
				return key, nil
			}
		}
	}

	return nil, fmt.Errorf("no active key found for tenant %s", tenant)
}

// GetKey retrieves a specific key by ID.
func (v *VaultKeyStore) GetKey(ctx context.Context, tenant, keyID string) (*Key, error) {
	path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, keyID)
	resp, err := v.client.Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	keyData, ok := resp.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid key data format")
	}

	return v.parseVaultKey(keyID, keyData)
}

// ListKeys returns all keys for a tenant.
func (v *VaultKeyStore) ListKeys(ctx context.Context, tenant string) ([]*Key, error) {
	// List keys in the tenant's path
	listPath := fmt.Sprintf("%s/metadata/agentauth/keys/%s", v.kvPath, tenant)
	resp, err := v.client.Read(ctx, listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	keysInterface, ok := resp.Data["keys"]
	if !ok {
		return []*Key{}, nil // No keys found
	}

	keyNames, ok := keysInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid keys list format")
	}

	var keys []*Key
	for _, keyNameInterface := range keyNames {
		keyName, ok := keyNameInterface.(string)
		if !ok {
			continue
		}

		key, err := v.GetKey(ctx, tenant, keyName)
		if err != nil {
			continue // Skip failed key reads
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// Delete permanently removes a key.
func (v *VaultKeyStore) Delete(ctx context.Context, tenant, keyID string) error {
	path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, keyID)
	return v.client.Delete(ctx, path)
}

// Health checks Vault connectivity.
func (v *VaultKeyStore) Health(ctx context.Context) error {
	return v.client.Health(ctx)
}

// Helper methods

// deactivateAllKeys deactivates all keys for a tenant.
func (v *VaultKeyStore) deactivateAllKeys(ctx context.Context, tenant string) error {
	keys, err := v.ListKeys(ctx, tenant)
	if err != nil {
		return err
	}

	for _, key := range keys {
		path := fmt.Sprintf("%s/data/agentauth/keys/%s/%s", v.kvPath, tenant, key.ID)
		resp, err := v.client.Read(ctx, path)
		if err != nil {
			continue
		}

		keyData, ok := resp.Data["data"].(map[string]interface{})
		if !ok {
			continue
		}

		if active, ok := keyData["active"].(bool); ok && active {
			keyData["active"] = false
			if _, err := v.client.Write(ctx, path, map[string]interface{}{"data": keyData}); err != nil {
				// Log error but continue deactivating other keys
				continue
			}
		}
	}

	return nil
}

// parseVaultKey converts Vault key data to Key struct.
func (v *VaultKeyStore) parseVaultKey(keyID string, keyData map[string]interface{}) (*Key, error) {
	algorithm, _ := keyData["algorithm"].(string)
	createdAtStr, _ := keyData["created_at"].(string)
	expiresAtStr, _ := keyData["expires_at"].(string)
	privateKeyEnc, _ := keyData["private_key"].(string)
	publicKeyB64, _ := keyData["public_key"].(string)

	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)

	var privateKey []byte
	if v.transitPath != "" && len(privateKeyEnc) > 7 && privateKeyEnc[:7] == "vault:v" {
		// Use Vault transit decrypt API
		transitKey := "agentauth-key"
		transitDecryptPath := fmt.Sprintf("%s/decrypt/%s", v.transitPath, transitKey)
		req := map[string]interface{}{"ciphertext": privateKeyEnc}
		resp, err := v.client.Write(context.Background(), transitDecryptPath, req)
		if err != nil {
			return nil, fmt.Errorf("vault transit decrypt failed: %w", err)
		}
		plaintextB64, ok := resp.Data["plaintext"].(string)
		if !ok {
			return nil, fmt.Errorf("vault transit decrypt: missing plaintext")
		}
		privateKey, err = base64.StdEncoding.DecodeString(plaintextB64)
		if err != nil {
			return nil, fmt.Errorf("vault transit decode failed: %w", err)
		}
	} else {
		var err error
		privateKey, err = base64.StdEncoding.DecodeString(privateKeyEnc)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key: %w", err)
		}
	}

	publicKey, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return &Key{
		ID:        keyID,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		Private:   ed25519.PrivateKey(privateKey),
		Public:    ed25519.PublicKey(publicKey),
		Alg:       algorithm,
		Use:       "sig",
	}, nil
}
