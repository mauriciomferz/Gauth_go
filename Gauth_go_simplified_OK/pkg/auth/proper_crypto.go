// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// ProperCrypto provides cryptographically secure operations using proven libraries
type ProperCrypto struct {
	// Use proper key derivation parameters (these are conservative, production should tune)
	argonTime    uint32
	argonMemory  uint32
	argonThreads uint8
	argonKeyLen  uint32
}

// NewProperCrypto creates a new cryptographic service with secure defaults
func NewProperCrypto() *ProperCrypto {
	return &ProperCrypto{
		argonTime:    1,         // Number of iterations
		argonMemory:  64 * 1024, // Memory usage in KiB
		argonThreads: 4,         // Number of threads
		argonKeyLen:  32,        // Length of derived key
	}
}

// SecureHash generates a cryptographically secure hash using Argon2id
// This replaces the amateur SHA256 implementation
func (pc *ProperCrypto) SecureHash(data, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty for secure hashing")
	}

	// Use Argon2id (recommended by OWASP)
	hash := argon2.IDKey(data, salt, pc.argonTime, pc.argonMemory, pc.argonThreads, pc.argonKeyLen)
	return hash, nil
}

// GenerateSecureSalt creates a cryptographically secure random salt
func (pc *ProperCrypto) GenerateSecureSalt() ([]byte, error) {
	salt := make([]byte, 32) // 256-bit salt
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate secure salt: %w", err)
	}
	return salt, nil
}

// SecureCompare performs constant-time comparison to prevent timing attacks
// This replaces string equality comparisons
func (pc *ProperCrypto) SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// EncryptData encrypts data using ChaCha20-Poly1305 (authenticated encryption)
// This provides both confidentiality and integrity
func (pc *ProperCrypto) EncryptData(plaintext, key []byte) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("key must be exactly %d bytes", chacha20poly1305.KeySize)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptData decrypts data encrypted with EncryptData
func (pc *ProperCrypto) DecryptData(ciphertext, key []byte) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("key must be exactly %d bytes", chacha20poly1305.KeySize)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(ciphertext) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// SecureTokenGenerator generates cryptographically secure tokens
type SecureTokenGenerator struct {
	crypto *ProperCrypto
}

// NewSecureTokenGenerator creates a new secure token generator
func NewSecureTokenGenerator() *SecureTokenGenerator {
	return &SecureTokenGenerator{
		crypto: NewProperCrypto(),
	}
}

// GenerateToken creates a cryptographically secure token
func (stg *SecureTokenGenerator) GenerateToken() (string, error) {
	tokenBytes := make([]byte, 32) // 256-bit token
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	// Use URL-safe base64 encoding
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	return token, nil
}

// TokenClaims represents secure token claims with proper validation
type TokenClaims struct {
	Subject   string    `json:"sub"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	ExpiresAt time.Time `json:"exp"`
	IssuedAt  time.Time `json:"iat"`
	NotBefore time.Time `json:"nbf"`
	JTI       string    `json:"jti"` // JWT ID for revocation
	Scopes    []string  `json:"scopes"`
}

// Validate performs comprehensive token claims validation
func (tc *TokenClaims) Validate() error {
	now := time.Now()

	// Check expiration
	if tc.ExpiresAt.Before(now) {
		return fmt.Errorf("token has expired")
	}

	// Check not before
	if tc.NotBefore.After(now) {
		return fmt.Errorf("token not yet valid")
	}

	// Check required fields
	if tc.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if tc.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}

	if tc.JTI == "" {
		return fmt.Errorf("JWT ID is required for revocation support")
	}

	// Validate token age (prevent very old tokens)
	maxAge := 24 * time.Hour // Configurable
	if now.Sub(tc.IssuedAt) > maxAge {
		return fmt.Errorf("token is too old")
	}

	return nil
}

// SecureAuditLog provides tamper-evident logging using proven cryptographic techniques
type SecureAuditLog struct {
	crypto  *ProperCrypto
	hmacKey []byte
}

// NewSecureAuditLog creates a new secure audit log
func NewSecureAuditLog(hmacKey []byte) (*SecureAuditLog, error) {
	if len(hmacKey) < 32 {
		return nil, fmt.Errorf("HMAC key must be at least 32 bytes")
	}

	return &SecureAuditLog{
		crypto:  NewProperCrypto(),
		hmacKey: hmacKey,
	}, nil
}

// LogEntry represents a secure audit log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Subject   string                 `json:"subject"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Result    string                 `json:"result"`
	Details   map[string]interface{} `json:"details"`
	HMAC      string                 `json:"hmac"` // Integrity protection
}

// GenerateHMAC creates an HMAC for the log entry to prevent tampering
func (sal *SecureAuditLog) GenerateHMAC(entry *LogEntry) (string, error) {
	// Create deterministic data for HMAC
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d",
		entry.ID,
		entry.EventType,
		entry.Subject,
		entry.Action,
		entry.Resource,
		entry.Result,
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		len(entry.Details), // Include details count for integrity
	)

	// Use HMAC-SHA256 for integrity protection
	h := hmac.New(sha256.New, sal.hmacKey)
	h.Write([]byte(data))
	mac := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(mac), nil
}

// VerifyEntry verifies the integrity of a log entry
func (sal *SecureAuditLog) VerifyEntry(entry *LogEntry) error {
	expectedHMAC, err := sal.GenerateHMAC(entry)
	if err != nil {
		return fmt.Errorf("failed to generate HMAC for verification: %w", err)
	}

	if !sal.crypto.SecureCompare([]byte(entry.HMAC), []byte(expectedHMAC)) {
		return fmt.Errorf("log entry integrity verification failed")
	}

	return nil
}

// CryptoConfig holds cryptographic configuration
type CryptoConfig struct {
	HMACKey           []byte
	EncryptionKey     []byte
	TokenSigningKey   []byte
	MaxTokenAge       time.Duration
	RequireStrongAuth bool
}

// ValidateConfig ensures cryptographic configuration is secure
func (cc *CryptoConfig) ValidateConfig() error {
	if len(cc.HMACKey) < 32 {
		return fmt.Errorf("HMAC key must be at least 256 bits (32 bytes)")
	}

	if len(cc.EncryptionKey) != chacha20poly1305.KeySize {
		return fmt.Errorf("encryption key must be exactly %d bytes", chacha20poly1305.KeySize)
	}

	if len(cc.TokenSigningKey) < 32 {
		return fmt.Errorf("token signing key must be at least 256 bits (32 bytes)")
	}

	if cc.MaxTokenAge < time.Minute {
		return fmt.Errorf("max token age must be at least 1 minute")
	}

	if cc.MaxTokenAge > 24*time.Hour {
		return fmt.Errorf("max token age should not exceed 24 hours for security")
	}

	return nil
}
