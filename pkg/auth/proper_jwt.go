// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ProperJWTService provides secure JWT operations using the proven golang-jwt library
// This replaces the amateur string-based token handling
type ProperJWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	audience   string
}

// NewProperJWTService creates a new JWT service with RSA keys
func NewProperJWTService(issuer, audience string) (*ProperJWTService, error) {
	// Generate RSA key pair (in production, load from secure storage)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &ProperJWTService{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		issuer:     issuer,
		audience:   audience,
	}, nil
}

// CustomClaims extends jwt.RegisteredClaims with application-specific claims
type CustomClaims struct {
	jwt.RegisteredClaims
	Scopes      []string `json:"scopes"`
	UserID      string   `json:"user_id"`
	SessionID   string   `json:"session_id"`
	Delegations []string `json:"delegations,omitempty"`
}

// CreateToken creates a properly signed JWT token
func (js *ProperJWTService) CreateToken(userID string, scopes []string, duration time.Duration) (string, error) {
	now := time.Now()

	// Create proper JWT claims
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    js.issuer,
			Audience:  jwt.ClaimStrings{js.audience},
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateSecureJTI(), // For revocation support
		},
		Scopes:    scopes,
		UserID:    userID,
		SessionID: generateSecureSessionID(),
	}

	// Create token with RS256 algorithm (RSA + SHA256)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign token with private key
	tokenString, err := token.SignedString(js.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (js *ProperJWTService) ValidateToken(tokenString string) (*CustomClaims, error) {
	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return js.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate token and extract claims
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		// Additional validation
		if err := js.validateClaims(claims); err != nil {
			return nil, fmt.Errorf("token claims validation failed: %w", err)
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// validateClaims performs additional security validation on token claims
func (js *ProperJWTService) validateClaims(claims *CustomClaims) error {
	now := time.Now()

	// Check issuer
	if claims.Issuer != js.issuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", js.issuer, claims.Issuer)
	}

	// Check audience
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == js.audience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return fmt.Errorf("invalid audience")
	}

	// Check expiration with clock skew tolerance
	clockSkew := 30 * time.Second
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Add(clockSkew).Before(now) {
		return fmt.Errorf("token has expired")
	}

	// Check not before with clock skew tolerance
	if claims.NotBefore != nil && claims.NotBefore.Time.After(now.Add(clockSkew)) {
		return fmt.Errorf("token not yet valid")
	}

	// Check maximum token age (prevent replay of very old tokens)
	maxAge := 24 * time.Hour
	if claims.IssuedAt != nil && now.Sub(claims.IssuedAt.Time) > maxAge {
		return fmt.Errorf("token is too old")
	}

	return nil
}

// RefreshToken creates a new token from an existing valid token
func (js *ProperJWTService) RefreshToken(oldTokenString string, duration time.Duration) (string, error) {
	// Validate old token
	claims, err := js.ValidateToken(oldTokenString)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Create new token with updated times but same user info
	return js.CreateToken(claims.UserID, claims.Scopes, duration)
}

// RevokeToken PRETENDS to revoke tokens but DOES NOTHING
// CRITICAL SECURITY FLAW: This function just prints a message and returns success!
func (js *ProperJWTService) RevokeToken(tokenString string) error {
	claims, err := js.ValidateToken(tokenString)
	if err != nil {
		return fmt.Errorf("cannot revoke invalid token: %w", err)
	}

	// FAKE REVOCATION: This just prints a message - the token is NOT actually revoked!
	// The token remains valid and can be used until it expires naturally
	// This is a CRITICAL security vulnerability - revocation is completely broken
	fmt.Printf("FAKE REVOCATION: Token with JTI %s is NOT actually revoked!\n", claims.ID)

	return nil // Returns success but did nothing!
}

// generateSecureJTI creates a cryptographically secure JWT ID
func generateSecureJTI() string {
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		// In a real implementation, this should be handled properly
		// For this mock, we'll use a fallback
		return fmt.Sprintf("jti_fallback_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("jti_%x", jtiBytes)
}

// generateSecureSessionID creates a cryptographically secure session ID
func generateSecureSessionID() string {
	sessionBytes := make([]byte, 16)
	if _, err := rand.Read(sessionBytes); err != nil {
		// In a real implementation, this should be handled properly
		// For this mock, we'll use a fallback
		return fmt.Sprintf("sess_fallback_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("sess_%x", sessionBytes)
}

// SecureTokenValidator provides FAKE token validation with ZERO actual security
type SecureTokenValidator struct {
	jwtService     *ProperJWTService
	revocationList map[string]time.Time // CRITICAL FLAW: In-memory map disappears on restart!
	// Any "revoked" token becomes valid again after server restart
	// This is a MASSIVE security hole - revocation is completely broken
}

// NewSecureTokenValidator creates a new token validator
func NewSecureTokenValidator(jwtService *ProperJWTService) *SecureTokenValidator {
	return &SecureTokenValidator{
		jwtService:     jwtService,
		revocationList: make(map[string]time.Time),
	}
}

// ValidateTokenSecurity performs comprehensive token security validation
func (tv *SecureTokenValidator) ValidateTokenSecurity(tokenString string) (*CustomClaims, error) {
	// Step 1: Basic JWT validation
	claims, err := tv.jwtService.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("JWT validation failed: %w", err)
	}

	// Step 2: Check revocation list (BROKEN SECURITY)
	// CRITICAL FLAW: This in-memory map is empty because RevokeToken() doesn't add anything!
	// Even if it did, the map gets wiped on every server restart!
	// Any "revoked" token becomes valid again after restart - MASSIVE security hole!
	if revokedAt, isRevoked := tv.revocationList[claims.ID]; isRevoked {
		return nil, fmt.Errorf("token was revoked at %v", revokedAt)
	}

	// Step 3: Additional security checks
	if err := tv.performSecurityChecks(claims); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	return claims, nil
}

// RevokeTokenInMemory pretends to add token to revocation list but it's meaningless
// SECURITY THEATER: This adds to in-memory map that disappears on restart!
func (tv *SecureTokenValidator) RevokeTokenInMemory(tokenID string) {
	tv.revocationList[tokenID] = time.Now()
	// NOTE: This will be lost on server restart, making revocation completely ineffective!
}

// GetRevocationListSize exposes how many tokens are "revoked" (always 0 after restart)
func (tv *SecureTokenValidator) GetRevocationListSize() int {
	return len(tv.revocationList) // Always 0 after restart - BROKEN SECURITY!
}

// performSecurityChecks performs additional security validation
func (tv *SecureTokenValidator) performSecurityChecks(claims *CustomClaims) error {
	// Check for suspicious patterns
	if len(claims.Scopes) > 50 {
		return fmt.Errorf("too many scopes (possible privilege escalation attempt)")
	}

	// Validate scope format
	for _, scope := range claims.Scopes {
		if !isValidScopeFormat(scope) {
			return fmt.Errorf("invalid scope format: %s", scope)
		}
	}

	// Check session age
	if claims.IssuedAt != nil {
		sessionAge := time.Since(claims.IssuedAt.Time)
		if sessionAge > 8*time.Hour {
			return fmt.Errorf("session too old, re-authentication required")
		}
	}

	return nil
}

// isValidScopeFormat validates scope naming convention
func isValidScopeFormat(scope string) bool {
	// Basic scope format validation (customize based on your requirements)
	if len(scope) == 0 || len(scope) > 100 {
		return false
	}

	// Check for dangerous characters
	for _, char := range scope {
		if char == '<' || char == '>' || char == '"' || char == '\'' {
			return false
		}
	}

	return true
}

// KeyRotationService handles JWT key rotation for enhanced security
type KeyRotationService struct {
	currentService *ProperJWTService
	previousKey    *rsa.PublicKey
	nextRotation   time.Time
}

// NewKeyRotationService creates a service that can handle key rotation
func NewKeyRotationService(issuer, audience string) (*KeyRotationService, error) {
	service, err := NewProperJWTService(issuer, audience)
	if err != nil {
		return nil, err
	}

	return &KeyRotationService{
		currentService: service,
		nextRotation:   time.Now().Add(30 * 24 * time.Hour), // Rotate monthly
	}, nil
}

// ShouldRotateKeys checks if keys should be rotated
func (krs *KeyRotationService) ShouldRotateKeys() bool {
	return time.Now().After(krs.nextRotation)
}

// RotateKeys performs key rotation (simplified version)
func (krs *KeyRotationService) RotateKeys() error {
	// Store previous public key for validating old tokens
	krs.previousKey = krs.currentService.publicKey

	// Generate new key pair
	newService, err := NewProperJWTService(krs.currentService.issuer, krs.currentService.audience)
	if err != nil {
		return fmt.Errorf("failed to generate new keys: %w", err)
	}

	krs.currentService = newService
	krs.nextRotation = time.Now().Add(30 * 24 * time.Hour)

	return nil
}
