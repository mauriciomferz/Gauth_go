// Package agentauth - JWE Service Tests
// Tests for JWE encryption/decryption of Extended Tokens
package agentauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper: generate temporary RSA key pair
func generateTestRSAKeys(t *testing.T, bits int) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err, "Failed to generate RSA key pair")
	return privateKey, &privateKey.PublicKey
}

// Test helper: create temporary key files
func createTempKeyFiles(t *testing.T, privateKey *rsa.PrivateKey) (string, string) {
	tempDir := t.TempDir()

	privKeyPath := filepath.Join(tempDir, "private.pem")
	pubKeyPath := filepath.Join(tempDir, "public.pem")

	err := SaveRSAPrivateKey(privateKey, privKeyPath)
	require.NoError(t, err, "Failed to save private key")

	err = SaveRSAPublicKey(&privateKey.PublicKey, pubKeyPath)
	require.NoError(t, err, "Failed to save public key")

	return pubKeyPath, privKeyPath
}

// Test: JWE configuration validation
func TestJWEConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *JWEConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "disabled config is valid",
			config:  DisabledJWEConfig(),
			wantErr: false,
		},
		{
			name: "valid RSA config",
			config: &JWEConfig{
				Enabled:        true,
				Algorithm:      "RSA-OAEP-256",
				Encryption:     "A256GCM",
				PublicKeyPath:  "testdata/public.pem",
				PrivateKeyPath: "testdata/private.pem",
				KeyID:          "test-key-2025-11",
			},
			wantErr: true, // Files don't exist
			errMsg:  "public key file not found",
		},
		{
			name: "valid symmetric config",
			config: &JWEConfig{
				Enabled:      true,
				Algorithm:    "A256KW",
				Encryption:   "A256GCM",
				SymmetricKey: make([]byte, 32),
				KeyID:        "test-key-2025-11",
			},
			wantErr: false,
		},
		{
			name: "invalid algorithm",
			config: &JWEConfig{
				Enabled:    true,
				Algorithm:  "INVALID",
				Encryption: "A256GCM",
				KeyID:      "test",
			},
			wantErr: true,
			errMsg:  "unsupported JWE algorithm",
		},
		{
			name: "invalid encryption",
			config: &JWEConfig{
				Enabled:    true,
				Algorithm:  "A256KW",
				Encryption: "INVALID",
				KeyID:      "test",
			},
			wantErr: true,
			errMsg:  "unsupported JWE encryption",
		},
		{
			name: "missing KeyID",
			config: &JWEConfig{
				Enabled:      true,
				Algorithm:    "A256KW",
				Encryption:   "A256GCM",
				SymmetricKey: make([]byte, 32),
			},
			wantErr: true,
			errMsg:  "KeyID is required",
		},
		{
			name: "symmetric key wrong size",
			config: &JWEConfig{
				Enabled:      true,
				Algorithm:    "A256KW",
				Encryption:   "A256GCM",
				SymmetricKey: make([]byte, 16), // Should be 32
				KeyID:        "test",
			},
			wantErr: true,
			errMsg:  "SymmetricKey must be 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test: RSA key generation
func TestGenerateRSAKeyPair(t *testing.T) {
	tests := []struct {
		name    string
		bits    int
		wantErr bool
	}{
		{"2048 bits", 2048, false},
		{"3072 bits", 3072, false},
		{"4096 bits", 4096, false},
		{"1024 bits (too small)", 1024, true},
		{"8192 bits (too large)", 8192, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKey, err := GenerateRSAKeyPair(tt.bits)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, privateKey)
				assert.Equal(t, tt.bits/8, privateKey.Size())
			}
		})
	}
}

// Test: RSA key save/load
func TestSaveLoadRSAKeys(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeys(t, 2048)

	tempDir := t.TempDir()
	privKeyPath := filepath.Join(tempDir, "private.pem")
	pubKeyPath := filepath.Join(tempDir, "public.pem")

	// Save keys
	err := SaveRSAPrivateKey(privateKey, privKeyPath)
	require.NoError(t, err)

	err = SaveRSAPublicKey(publicKey, pubKeyPath)
	require.NoError(t, err)

	// Verify files exist
	assert.FileExists(t, privKeyPath)
	assert.FileExists(t, pubKeyPath)

	// Load keys back
	loadedPrivKey, err := LoadRSAPrivateKey(privKeyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey.N, loadedPrivKey.N)

	loadedPubKey, err := LoadRSAPublicKey(pubKeyPath)
	require.NoError(t, err)
	assert.Equal(t, publicKey.N, loadedPubKey.N)
}

// Test: JWE Service creation with RSA
func TestNewJWEService_RSA(t *testing.T) {
	privateKey, _ := generateTestRSAKeys(t, 2048)
	pubKeyPath, privKeyPath := createTempKeyFiles(t, privateKey)

	config := &JWEConfig{
		Enabled:        true,
		Algorithm:      "RSA-OAEP-256",
		Encryption:     "A256GCM",
		PublicKeyPath:  pubKeyPath,
		PrivateKeyPath: privKeyPath,
		KeyID:          "test-rsa-2025-11",
	}

	service, err := NewJWEService(config)
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.True(t, service.IsEnabled())

	// Verify public key loaded
	pubKey, err := service.GetPublicKey(context.Background(), "test-rsa-2025-11")
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
}

// Test: JWE Service creation with symmetric key
func TestNewJWEService_Symmetric(t *testing.T) {
	config := DevelopmentJWEConfig()

	service, err := NewJWEService(config)
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.True(t, service.IsEnabled())
}

// Test: JWE encryption/decryption with RSA
func TestJWEService_EncryptDecrypt_RSA(t *testing.T) {
	privateKey, _ := generateTestRSAKeys(t, 2048)
	pubKeyPath, privKeyPath := createTempKeyFiles(t, privateKey)

	config := &JWEConfig{
		Enabled:        true,
		Algorithm:      "RSA-OAEP-256",
		Encryption:     "A256GCM",
		PublicKeyPath:  pubKeyPath,
		PrivateKeyPath: privKeyPath,
		KeyID:          "test-rsa-2025-11",
	}

	service, err := NewJWEService(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test JWT string (simplified)
	jwtString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	// Encrypt
	jweString, err := service.EncryptToken(ctx, jwtString)
	require.NoError(t, err)
	assert.NotEmpty(t, jweString)
	assert.NotEqual(t, jwtString, jweString)

	// Verify JWE format (5 parts)
	assert.True(t, IsJWE(jweString), "Encrypted token should have JWE format")

	// Decrypt
	decrypted, err := service.DecryptToken(ctx, jweString)
	require.NoError(t, err)
	assert.Equal(t, jwtString, decrypted)
}

// Test: JWE encryption/decryption with symmetric key
func TestJWEService_EncryptDecrypt_Symmetric(t *testing.T) {
	config := DevelopmentJWEConfig()

	service, err := NewJWEService(config)
	require.NoError(t, err)

	ctx := context.Background()

	jwtString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	// Encrypt
	jweString, err := service.EncryptToken(ctx, jwtString)
	require.NoError(t, err)
	assert.NotEmpty(t, jweString)
	assert.True(t, IsJWE(jweString))

	// Decrypt
	decrypted, err := service.DecryptToken(ctx, jweString)
	require.NoError(t, err)
	assert.Equal(t, jwtString, decrypted)
}

// Test: JWE encryption error cases
func TestJWEService_EncryptErrors(t *testing.T) {
	config := DevelopmentJWEConfig()
	service, err := NewJWEService(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Empty JWT string
	_, err = service.EncryptToken(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// Test: JWE decryption error cases
func TestJWEService_DecryptErrors(t *testing.T) {
	config := DevelopmentJWEConfig()
	service, err := NewJWEService(config)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name      string
		jweString string
		errMsg    string
	}{
		{
			name:      "empty JWE string",
			jweString: "",
			errMsg:    "cannot be empty",
		},
		{
			name:      "invalid JWE format",
			jweString: "invalid.jwe.format",
			errMsg:    "failed to parse JWE",
		},
		{
			name:      "malformed JWE",
			jweString: "eyJhbGciOiJSU0EtT0FFUCIsImVuYyI6IkEyNTZHQ00ifQ.invalid.invalid.invalid.invalid",
			errMsg:    "key ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.DecryptToken(ctx, tt.jweString)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

// Test: JWE disabled service
func TestJWEService_Disabled(t *testing.T) {
	config := DisabledJWEConfig()
	service, err := NewJWEService(config)
	require.NoError(t, err)
	assert.False(t, service.IsEnabled())

	ctx := context.Background()

	// Encryption should fail when disabled
	_, err = service.EncryptToken(ctx, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")

	// Decryption should fail when disabled
	_, err = service.DecryptToken(ctx, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

// Test: IsJWE helper function
func TestIsJWE(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		want        bool
	}{
		{
			name:        "JWE token (5 parts)",
			tokenString: "header.encrypted_key.iv.ciphertext.tag",
			want:        true,
		},
		{
			name:        "JWT token (3 parts)",
			tokenString: "header.payload.signature",
			want:        false,
		},
		{
			name:        "invalid token (2 parts)",
			tokenString: "part1.part2",
			want:        false,
		},
		{
			name:        "empty string",
			tokenString: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsJWE(tt.tokenString)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test: Key rotation
func TestJWEService_RotateKeys(t *testing.T) {
	privateKey, _ := generateTestRSAKeys(t, 2048)
	pubKeyPath, privKeyPath := createTempKeyFiles(t, privateKey)

	config := &JWEConfig{
		Enabled:        true,
		Algorithm:      "RSA-OAEP-256",
		Encryption:     "A256GCM",
		PublicKeyPath:  pubKeyPath,
		PrivateKeyPath: privKeyPath,
		KeyID:          "test-rotation-2025-11",
	}

	service, err := NewJWEService(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Rotate keys (in this simple implementation, just reloads same keys)
	err = service.RotateKeys(ctx)
	assert.NoError(t, err)

	// Verify keys still work
	jwtString := "test.jwt.token"
	jweString, err := service.EncryptToken(ctx, jwtString)
	require.NoError(t, err)

	decrypted, err := service.DecryptToken(ctx, jweString)
	require.NoError(t, err)
	assert.Equal(t, jwtString, decrypted)
}

// Benchmark: JWE encryption
func BenchmarkJWEService_Encrypt(b *testing.B) {
	config := DevelopmentJWEConfig()
	service, _ := NewJWEService(config)
	ctx := context.Background()
	jwtString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.EncryptToken(ctx, jwtString)
	}
}

// Benchmark: JWE decryption
func BenchmarkJWEService_Decrypt(b *testing.B) {
	config := DevelopmentJWEConfig()
	service, _ := NewJWEService(config)
	ctx := context.Background()
	jwtString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	jweString, _ := service.EncryptToken(ctx, jwtString)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.DecryptToken(ctx, jweString)
	}
}
