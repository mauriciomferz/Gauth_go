package secret

import (
	"context"
	"strings"
	"testing"
)

func TestEncryptedProviderCRUD(t *testing.T) {
	backend := NewMemory()
	passphrase := "test-passphrase-with-sufficient-length"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	ctx := context.Background()
	secretKey := "db/password"
	secretValue := "super-secret-password-123"

	// Set encrypted secret
	if err2 := enc.Set(ctx, secretKey, secretValue); err2 != nil {
		t.Fatalf("Set: %v", err2)
	}

	// Get decrypted secret
	retrieved, err := enc.Get(ctx, secretKey)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if retrieved != secretValue {
		t.Errorf("expected %q, got %q", secretValue, retrieved)
	}

	// Verify backend stores encrypted data (not plaintext)
	backendValue, err := backend.Get(ctx, secretKey)
	if err != nil {
		t.Fatalf("backend Get: %v", err)
	}
	if backendValue == secretValue {
		t.Error("backend contains plaintext secret (encryption failed)")
	}
	if !strings.Contains(backendValue, secretValue) {
		// Good: encrypted value doesn't contain plaintext
	} else {
		t.Error("encrypted value contains plaintext substring")
	}

	// Delete
	if err := enc.Delete(ctx, secretKey); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := enc.Get(ctx, secretKey); err == nil {
		t.Error("expected not found after delete")
	}
}

func TestEncryptedProviderList(t *testing.T) {
	backend := NewMemory()
	passphrase := "list-test-passphrase-sufficient-length"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	ctx := context.Background()
	secrets := map[string]string{
		"api/key1":    "secret1",
		"api/key2":    "secret2",
		"db/password": "dbpass",
		"db/username": "dbuser",
	}

	for k, v := range secrets {
		if err2 := enc.Set(ctx, k, v); err2 != nil {
			t.Fatalf("Set %s: %v", k, err2)
		}
	}

	// List all
	all, err := enc.List(ctx, "")
	if err != nil || len(all) != 4 {
		t.Errorf("List all: expected 4, got %d (err=%v)", len(all), err)
	}

	// List with prefix
	apiKeys, err := enc.List(ctx, "api/")
	if err != nil || len(apiKeys) != 2 {
		t.Errorf("List api/: expected 2, got %d (err=%v)", len(apiKeys), err)
	}

	dbKeys, err := enc.List(ctx, "db/")
	if err != nil || len(dbKeys) != 2 {
		t.Errorf("List db/: expected 2, got %d (err=%v)", len(dbKeys), err)
	}
}

func TestEncryptedProviderIfNotExists(t *testing.T) {
	backend := NewMemory()
	passphrase := "ifnotexists-test-passphrase-length"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	ctx := context.Background()
	key := "unique-key"

	// First set should succeed
	if err2 := enc.Set(ctx, key, "value1", IfNotExists()); err2 != nil {
		t.Fatalf("first Set with IfNotExists: %v", err2)
	}

	// Second set with IfNotExists should fail
	if err2 := enc.Set(ctx, key, "value2", IfNotExists()); err2 == nil {
		t.Error("expected error on second Set with IfNotExists")
	}

	// Value should still be value1
	val, err := enc.Get(ctx, key)
	if err != nil || val != "value1" {
		t.Errorf("expected value1, got %q (err=%v)", val, err)
	}

	// Set without flag should overwrite
	if err := enc.Set(ctx, key, "value2"); err != nil {
		t.Fatalf("overwrite Set: %v", err)
	}
	val, _ = enc.Get(ctx, key)
	if val != "value2" {
		t.Errorf("expected value2 after overwrite, got %q", val)
	}
}

func TestEncryptedProviderName(t *testing.T) {
	backend := NewMemory()
	passphrase := "name-test-passphrase-sufficient"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	name := enc.Name()
	expected := "encrypted(memory)"
	if name != expected {
		t.Errorf("expected %q, got %q", expected, name)
	}
}

func TestEncryptedProviderWeakPassphrase(t *testing.T) {
	backend := NewMemory()
	shortPass := "short"
	_, err := NewEncrypted(backend, shortPass)
	if err == nil {
		t.Error("expected error for short passphrase")
	}
}

func TestEncryptedProviderNilBackend(t *testing.T) {
	_, err := NewEncrypted(nil, "valid-passphrase-length")
	if err == nil {
		t.Error("expected error for nil backend")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	backend := NewMemory()
	passphrase := "roundtrip-test-passphrase-sufficient"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	testCases := []string{
		"simple",
		"with spaces and special chars: !@#$%^&*()",
		"unicode: 你好世界 🔐🔑",
		"multiline\nwith\ntabs\tand\nnewlines",
		"", // empty string
	}

	for i, plaintext := range testCases {
		encrypted, err := enc.encrypt(plaintext)
		if err != nil {
			t.Errorf("case %d: encrypt: %v", i, err)
			continue
		}

		decrypted, err := enc.decrypt(encrypted)
		if err != nil {
			t.Errorf("case %d: decrypt: %v", i, err)
			continue
		}

		if decrypted != plaintext {
			t.Errorf("case %d: expected %q, got %q", i, plaintext, decrypted)
		}
	}
}

func TestEncryptedProviderTamperDetection(t *testing.T) {
	backend := NewMemory()
	passphrase := "tamper-test-passphrase-sufficient"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	ctx := context.Background()
	key := "tamper-test"
	value := "original-secret"

	if err2 := enc.Set(ctx, key, value); err2 != nil {
		t.Fatalf("Set: %v", err2)
	}

	// Get encrypted value from backend
	encryptedValue, err := backend.Get(ctx, key)
	if err != nil {
		t.Fatalf("backend Get: %v", err)
	}

	// Tamper with encrypted value
	tampered := encryptedValue[:len(encryptedValue)-4] + "XXXX"
	if err2 := backend.Set(ctx, key, tampered); err2 != nil {
		t.Fatalf("backend Set tampered: %v", err2)
	}

	// Attempt to retrieve should fail
	_, err = enc.Get(ctx, key)
	if err == nil {
		t.Error("expected decryption failure for tampered data")
	}
}

func TestEncryptedProviderDifferentKeys(t *testing.T) {
	backend1 := NewMemory()
	backend2 := NewMemory()

	pass1 := "first-passphrase-sufficient"
	pass2 := "second-passphrase-different"

	enc1, err := NewEncrypted(backend1, pass1)
	if err != nil {
		t.Fatalf("NewEncrypted 1: %v", err)
	}

	enc2, err := NewEncrypted(backend2, pass2)
	if err != nil {
		t.Fatalf("NewEncrypted 2: %v", err)
	}

	secret := "shared-secret-value"

	// Encrypt with first key
	encrypted1, err := enc1.encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt1: %v", err)
	}

	// Encrypt with second key
	encrypted2, err := enc2.encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt2: %v", err)
	}

	// Encrypted values should differ
	if encrypted1 == encrypted2 {
		t.Error("different keys produced identical ciphertext")
	}

	// Each provider can only decrypt its own
	_, err = enc1.decrypt(encrypted2)
	if err == nil {
		t.Error("provider 1 decrypted provider 2's ciphertext")
	}

	_, err = enc2.decrypt(encrypted1)
	if err == nil {
		t.Error("provider 2 decrypted provider 1's ciphertext")
	}
}

func TestEncryptedProviderNonceUniqueness(t *testing.T) {
	backend := NewMemory()
	passphrase := "nonce-test-passphrase-sufficient"
	enc, err := NewEncrypted(backend, passphrase)
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}

	// Encrypt same plaintext multiple times
	plaintext := "repeated-encryption"
	encrypted := make(map[string]bool)

	for i := 0; i < 100; i++ {
		ciphertext, err := enc.encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt iteration %d: %v", i, err)
		}

		if encrypted[ciphertext] {
			t.Errorf("duplicate ciphertext at iteration %d (nonce reuse)", i)
		}
		encrypted[ciphertext] = true

		// Verify decryption still works
		decrypted, err := enc.decrypt(ciphertext)
		if err != nil {
			t.Errorf("decrypt iteration %d: %v", i, err)
		}
		if decrypted != plaintext {
			t.Errorf("iteration %d: expected %q, got %q", i, plaintext, decrypted)
		}
	}

	if len(encrypted) != 100 {
		t.Errorf("expected 100 unique ciphertexts, got %d", len(encrypted))
	}
}
