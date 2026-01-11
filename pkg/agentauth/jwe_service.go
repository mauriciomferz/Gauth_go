// Package agentauth - JWE (JSON Web Encryption) Service
// Implements token encryption/decryption for Extended Tokens
// Addresses Security Hardening gap identified in audit
package agentauth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-jose/go-jose/v3"
)

// JWEService provides JWE encryption and decryption operations
type JWEService interface {
	// EncryptToken encrypts a JWT string into a JWE token
	EncryptToken(ctx context.Context, jwtString string) (string, error)

	// DecryptToken decrypts a JWE token back to a JWT string
	DecryptToken(ctx context.Context, jweString string) (string, error)

	// RotateKeys initiates key rotation (loads new keys)
	RotateKeys(ctx context.Context) error

	// GetPublicKey retrieves a public key by key ID
	GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error)

	// IsEnabled returns whether JWE encryption is enabled
	IsEnabled() bool
}

// DefaultJWEService is the default implementation of JWEService
type DefaultJWEService struct {
	config      *JWEConfig
	publicKeys  map[string]*rsa.PublicKey  // kid -> public key
	privateKeys map[string]*rsa.PrivateKey // kid -> private key
	mu          sync.RWMutex
}

// NewJWEService creates a new JWE service
func NewJWEService(config *JWEConfig) (*DefaultJWEService, error) {
	if config == nil {
		return nil, errors.New("JWE config cannot be nil")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid JWE config: %w", err)
	}

	service := &DefaultJWEService{
		config:      config,
		publicKeys:  make(map[string]*rsa.PublicKey),
		privateKeys: make(map[string]*rsa.PrivateKey),
	}

	// Load keys if JWE is enabled
	if config.Enabled {
		if err := service.loadKeys(); err != nil {
			return nil, fmt.Errorf("failed to load JWE keys: %w", err)
		}
	}

	return service, nil
}

// loadKeys loads public and private keys based on configuration
func (s *DefaultJWEService) loadKeys() error {
	if s.config.Algorithm == "RSA-OAEP-256" {
		// Load RSA public key
		pubKey, err := LoadRSAPublicKey(s.config.PublicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load public key: %w", err)
		}
		s.publicKeys[s.config.KeyID] = pubKey

		// Load RSA private key
		privKey, err := LoadRSAPrivateKey(s.config.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}
		s.privateKeys[s.config.KeyID] = privKey
	}
	// For A256KW (symmetric), key is already in config

	return nil
}

// IsEnabled returns whether JWE encryption is enabled
func (s *DefaultJWEService) IsEnabled() bool {
	return s.config.Enabled
}

// EncryptToken encrypts a JWT string into a JWE token
// Implements nested JWT pattern: JWE(JWT(claims))
func (s *DefaultJWEService) EncryptToken(ctx context.Context, jwtString string) (string, error) {
	if !s.config.Enabled {
		return "", errors.New("JWE encryption is disabled")
	}

	if jwtString == "" {
		return "", errors.New("JWT string cannot be empty")
	}

	// Select encryption algorithm
	encAlg := jose.A256GCM
	if s.config.Encryption == "A128GCM" {
		encAlg = jose.A128GCM
	}

	var encrypter jose.Encrypter
	var err error

	if s.config.Algorithm == "RSA-OAEP-256" {
		// Get public key for encryption
		s.mu.RLock()
		pubKey, exists := s.publicKeys[s.config.KeyID]
		s.mu.RUnlock()
		if !exists {
			return "", fmt.Errorf("public key not found for kid: %s", s.config.KeyID)
		}

		// Create RSA-OAEP encrypter
		recipient := jose.Recipient{
			Algorithm: jose.RSA_OAEP_256,
			Key:       pubKey,
			KeyID:     s.config.KeyID,
		}

		opts := &jose.EncrypterOptions{
			Compression: jose.DEFLATE, // Compress JWT before encryption
		}
		opts.WithType("JWT") // Indicate nested JWT
		opts.WithContentType("JWT")

		encrypter, err = jose.NewEncrypter(encAlg, recipient, opts)
		if err != nil {
			return "", fmt.Errorf("failed to create RSA encrypter: %w", err)
		}
	} else if s.config.Algorithm == "A256KW" {
		// Create symmetric key encrypter
		recipient := jose.Recipient{
			Algorithm: jose.A256KW,
			Key:       s.config.SymmetricKey,
			KeyID:     s.config.KeyID,
		}

		opts := &jose.EncrypterOptions{
			Compression: jose.DEFLATE,
		}
		opts.WithType("JWT")
		opts.WithContentType("JWT")

		encrypter, err = jose.NewEncrypter(encAlg, recipient, opts)
		if err != nil {
			return "", fmt.Errorf("failed to create symmetric encrypter: %w", err)
		}
	} else {
		return "", fmt.Errorf("unsupported JWE algorithm: %s", s.config.Algorithm)
	}

	// Encrypt the JWT
	jwe, err := encrypter.Encrypt([]byte(jwtString))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Serialize to compact format
	jweString, err := jwe.CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("failed to serialize JWE: %w", err)
	}

	return jweString, nil
}

// DecryptToken decrypts a JWE token back to a JWT string
func (s *DefaultJWEService) DecryptToken(ctx context.Context, jweString string) (string, error) {
	if !s.config.Enabled {
		return "", errors.New("JWE encryption is disabled")
	}

	if jweString == "" {
		return "", errors.New("JWE string cannot be empty")
	}

	// Parse JWE compact serialization
	jwe, err := jose.ParseEncrypted(jweString)
	if err != nil {
		return "", fmt.Errorf("failed to parse JWE: %w", err)
	}

	// Extract key ID from JWE header
	kid := jwe.Header.KeyID
	if kid == "" {
		return "", errors.New("JWE missing key ID (kid)")
	}

	var decrypted []byte

	if s.config.Algorithm == "RSA-OAEP-256" {
		// Get private key for decryption
		s.mu.RLock()
		privKey, exists := s.privateKeys[kid]
		s.mu.RUnlock()
		if !exists {
			return "", fmt.Errorf("private key not found for kid: %s", kid)
		}

		// Decrypt JWE
		decrypted, err = jwe.Decrypt(privKey)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt token with RSA: %w", err)
		}
	} else if s.config.Algorithm == "A256KW" {
		// Decrypt with symmetric key
		decrypted, err = jwe.Decrypt(s.config.SymmetricKey)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt token with symmetric key: %w", err)
		}
	} else {
		return "", fmt.Errorf("unsupported JWE algorithm: %s", s.config.Algorithm)
	}

	return string(decrypted), nil
}

// RotateKeys loads new keys for key rotation
func (s *DefaultJWEService) RotateKeys(ctx context.Context) error {
	if !s.config.Enabled {
		return errors.New("JWE encryption is disabled")
	}

	// Reload keys (in production, this would load new keys while keeping old ones)
	if err := s.loadKeys(); err != nil {
		return fmt.Errorf("failed to rotate keys: %w", err)
	}

	return nil
}

// GetPublicKey retrieves a public key by key ID
func (s *DefaultJWEService) GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pubKey, exists := s.publicKeys[kid]
	if !exists {
		return nil, fmt.Errorf("public key not found for kid: %s", kid)
	}

	return pubKey, nil
}

// IsJWE checks if a token string is JWE-encrypted (vs plain JWT)
// JWE has 5 parts: header.encrypted_key.iv.ciphertext.tag
// JWT has 3 parts: header.payload.signature
func IsJWE(tokenString string) bool {
	parts := strings.Split(tokenString, ".")
	return len(parts) == 5
}
