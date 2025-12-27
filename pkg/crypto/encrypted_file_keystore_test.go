package crypto

import (
	"context"
	"os"
	"testing"
)

func TestEncryptedFileKeyStore(t *testing.T) {
	// Generate master key
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("failed to generate master key: %v", err)
	}

	// Create temp file for keystore
	tmpDir := t.TempDir()
	keystorePath := tmpDir + "/keys"
	masterKeyPath := tmpDir + "/master.key"

	// Save master key
	if err := os.WriteFile(masterKeyPath, masterKey, 0600); err != nil {
		t.Fatalf("failed to write master key: %v", err)
	}

	// Create encrypted keystore
	store, err := NewEncryptedFileKeyStore(EncryptedStoreConfig{
		FilePath:      keystorePath,
		MasterKeyPath: masterKeyPath,
	})
	if err != nil {
		t.Fatalf("failed to create encrypted keystore: %v", err)
	}

	ctx := context.Background()
	tenant := "test-tenant"

	// Generate a key
	keyID, err := store.Generate(ctx, tenant)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Reload and verify
	store2, err := NewEncryptedFileKeyStore(EncryptedStoreConfig{
		FilePath:      keystorePath,
		MasterKeyPath: masterKeyPath,
	})
	if err != nil {
		t.Fatalf("failed to reload keystore: %v", err)
	}

	key, err := store2.GetKey(ctx, tenant, keyID)
	if err != nil {
		t.Fatalf("failed to get key: %v", err)
	}

	if key.ID != keyID {
		t.Errorf("expected key ID %s, got %s", keyID, key.ID)
	}
}

func TestEncryptedFileKeyStore_EnvVar(t *testing.T) {
	// Set master key in environment
	os.Setenv("GAUTH_MASTER_KEY", "test-passphrase-for-encryption")
	defer os.Unsetenv("GAUTH_MASTER_KEY")

	tmpDir := t.TempDir()
	keystorePath := tmpDir + "/keys"

	// Create encrypted keystore using env var
	store, err := NewEncryptedFileKeyStore(EncryptedStoreConfig{
		FilePath:     keystorePath,
		MasterKeyEnv: "GAUTH_MASTER_KEY",
	})
	if err != nil {
		t.Fatalf("failed to create encrypted keystore: %v", err)
	}

	ctx := context.Background()
	tenant := "test-tenant"

	// Generate
	keyID, err := store.Generate(ctx, tenant)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Reload
	store2, err := NewEncryptedFileKeyStore(EncryptedStoreConfig{
		FilePath:     keystorePath,
		MasterKeyEnv: "GAUTH_MASTER_KEY",
	})
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}

	// Verify
	key, err := store2.GetKey(ctx, tenant, keyID)
	if err != nil {
		t.Fatalf("failed to get key: %v", err)
	}

	if key.ID != keyID {
		t.Errorf("key ID mismatch")
	}
}

func TestMasterKeyGeneration(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("failed to generate master key: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(key))
	}

	// Encode and verify
	encoded := EncodeMasterKey(key)
	if encoded == "" {
		t.Error("encoded key is empty")
	}

	t.Logf("Generated master key (base64): %s", encoded)
}

func TestEncryptDecrypt(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte("sensitive data to encrypt")

	// Encrypt
	ciphertext, err := encryptBytes(masterKey, plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if len(ciphertext) <= len(plaintext) {
		t.Error("ciphertext should be longer than plaintext (includes nonce and tag)")
	}

	// Decrypt
	decrypted, err := decryptBytes(masterKey, ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted data doesn't match original")
	}
}
