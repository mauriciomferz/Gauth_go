// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// VaultBackend provides production-ready secret storage using HashiCorp Vault.
// Addresses gap sec8.item1 (P0): Secure secret storage with production backend.
type VaultBackend struct {
	address    string
	token      string
	namespace  string
	mountPath  string
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]*vaultSecret // short-lived cache to reduce API calls
}

type vaultSecret struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	keyID      string
	cachedAt   time.Time
	ttl        time.Duration
}

// VaultBackendConfig configures the Vault backend.
type VaultBackendConfig struct {
	Address   string        // Vault server address (e.g., https://vault.example.com:8200)
	Token     string        // Vault authentication token
	Namespace string        // Vault namespace (optional, for Enterprise)
	MountPath string        // KV mount path (default: "secret")
	CacheTTL  time.Duration // How long to cache secrets (default: 5 minutes)
}

// NewVaultBackend creates a new Vault-backed secret store.
func NewVaultBackend(config VaultBackendConfig) (*VaultBackend, error) {
	if config.Address == "" {
		return nil, errors.New("vault_backend: address required")
	}
	if config.Token == "" {
		return nil, errors.New("vault_backend: token required")
	}
	if config.MountPath == "" {
		config.MountPath = "secret"
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &VaultBackend{
		address:    strings.TrimSuffix(config.Address, "/"),
		token:      config.Token,
		namespace:  config.Namespace,
		mountPath:  config.MountPath,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[string]*vaultSecret),
	}, nil
}

// StoreKey stores a key pair in Vault at the specified path.
func (v *VaultBackend) StoreKey(keyID string, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) error {
	path := fmt.Sprintf("v1/%s/data/gauth/keys/%s", v.mountPath, keyID)

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"private_key": base64.StdEncoding.EncodeToString(privateKey),
			"public_key":  base64.StdEncoding.EncodeToString(publicKey),
			"key_id":      keyID,
			"created_at":  time.Now().Unix(),
			"algorithm":   "ed25519",
		},
	}

	return v.vaultRequest("POST", path, data, nil)
}

// RetrieveKey retrieves a key pair from Vault.
func (v *VaultBackend) RetrieveKey(keyID string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	// Check cache first
	v.mu.RLock()
	cached, ok := v.cache[keyID]
	v.mu.RUnlock()

	if ok && time.Since(cached.cachedAt) < cached.ttl {
		return cached.privateKey, cached.publicKey, nil
	}

	// Fetch from Vault
	path := fmt.Sprintf("v1/%s/data/gauth/keys/%s", v.mountPath, keyID)

	var response struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}

	if err := v.vaultRequest("GET", path, nil, &response); err != nil {
		return nil, nil, err
	}

	privKeyB64, ok := response.Data.Data["private_key"].(string)
	if !ok {
		return nil, nil, errors.New("vault_backend: private_key not found")
	}
	pubKeyB64, ok := response.Data.Data["public_key"].(string)
	if !ok {
		return nil, nil, errors.New("vault_backend: public_key not found")
	}

	privBytes, err := base64.StdEncoding.DecodeString(privKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("vault_backend: decode private key: %w", err)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("vault_backend: decode public key: %w", err)
	}

	privKey := ed25519.PrivateKey(privBytes)
	pubKey := ed25519.PublicKey(pubBytes)

	// Update cache
	v.mu.Lock()
	v.cache[keyID] = &vaultSecret{
		privateKey: privKey,
		publicKey:  pubKey,
		keyID:      keyID,
		cachedAt:   time.Now(),
		ttl:        5 * time.Minute,
	}
	v.mu.Unlock()

	return privKey, pubKey, nil
}

// GenerateAndStoreKey generates a new key pair and stores it in Vault.
func (v *VaultBackend) GenerateAndStoreKey() (keyID string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("vault_backend: generate key: %w", err)
	}

	keyID = deriveKeyIDFromPub(pub)

	if err := v.StoreKey(keyID, priv, pub); err != nil {
		return "", err
	}

	// Populate cache
	v.mu.Lock()
	v.cache[keyID] = &vaultSecret{
		privateKey: priv,
		publicKey:  pub,
		keyID:      keyID,
		cachedAt:   time.Now(),
		ttl:        5 * time.Minute,
	}
	v.mu.Unlock()

	return keyID, nil
}

// ListKeys lists all keys stored in Vault under gauth/keys/.
func (v *VaultBackend) ListKeys() ([]string, error) {
	path := fmt.Sprintf("v1/%s/metadata/gauth/keys", v.mountPath)

	var response struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}

	if err := v.vaultRequest("LIST", path, nil, &response); err != nil {
		return nil, err
	}

	return response.Data.Keys, nil
}

// DeleteKey removes a key from Vault and cache.
func (v *VaultBackend) DeleteKey(keyID string) error {
	path := fmt.Sprintf("v1/%s/data/gauth/keys/%s", v.mountPath, keyID)

	if err := v.vaultRequest("DELETE", path, nil, nil); err != nil {
		return err
	}

	// Remove from cache
	v.mu.Lock()
	delete(v.cache, keyID)
	v.mu.Unlock()

	return nil
}

// vaultRequest performs HTTP request to Vault API.
func (v *VaultBackend) vaultRequest(method, path string, requestBody interface{}, responseBody interface{}) error {
	url := fmt.Sprintf("%s/%s", v.address, path)

	var bodyReader io.Reader
	if requestBody != nil {
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("vault_backend: marshal request: %w", err)
		}
		bodyReader = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("vault_backend: create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.token)
	req.Header.Set("Content-Type", "application/json")

	if v.namespace != "" {
		req.Header.Set("X-Vault-Namespace", v.namespace)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vault_backend: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault_backend: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
			return fmt.Errorf("vault_backend: decode response: %w", err)
		}
	}

	return nil
}

// deriveKeyIDFromPub derives a stable key ID from public key.
func deriveKeyIDFromPub(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:6])
}

// ClearCache clears the in-memory cache (useful for testing or after key rotation).
func (v *VaultBackend) ClearCache() {
	v.mu.Lock()
	v.cache = make(map[string]*vaultSecret)
	v.mu.Unlock()
}
