package crypto

import (
	"context"
	"testing"
	"time"
)

// MockVaultClient facilitates testing of VaultKeyStore logic without a real Vault.
type MockVaultClient struct {
	Data map[string]map[string]interface{}
}

func NewMockVaultClient() *MockVaultClient {
	return &MockVaultClient{
		Data: make(map[string]map[string]interface{}),
	}
}

func (m *MockVaultClient) Read(ctx context.Context, path string) (*VaultResponse, error) {
	if data, ok := m.Data[path]; ok {
		return &VaultResponse{Data: data}, nil
	}
	return nil, nil // Not found
}

func (m *MockVaultClient) Write(ctx context.Context, path string, data map[string]interface{}) (*VaultResponse, error) {
	// If writing data, store it
	if d, ok := data["data"].(map[string]interface{}); ok {
		m.Data[path] = map[string]interface{}{"data": d}

		// Simulate KV v2 listing: if path implies a key, add to metadata keys list
		// Path format: secret/data/gauth/keys/TENANT/KEYID
		// Metadata format: secret/metadata/gauth/keys/TENANT -> { "keys": ["KEYID", ...] }
		if len(path) > 7 { // Simple check
			parts := splitPath(path)
			// Expecting secret/data/...
			if len(parts) >= 2 && parts[1] == "data" {
				// Reconstruct metadata path: secret/metadata/.../tenant
				// And keyID is the last part
				keyID := parts[len(parts)-1]

				// Metadata path is everything up to keyID, replacing "data" with "metadata"
				metaParts := make([]string, 0, len(parts)-1)
				metaParts = append(metaParts, parts[0])
				metaParts = append(metaParts, "metadata")
				metaParts = append(metaParts, parts[2:len(parts)-1]...)

				metaPath := joinPath(metaParts)

				if m.Data[metaPath] == nil {
					m.Data[metaPath] = map[string]interface{}{"keys": []interface{}{}}
				}

				existingKeys, _ := m.Data[metaPath]["keys"].([]interface{})
				found := false
				for _, k := range existingKeys {
					if k.(string) == keyID {
						found = true
						break
					}
				}
				if !found {
					m.Data[metaPath]["keys"] = append(existingKeys, keyID)
				}
			}
		}

	} else if d, ok := data["data"]; ok {
		if mapData, ok := d.(map[string]interface{}); ok {
			m.Data[path] = map[string]interface{}{"data": mapData}
		}
	} else {
		m.Data[path] = data
	}

	// Handle Transit Encrypt/Decrypt mock
	parts := splitPath(path)
	var op string
	for _, p := range parts {
		if p == "encrypt" || p == "decrypt" {
			op = p
			break
		}
	}

	if op == "encrypt" {
		plaintext, _ := data["plaintext"].(string)
		return &VaultResponse{
			Data: map[string]interface{}{
				"ciphertext": "vault:v1:" + plaintext,
			},
		}, nil
	} else if op == "decrypt" {
		ciphertext, _ := data["ciphertext"].(string)
		// Strip vault:v1: prefix
		if len(ciphertext) > 9 && ciphertext[:9] == "vault:v1:" {
			ciphertext = ciphertext[9:]
		}
		return &VaultResponse{
			Data: map[string]interface{}{
				"plaintext": ciphertext,
			},
		}, nil
	}

	return &VaultResponse{Data: m.Data[path]}, nil
}

func (m *MockVaultClient) Delete(ctx context.Context, path string) error {
	delete(m.Data, path)
	return nil
}

func (m *MockVaultClient) Health(ctx context.Context) error {
	return nil
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

func joinPath(parts []string) string {
	res := ""
	for i, p := range parts {
		if i > 0 {
			res += "/"
		}
		res += p
	}
	return res
}

func TestVaultKeyStore_Generate(t *testing.T) {
	mockClient := NewMockVaultClient()

	store := &VaultKeyStore{
		client:      mockClient,
		kvPath:      "secret",
		transitPath: "transit",
		tokenTTL:    time.Hour,
	}

	ctx := context.Background()
	tenant := "test-tenant"

	keyID, err := store.Generate(ctx, tenant)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if keyID == "" {
		t.Error("Generate returned empty keyID")
	}

	// Verify key stored
	key, err := store.GetKey(ctx, tenant, keyID)
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}

	if key.ID != keyID {
		t.Errorf("Expected keyID %s, got %s", keyID, key.ID)
	}

	if key.Alg != "Ed25519" {
		t.Errorf("Expected algorithm Ed25519, got %s", key.Alg)
	}
}

func TestVaultKeyStore_Archive(t *testing.T) {
	mockClient := NewMockVaultClient()
	store := &VaultKeyStore{
		client:   mockClient,
		kvPath:   "secret",
		tokenTTL: time.Hour,
	}

	ctx := context.Background()
	tenant := "test-tenant"
	keyID, _ := store.Generate(ctx, tenant)

	// Activate first
	if err := store.Activate(ctx, tenant, keyID); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}

	// Verify it is active
	activeKey, err := store.GetActive(ctx, tenant)
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if activeKey.ID != keyID {
		t.Errorf("Expected active key %s, got %s", keyID, activeKey.ID)
	}

	// Archive
	if err := store.Archive(ctx, tenant, keyID); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Verify no longer active
	_, err = store.GetActive(ctx, tenant)
	if err == nil {
		t.Error("Expected error from GetActive after archiving, got nil")
	}
}

func TestVaultKeyStore_Activate(t *testing.T) {
	mockClient := NewMockVaultClient()
	store := &VaultKeyStore{
		client:   mockClient,
		kvPath:   "secret",
		tokenTTL: time.Hour,
	}

	ctx := context.Background()
	tenant := "test-tenant-2"
	keyID, _ := store.Generate(ctx, tenant)

	if err := store.Activate(ctx, tenant, keyID); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}

	active, err := store.GetActive(ctx, tenant)
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if active.ID != keyID {
		t.Errorf("Expected active key %s, got %s", keyID, active.ID)
	}
}
