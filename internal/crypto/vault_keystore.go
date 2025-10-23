package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VaultKeyStore implements KeyStore using HashiCorp Vault.
type VaultKeyStore struct {
	client    VaultClient
	kvPath    string // KV mount path
	transitPath string // Transit mount path (optional)
	tokenTTL  time.Duration
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
	client := &httpVaultClient{
		address: config.Address,
		token:   config.Token,
		client:  &http.Client{Timeout: 30 * time.Second},
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
	
	// Store key in Vault KV
	keyData := map[string]interface{}{
		"algorithm":   "Ed25519",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"expires_at":  time.Now().Add(v.tokenTTL).UTC().Format(time.RFC3339),
		"private_key": base64.StdEncoding.EncodeToString(priv),
		"public_key":  base64.StdEncoding.EncodeToString(pub),
		"tenant":      tenant,
		"active":      false,
	}
	
	path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, keyID)
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
	path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, keyID)
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
	path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, keyID)
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
			path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, key.ID)
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
	path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, keyID)
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
	listPath := fmt.Sprintf("%s/metadata/gauth/keys/%s", v.kvPath, tenant)
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
	path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, keyID)
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
		path := fmt.Sprintf("%s/data/gauth/keys/%s/%s", v.kvPath, tenant, key.ID)
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
	privateKeyB64, _ := keyData["private_key"].(string)
	publicKeyB64, _ := keyData["public_key"].(string)
	
	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
	
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
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

// HTTP Vault Client implementation

type httpVaultClient struct {
	address string
	token   string
	client  *http.Client
}

func (c *httpVaultClient) Read(ctx context.Context, path string) (*VaultResponse, error) {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimSuffix(c.address, "/"), path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("X-Vault-Token", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault request failed with status %d", resp.StatusCode)
	}
	
	var vaultResp VaultResponse
	if err := json.NewDecoder(resp.Body).Decode(&vaultResp); err != nil {
		return nil, err
	}
	
	return &vaultResp, nil
}

func (c *httpVaultClient) Write(ctx context.Context, path string, data map[string]interface{}) (*VaultResponse, error) {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimSuffix(c.address, "/"), path)
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vault write failed with status %d", resp.StatusCode)
	}
	
	var vaultResp VaultResponse
	if err := json.NewDecoder(resp.Body).Decode(&vaultResp); err != nil {
		// Write operations may not return data, so ignore decode errors
		return &VaultResponse{}, nil
	}
	
	return &vaultResp, nil
}

func (c *httpVaultClient) Delete(ctx context.Context, path string) error {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimSuffix(c.address, "/"), path)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("X-Vault-Token", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("vault delete failed with status %d", resp.StatusCode)
	}
	
	return nil
}

func (c *httpVaultClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/sys/health", strings.TrimSuffix(c.address, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Vault health endpoint returns various status codes based on health
	// 200 = initialized, unsealed, and active
	// 429 = unsealed and standby
	// 472 = disaster recovery mode replication secondary and active
	// 473 = performance standby
	// 501 = not initialized
	// 503 = sealed
	
	if resp.StatusCode == 200 || resp.StatusCode == 429 {
		return nil // Healthy states
	}
	
	return fmt.Errorf("vault unhealthy, status code: %d", resp.StatusCode)
}