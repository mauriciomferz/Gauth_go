package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// KMSKeyStore implements KeyStore using AWS KMS for key management.
// This implementation stores key metadata in KMS tags and uses KMS for encryption/decryption.
type KMSKeyStore struct {
	client   KMSClient
	keySpec  string
	keyUsage string
	region   string
	kmsKeyID string // Master key for envelope encryption
	ttl      time.Duration
}

// KMSClient interface for AWS KMS operations.
type KMSClient interface {
	CreateKey(ctx context.Context, params *CreateKeyInput) (*CreateKeyOutput, error)
	DescribeKey(ctx context.Context, keyID string) (*DescribeKeyOutput, error)
	ListKeys(ctx context.Context, params *ListKeysInput) (*ListKeysOutput, error)
	ScheduleKeyDeletion(ctx context.Context, keyID string, pendingWindowInDays int) error
	Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
	TagResource(ctx context.Context, keyID string, tags map[string]string) error
	ListResourceTags(ctx context.Context, keyID string) (map[string]string, error)
}

// KMS API structures
type CreateKeyInput struct {
	KeyUsage    string
	KeySpec     string
	Description string
	Tags        map[string]string
}

type CreateKeyOutput struct {
	KeyID string
	Arn   string
}

type DescribeKeyOutput struct {
	KeyID       string
	Arn         string
	KeyUsage    string
	KeySpec     string
	KeyState    string
	Description string
	CreatedAt   time.Time
}

type ListKeysInput struct {
	Limit  int
	Marker string
}

type ListKeysOutput struct {
	Keys       []KeyListEntry
	NextMarker string
	Truncated  bool
}

type KeyListEntry struct {
	KeyID string
	Arn   string
}

// KMSConfig holds KMS configuration.
type KMSConfig struct {
	Region      string
	MasterKeyID string
	KeySpec     string
	KeyUsage    string
	TTL         time.Duration
}

// NewKMSKeyStore creates a new KMS-backed key store.
func NewKMSKeyStore(client KMSClient, config KMSConfig) (*KMSKeyStore, error) {
	if config.KeySpec == "" {
		config.KeySpec = "ECC_NIST_P256" // Default for signing
	}
	if config.KeyUsage == "" {
		config.KeyUsage = "SIGN_VERIFY"
	}
	if config.TTL == 0 {
		config.TTL = 365 * 24 * time.Hour // 1 year default
	}

	store := &KMSKeyStore{
		client:   client,
		keySpec:  config.KeySpec,
		keyUsage: config.KeyUsage,
		region:   config.Region,
		kmsKeyID: config.MasterKeyID,
		ttl:      config.TTL,
	}

	return store, nil
}

// Generate creates a new key in KMS for the tenant.
func (k *KMSKeyStore) Generate(ctx context.Context, tenant string) (string, error) {
	// For Ed25519 keys, we'll generate locally and store encrypted in KMS
	// since KMS doesn't natively support Ed25519
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("key generation failed: %w", err)
	}

	keyID := base64.RawURLEncoding.EncodeToString(pub[:8])

	// Encrypt the private key using KMS
	encryptedPrivateKey, err := k.client.Encrypt(ctx, k.kmsKeyID, priv)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Create metadata structure
	keyMetadata := map[string]interface{}{
		"algorithm":             "Ed25519",
		"created_at":            time.Now().UTC().Format(time.RFC3339),
		"expires_at":            time.Now().Add(k.ttl).UTC().Format(time.RFC3339),
		"encrypted_private_key": base64.StdEncoding.EncodeToString(encryptedPrivateKey),
		"public_key":            base64.StdEncoding.EncodeToString(pub),
		"tenant":                tenant,
		"active":                false,
	}

	// Create a KMS key for this tenant key (using it as a metadata container)
	tags := map[string]string{
		"GauthKeyID":    keyID,
		"GauthTenant":   tenant,
		"GauthKeyType":  "Ed25519",
		"GauthActive":   "false",
		"GauthMetadata": k.encodeMetadata(keyMetadata),
	}

	description := fmt.Sprintf("Gauth Ed25519 key for tenant %s (key ID: %s)", tenant, keyID)

	output, err := k.client.CreateKey(ctx, &CreateKeyInput{
		KeyUsage:    "ENCRYPT_DECRYPT", // Use for metadata storage
		KeySpec:     "SYMMETRIC_DEFAULT",
		Description: description,
		Tags:        tags,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create KMS key: %w", err)
	}

	// Store the KMS key ID mapping (in practice, you'd use a database or other storage)
	// For now, we'll use the KMS key ID as our key identifier
	return output.KeyID, nil
}

// Activate marks a key as active for a tenant.
func (k *KMSKeyStore) Activate(ctx context.Context, tenant, keyID string) error {
	// First, deactivate all existing active keys for the tenant
	if err := k.deactivateAllKeys(ctx, tenant); err != nil {
		return fmt.Errorf("failed to deactivate existing keys: %w", err)
	}

	// Get current tags
	tags, err := k.client.ListResourceTags(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get key tags: %w", err)
	}

	// Update active status
	tags["GauthActive"] = "true"
	tags["GauthActivatedAt"] = time.Now().UTC().Format(time.RFC3339)

	if err := k.client.TagResource(ctx, keyID, tags); err != nil {
		return fmt.Errorf("failed to update key tags: %w", err)
	}

	return nil
}

// Archive marks a key as archived.
func (k *KMSKeyStore) Archive(ctx context.Context, tenant, keyID string) error {
	tags, err := k.client.ListResourceTags(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get key tags: %w", err)
	}

	tags["GauthActive"] = "false"
	tags["GauthArchivedAt"] = time.Now().UTC().Format(time.RFC3339)

	if err := k.client.TagResource(ctx, keyID, tags); err != nil {
		return fmt.Errorf("failed to update key tags: %w", err)
	}

	return nil
}

// GetActive retrieves the currently active key for a tenant.
func (k *KMSKeyStore) GetActive(ctx context.Context, tenant string) (*Key, error) {
	keys, err := k.ListKeys(ctx, tenant)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		// Check if this key is active by examining tags
		tags, err := k.client.ListResourceTags(ctx, key.ID)
		if err != nil {
			continue
		}

		if tags["GauthActive"] == "true" && tags["GauthTenant"] == tenant {
			return key, nil
		}
	}

	return nil, fmt.Errorf("no active key found for tenant %s", tenant)
}

// GetKey retrieves a specific key by ID.
func (k *KMSKeyStore) GetKey(ctx context.Context, tenant, keyID string) (*Key, error) {
	// Get key description and tags
	keyDesc, err := k.client.DescribeKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe key: %w", err)
	}

	tags, err := k.client.ListResourceTags(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key tags: %w", err)
	}

	// Verify this key belongs to the tenant
	if tags["GauthTenant"] != tenant {
		return nil, fmt.Errorf("key does not belong to tenant %s", tenant)
	}

	return k.parseKMSKey(keyID, keyDesc, tags)
}

// ListKeys returns all keys for a tenant.
func (k *KMSKeyStore) ListKeys(ctx context.Context, tenant string) ([]*Key, error) {
	var allKeys []*Key
	var marker string

	for {
		listOutput, err := k.client.ListKeys(ctx, &ListKeysInput{
			Limit:  100,
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list keys: %w", err)
		}

		for _, keyEntry := range listOutput.Keys {
			// Check if this key belongs to the tenant
			tags, err := k.client.ListResourceTags(ctx, keyEntry.KeyID)
			if err != nil {
				continue // Skip keys we can't read
			}

			if tags["GauthTenant"] == tenant {
				keyDesc, err := k.client.DescribeKey(ctx, keyEntry.KeyID)
				if err != nil {
					continue
				}

				key, err := k.parseKMSKey(keyEntry.KeyID, keyDesc, tags)
				if err != nil {
					continue
				}

				allKeys = append(allKeys, key)
			}
		}

		if !listOutput.Truncated {
			break
		}
		marker = listOutput.NextMarker
	}

	return allKeys, nil
}

// Delete schedules a key for deletion in KMS.
func (k *KMSKeyStore) Delete(ctx context.Context, tenant, keyID string) error {
	// Verify key belongs to tenant
	tags, err := k.client.ListResourceTags(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get key tags: %w", err)
	}

	if tags["GauthTenant"] != tenant {
		return fmt.Errorf("key does not belong to tenant %s", tenant)
	}

	// Schedule key deletion (minimum 7 days for KMS)
	if err := k.client.ScheduleKeyDeletion(ctx, keyID, 7); err != nil {
		return fmt.Errorf("failed to schedule key deletion: %w", err)
	}

	return nil
}

// Health checks KMS connectivity.
func (k *KMSKeyStore) Health(ctx context.Context) error {
	// Try to list keys as a health check
	_, err := k.client.ListKeys(ctx, &ListKeysInput{Limit: 1})
	return err
}

// Helper methods

// deactivateAllKeys deactivates all keys for a tenant.
func (k *KMSKeyStore) deactivateAllKeys(ctx context.Context, tenant string) error {
	keys, err := k.ListKeys(ctx, tenant)
	if err != nil {
		return err
	}

	for _, key := range keys {
		tags, err := k.client.ListResourceTags(ctx, key.ID)
		if err != nil {
			continue
		}

		if tags["GauthActive"] == "true" {
			tags["GauthActive"] = "false"
			if err := k.client.TagResource(ctx, key.ID, tags); err != nil {
				// Log error but continue deactivating other keys
				continue
			}
		}
	}

	return nil
}

// parseKMSKey converts KMS key data to Key struct.
func (k *KMSKeyStore) parseKMSKey(keyID string, keyDesc *DescribeKeyOutput, tags map[string]string) (*Key, error) {
	// Decode metadata from tags
	metadataStr := tags["GauthMetadata"]
	if metadataStr == "" {
		return nil, fmt.Errorf("no metadata found for key")
	}

	metadata, err := k.decodeMetadata(metadataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Parse timestamps
	createdAtStr, _ := metadata["created_at"].(string)
	expiresAtStr, _ := metadata["expires_at"].(string)

	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)

	// Decode public key
	publicKeyB64, _ := metadata["public_key"].(string)
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// For the private key, we need to decrypt it using KMS
	encryptedPrivateKeyB64, _ := metadata["encrypted_private_key"].(string)
	encryptedPrivateKey, err := base64.StdEncoding.DecodeString(encryptedPrivateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted private key: %w", err)
	}

	privateKey, err := k.client.Decrypt(context.Background(), encryptedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	algorithm, _ := metadata["algorithm"].(string)

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

// encodeMetadata encodes metadata as a JSON string for tag storage.
func (k *KMSKeyStore) encodeMetadata(metadata map[string]interface{}) string {
	data, _ := json.Marshal(metadata)
	return base64.StdEncoding.EncodeToString(data)
}

// decodeMetadata decodes metadata from a base64-encoded JSON string.
func (k *KMSKeyStore) decodeMetadata(encoded string) (map[string]interface{}, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}
