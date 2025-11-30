// Package gauth - JWE (JSON Web Encryption) Configuration
// Implements Security Hardening as identified in JWE_ENCRYPTION_ASSESSMENT.md
package gauth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	algorithmRSAOAEP256 = "RSA-OAEP-256"
)

// JWEConfig defines configuration for JWE token encryption
type JWEConfig struct {
	// Enabled controls whether JWE encryption is active
	// Default: true (encryption enabled)
	// Can be set to false for development/debugging
	Enabled bool

	// Algorithm specifies the key encryption algorithm
	// Supported: "RSA-OAEP-256" (recommended), "A256KW" (symmetric)
	// Default: "RSA-OAEP-256"
	Algorithm string

	// Encryption specifies the content encryption algorithm
	// Supported: "A256GCM" (recommended), "A128GCM"
	// Default: "A256GCM"
	Encryption string

	// PublicKeyPath is the path to the RSA public key PEM file
	// Used for encrypting tokens (can be distributed)
	PublicKeyPath string

	// PrivateKeyPath is the path to the RSA private key PEM file
	// Used for decrypting tokens (must be kept secure)
	PrivateKeyPath string

	// SymmetricKey is the AES key for symmetric encryption (A256KW)
	// Only used if Algorithm is "A256KW"
	// Length: 32 bytes (256 bits) for A256KW
	SymmetricKey []byte

	// KeyID identifies the encryption key (for key rotation)
	// Format: "gauth-{environment}-{date}" (e.g., "gauth-prod-2025-11")
	KeyID string

	// KeyRotationDays specifies how often keys should be rotated
	// Default: 365 (annual rotation)
	KeyRotationDays int
}

// DefaultJWEConfig returns the recommended JWE configuration
func DefaultJWEConfig() *JWEConfig {
	return &JWEConfig{
		Enabled:         true,
		Algorithm:       "RSA-OAEP-256",
		Encryption:      "A256GCM",
		KeyID:           fmt.Sprintf("gauth-default-%s", time.Now().Format("2006-01")),
		KeyRotationDays: 365,
	}
}

// DevelopmentJWEConfig returns a JWE config suitable for development
// Uses symmetric encryption for simplicity
func DevelopmentJWEConfig() *JWEConfig {
	// Generate a random 256-bit symmetric key
	key := make([]byte, 32)
	_, _ = rand.Read(key) // crypto/rand.Read always succeeds on supported platforms

	return &JWEConfig{
		Enabled:         true,
		Algorithm:       "A256KW",
		Encryption:      "A256GCM",
		SymmetricKey:    key,
		KeyID:           "gauth-dev-2025-11",
		KeyRotationDays: 30, // Monthly rotation for dev
	}
}

// ProductionJWEConfig returns a JWE config for production
// Requires RSA key paths to be specified
func ProductionJWEConfig(publicKeyPath, privateKeyPath, keyID string) *JWEConfig {
	return &JWEConfig{
		Enabled:         true,
		Algorithm:       algorithmRSAOAEP256,
		Encryption:      "A256GCM",
		PublicKeyPath:   publicKeyPath,
		PrivateKeyPath:  privateKeyPath,
		KeyID:           keyID,
		KeyRotationDays: 365,
	}
}

// DisabledJWEConfig returns a config with JWE encryption disabled
// Use only for internal networks or testing
func DisabledJWEConfig() *JWEConfig {
	return &JWEConfig{
		Enabled: false,
	}
}

// Validate checks the JWE configuration for correctness
func (c *JWEConfig) Validate() error {
	if !c.Enabled {
		// If disabled, no further validation needed
		return nil
	}

	// Validate algorithm
	if c.Algorithm != algorithmRSAOAEP256 && c.Algorithm != "A256KW" {
		return fmt.Errorf("unsupported JWE algorithm: %s (supported: RSA-OAEP-256, A256KW)", c.Algorithm)
	}

	// Validate encryption
	if c.Encryption != "A256GCM" && c.Encryption != "A128GCM" {
		return fmt.Errorf("unsupported JWE encryption: %s (supported: A256GCM, A128GCM)", c.Encryption)
	}

	// Validate key configuration based on algorithm
	if c.Algorithm == algorithmRSAOAEP256 {256 {
		if c.PublicKeyPath == "" {
			return errors.New("PublicKeyPath required for " + algorithmRSAOAEP256 + " algorithm")
		}
		if c.PrivateKeyPath == "" {
			return errors.New("PrivateKeyPath required for " + algorithmRSAOAEP256 + " algorithm")
		}
		// Check files exist
		if _, err := os.Stat(c.PublicKeyPath); err != nil {
			return fmt.Errorf("public key file not found: %w", err)
		}
		if _, err := os.Stat(c.PrivateKeyPath); err != nil {
			return fmt.Errorf("private key file not found: %w", err)
		}
	} else if c.Algorithm == "A256KW" {
		if len(c.SymmetricKey) != 32 {
			return fmt.Errorf("SymmetricKey must be 32 bytes for A256KW (got %d bytes)", len(c.SymmetricKey))
		}
	}

	// Validate KeyID
	if c.KeyID == "" {
		return errors.New("KeyID is required for key rotation support")
	}

	return nil
}

// LoadRSAPublicKey loads an RSA public key from a PEM file
func LoadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Try parsing as PKIX public key
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	// Validate key size (minimum 2048 bits)
	if rsaPubKey.Size()*8 < 2048 {
		return nil, fmt.Errorf("RSA key too small: %d bits (minimum 2048 bits required)", rsaPubKey.Size()*8)
	}

	return rsaPubKey, nil
}

// LoadRSAPrivateKey loads an RSA private key from a PEM file
func LoadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Try parsing as PKCS1 or PKCS8
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		keyInterface, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key (tried PKCS1 and PKCS8): %w", err)
		}
		rsaPrivKey, ok := keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		privKey = rsaPrivKey
	}

	// Validate key size (minimum 2048 bits)
	if privKey.Size()*8 < 2048 {
		return nil, fmt.Errorf("RSA key too small: %d bits (minimum 2048 bits required)", privKey.Size()*8)
	}

	return privKey, nil
}

// GenerateRSAKeyPair generates a new RSA key pair for JWE encryption
// Recommended for development/testing only. Production keys should be
// generated using secure key management systems (HSM, KMS, etc.)
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, error) {
	if bits < 2048 {
		return nil, fmt.Errorf("key size too small: %d bits (minimum 2048 bits)", bits)
	}
	if bits > 4096 {
		return nil, fmt.Errorf("key size too large: %d bits (maximum 4096 bits)", bits)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return privateKey, nil
}

// SaveRSAPrivateKey saves an RSA private key to a PEM file
func SaveRSAPrivateKey(privateKey *rsa.PrivateKey, path string) error {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer file.Close()

	if err := pem.Encode(file, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	return nil
}

// SaveRSAPublicKey saves an RSA public key to a PEM file
func SaveRSAPublicKey(publicKey *rsa.PublicKey, path string) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer file.Close()

	if err := pem.Encode(file, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to encode public key: %w", err)
	}

	return nil
}
