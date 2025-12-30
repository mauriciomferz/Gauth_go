// Package agentauth - JWE Integration Benchmarks (Phase 2)
// Performance and integration benchmarks for JWE encryption
package agentauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkJWEIntegration_FullCycle benchmarks the complete JWT vs JWE lifecycle
func BenchmarkJWEIntegration_FullCycle(b *testing.B) {
	ctx := context.Background()

	// Create test JWT
	testJWT := createTestJWT(b)

	// Setup JWE service
	tmpDir, _ := os.MkdirTemp("", "jwe-bench-*")
	defer os.RemoveAll(tmpDir)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	_ = SaveRSAPrivateKey(privateKey, privateKeyPath)
	_ = SaveRSAPublicKey(&privateKey.PublicKey, publicKeyPath)

	jweConfig := ProductionJWEConfig(publicKeyPath, privateKeyPath, "bench-key")
	jweService, _ := NewJWEService(jweConfig)

	b.ResetTimer()
	b.Run("Encrypt+Decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Encrypt
			jweToken, _ := jweService.EncryptToken(ctx, testJWT)
			// Decrypt
			_, _ = jweService.DecryptToken(ctx, jweToken)
		}
	})
}

// BenchmarkJWEIntegration_EncryptOnly benchmarks encryption only
func BenchmarkJWEIntegration_EncryptOnly(b *testing.B) {
	ctx := context.Background()
	testJWT := createTestJWT(b)

	tmpDir, _ := os.MkdirTemp("", "jwe-bench-*")
	defer os.RemoveAll(tmpDir)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	_ = SaveRSAPrivateKey(privateKey, privateKeyPath)
	_ = SaveRSAPublicKey(&privateKey.PublicKey, publicKeyPath)

	jweConfig := ProductionJWEConfig(publicKeyPath, privateKeyPath, "bench-key")
	jweService, _ := NewJWEService(jweConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jweService.EncryptToken(ctx, testJWT)
	}
}

// BenchmarkJWEIntegration_DecryptOnly benchmarks decryption only
func BenchmarkJWEIntegration_DecryptOnly(b *testing.B) {
	ctx := context.Background()
	testJWT := createTestJWT(b)

	tmpDir, _ := os.MkdirTemp("", "jwe-bench-*")
	defer os.RemoveAll(tmpDir)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	_ = SaveRSAPrivateKey(privateKey, privateKeyPath)
	_ = SaveRSAPublicKey(&privateKey.PublicKey, publicKeyPath)

	jweConfig := ProductionJWEConfig(publicKeyPath, privateKeyPath, "bench-key")
	jweService, _ := NewJWEService(jweConfig)

	// Pre-encrypt token
	jweToken, _ := jweService.EncryptToken(ctx, testJWT)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jweService.DecryptToken(ctx, jweToken)
	}
}

// TestJWEIntegration_TokenFormat validates JWE format
func TestJWEIntegration_TokenFormat(t *testing.T) {
	ctx := context.Background()
	testJWT := createTestJWT(t)

	tmpDir, _ := os.MkdirTemp("", "jwe-format-*")
	defer os.RemoveAll(tmpDir)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	_ = SaveRSAPrivateKey(privateKey, privateKeyPath)
	_ = SaveRSAPublicKey(&privateKey.PublicKey, publicKeyPath)

	jweConfig := ProductionJWEConfig(publicKeyPath, privateKeyPath, "format-key")
	jweService, _ := NewJWEService(jweConfig)

	// Encrypt
	jweToken, err := jweService.EncryptToken(ctx, testJWT)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Validate JWE format (5 parts)
	if !IsJWE(jweToken) {
		t.Error("Expected JWE format (5 parts)")
	}

	parts := strings.Split(jweToken, ".")
	if len(parts) != 5 {
		t.Errorf("Expected 5 JWE parts, got %d", len(parts))
	}

	// Decrypt
	decrypted, err := jweService.DecryptToken(ctx, jweToken)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// Validate decrypted token matches original
	if decrypted != testJWT {
		t.Error("Decrypted token doesn't match original")
	}

	t.Logf("✅ JWE format test passed: %d bytes", len(jweToken))
}

// TestJWEIntegration_TokenSizeOverhead measures encryption overhead
func TestJWEIntegration_TokenSizeOverhead(t *testing.T) {
	ctx := context.Background()
	testJWT := createTestJWT(t)

	tmpDir, _ := os.MkdirTemp("", "jwe-size-*")
	defer os.RemoveAll(tmpDir)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	_ = SaveRSAPrivateKey(privateKey, privateKeyPath)
	_ = SaveRSAPublicKey(&privateKey.PublicKey, publicKeyPath)

	jweConfig := ProductionJWEConfig(publicKeyPath, privateKeyPath, "size-key")
	jweService, _ := NewJWEService(jweConfig)

	jweToken, _ := jweService.EncryptToken(ctx, testJWT)

	jwtSize := len(testJWT)
	jweSize := len(jweToken)
	overhead := jweSize - jwtSize
	overheadPercent := (float64(overhead) / float64(jwtSize)) * 100

	t.Logf("JWT size: %d bytes", jwtSize)
	t.Logf("JWE size: %d bytes", jweSize)
	t.Logf("Overhead: %d bytes (%.1f%%)", overhead, overheadPercent)

	if jweSize <= jwtSize {
		t.Error("Expected JWE to be larger than JWT")
	}

	// Typical overhead: 30-60%
	if overheadPercent > 100 {
		t.Logf("Warning: High overhead %.1f%%", overheadPercent)
	}
}

// TestJWEIntegration_ErrorHandling tests error scenarios
func TestJWEIntegration_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tmpDir, _ := os.MkdirTemp("", "jwe-error-*")
	defer os.RemoveAll(tmpDir)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	_ = SaveRSAPrivateKey(privateKey, privateKeyPath)
	_ = SaveRSAPublicKey(&privateKey.PublicKey, publicKeyPath)

	jweConfig := ProductionJWEConfig(publicKeyPath, privateKeyPath, "error-key")
	jweService, _ := NewJWEService(jweConfig)

	// Test 1: Malformed JWE
	_, err := jweService.DecryptToken(ctx, "malformed.jwe.token")
	if err == nil {
		t.Error("Expected error for malformed JWE")
	}

	// Test 2: Tampered JWE
	testJWT := createTestJWT(t)
	validJWE, _ := jweService.EncryptToken(ctx, testJWT)
	tamperedJWE := validJWE[:len(validJWE)-10] + "TAMPERED!!"

	_, err = jweService.DecryptToken(ctx, tamperedJWE)
	if err == nil {
		t.Error("Expected error for tampered JWE")
	}

	t.Logf("✅ Error handling test passed")
}

// TestJWEIntegration_KeyRotation tests basic key rotation
func TestJWEIntegration_KeyRotation(t *testing.T) {
	ctx := context.Background()
	testJWT := createTestJWT(t)

	tmpDir, _ := os.MkdirTemp("", "jwe-rotation-*")
	defer os.RemoveAll(tmpDir)

	// Key 1
	key1, _ := rsa.GenerateKey(rand.Reader, 2048)
	key1PrivPath := filepath.Join(tmpDir, "key1_private.pem")
	key1PubPath := filepath.Join(tmpDir, "key1_public.pem")
	_ = SaveRSAPrivateKey(key1, key1PrivPath)
	_ = SaveRSAPublicKey(&key1.PublicKey, key1PubPath)

	config1 := ProductionJWEConfig(key1PubPath, key1PrivPath, "key-v1")
	service1, _ := NewJWEService(config1)

	// Encrypt with key1
	jweToken, _ := service1.EncryptToken(ctx, testJWT)

	// Decrypt with key1
	decrypted, err := service1.DecryptToken(ctx, jweToken)
	if err != nil {
		t.Fatalf("Decryption with key1 failed: %v", err)
	}
	if decrypted != testJWT {
		t.Error("Decrypted token doesn't match")
	}

	// Key 2
	key2, _ := rsa.GenerateKey(rand.Reader, 2048)
	key2PrivPath := filepath.Join(tmpDir, "key2_private.pem")
	key2PubPath := filepath.Join(tmpDir, "key2_public.pem")
	_ = SaveRSAPrivateKey(key2, key2PrivPath)
	_ = SaveRSAPublicKey(&key2.PublicKey, key2PubPath)

	config2 := ProductionJWEConfig(key2PubPath, key2PrivPath, "key-v2")
	service2, _ := NewJWEService(config2)

	// Try to decrypt key1 token with key2 service (should fail)
	_, err = service2.DecryptToken(ctx, jweToken)
	if err == nil {
		t.Error("Expected error: key2 should not decrypt key1 token")
	}

	t.Logf("✅ Key rotation test passed")
}

// Helper: Create test JWT token
func createTestJWT(t testing.TB) string {
	// Simulated JWT with realistic size (ExtendedToken)
	return strings.Repeat("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.", 3) +
		"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3QgVXNlciIsImNsaWVudF9pZCI6InRlc3QtY2xpZW50IiwicG93ZXJfb2ZfYXR0b3JuZXkiOnsiaWQiOiJwb2EtdGVzdCJ9LCJhdXRob3JpemF0aW9uX2NoYWluIjp7ImxpbmtzIjpbXX0sInNjb3BlIjpbInJlYWQ6YWNjb3VudHMiLCJ3cml0ZTpwYXltZW50cyJdLCJpc3MiOiJodHRwczovL2dhdXRoLmV4YW1wbGUuY29tIiwiaWF0IjoxNjE2MjM5MDIyLCJleHAiOjE2MTYyNDI2MjJ9." +
		"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}
