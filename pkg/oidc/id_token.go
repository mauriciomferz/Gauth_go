// Package oidc - ID Token Service
// Implements OIDC ID Token issuance and validation
package oidc

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IDTokenService handles OIDC ID Token issuance and validation
// Spec: OpenID Connect Core 1.0 Section 2 (ID Token), Section 3.1.3.3 (Validation)
type IDTokenService struct {
	issuerURL     string
	signingKey    *rsa.PrivateKey
	signingKeyID  string
	signingMethod jwt.SigningMethod
	defaultExpiry time.Duration
}

// IDTokenServiceConfig configures ID Token Service
type IDTokenServiceConfig struct {
	IssuerURL     string
	SigningKey    *rsa.PrivateKey
	SigningKeyID  string
	SigningMethod string        // "RS256", "RS384", "RS512"
	TokenExpiry   time.Duration // Default: 1 hour
}

// NewIDTokenService creates a new ID Token Service
func NewIDTokenService(config *IDTokenServiceConfig) (*IDTokenService, error) {
	if config.SigningKey == nil {
		return nil, fmt.Errorf("signing key is required")
	}

	// Default signing method: RS256 (REQUIRED by OIDC spec)
	signingMethod := jwt.SigningMethodRS256
	if config.SigningMethod != "" {
		switch config.SigningMethod {
		case "RS256":
			signingMethod = jwt.SigningMethodRS256
		case "RS384":
			signingMethod = jwt.SigningMethodRS384
		case "RS512":
			signingMethod = jwt.SigningMethodRS512
		default:
			return nil, fmt.Errorf("unsupported signing method: %s", config.SigningMethod)
		}
	}

	// Default expiry: 1 hour (3600 seconds)
	expiry := time.Hour
	if config.TokenExpiry > 0 {
		expiry = config.TokenExpiry
	}

	return &IDTokenService{
		issuerURL:     config.IssuerURL,
		signingKey:    config.SigningKey,
		signingKeyID:  config.SigningKeyID,
		signingMethod: signingMethod,
		defaultExpiry: expiry,
	}, nil
}

// IssueIDToken creates and signs a new OIDC ID Token
// Spec: OpenID Connect Core 1.0 Section 2
func (s *IDTokenService) IssueIDToken(ctx context.Context, claims *IDTokenClaims) (string, error) {
	// Validate required claims
	if err := s.validateClaims(claims); err != nil {
		return "", fmt.Errorf("invalid claims: %w", err)
	}

	// Set issuer
	claims.Issuer = s.issuerURL

	// Set issued at time
	now := time.Now()
	claims.IssuedAt = jwt.NewNumericDate(now)

	// Set expiration (default 1 hour)
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(s.defaultExpiry))
	}

	// Set Not Before (optional, but recommended)
	if claims.NotBefore == nil {
		claims.NotBefore = jwt.NewNumericDate(now)
	}

	// Create JWT token
	token := jwt.NewWithClaims(s.signingMethod, claims)

	// Set Key ID header (kid)
	if s.signingKeyID != "" {
		token.Header["kid"] = s.signingKeyID
	}

	// Sign token
	signedToken, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign ID token: %w", err)
	}

	return signedToken, nil
}

// ValidateIDToken validates an OIDC ID Token
// Spec: OpenID Connect Core 1.0 Section 3.1.3.7 (ID Token Validation)
func (s *IDTokenService) ValidateIDToken(
	ctx context.Context,
	idToken string,
	expectedAudience string,
) (*IDTokenClaims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(idToken, &IDTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if token.Method.Alg() != s.signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}

		// Return public key for verification
		return &s.signingKey.PublicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*IDTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid ID token")
	}

	// Validate claims according to OIDC spec Section 3.1.3.7
	if err := s.validateIDTokenClaims(claims, expectedAudience); err != nil {
		return nil, fmt.Errorf("ID token validation failed: %w", err)
	}

	return claims, nil
}

// validateClaims validates required claims before issuing ID token
func (s *IDTokenService) validateClaims(claims *IDTokenClaims) error {
	// Subject (sub) is REQUIRED
	if claims.Subject == "" {
		return fmt.Errorf("sub claim is required")
	}

	// Audience (aud) is REQUIRED
	if len(claims.Audience) == 0 {
		return fmt.Errorf("aud claim is required")
	}

	return nil
}

// validateIDTokenClaims validates ID token claims per OIDC spec
// Spec: OpenID Connect Core 1.0 Section 3.1.3.7
func (s *IDTokenService) validateIDTokenClaims(claims *IDTokenClaims, expectedAudience string) error {
	// 1. Validate issuer (iss)
	if claims.Issuer != s.issuerURL {
		return fmt.Errorf("invalid issuer: expected %s, got %s", s.issuerURL, claims.Issuer)
	}

	// 2. Validate audience (aud)
	// Must contain client_id that this ID token is intended for
	if expectedAudience != "" {
		audienceValid := false
		for _, aud := range claims.Audience {
			if aud == expectedAudience {
				audienceValid = true
				break
			}
		}
		if !audienceValid {
			return fmt.Errorf("invalid audience: expected %s", expectedAudience)
		}
	}

	// 3. Validate authorized party (azp) if multiple audiences
	if len(claims.Audience) > 1 && claims.AuthorizedParty == "" {
		return fmt.Errorf("azp claim required when multiple audiences present")
	}
	if claims.AuthorizedParty != "" && expectedAudience != "" {
		if claims.AuthorizedParty != expectedAudience {
			return fmt.Errorf("invalid authorized party: expected %s, got %s",
				expectedAudience, claims.AuthorizedParty)
		}
	}

	// 4. Validate expiration (exp)
	now := time.Now()
	if claims.ExpiresAt != nil {
		if claims.ExpiresAt.Time.Before(now) {
			return fmt.Errorf("ID token expired at %v", claims.ExpiresAt.Time)
		}
	}

	// 5. Validate issued at (iat)
	if claims.IssuedAt != nil {
		// Token must not be issued in the future
		// Allow 5 minutes clock skew
		if claims.IssuedAt.Time.After(now.Add(5 * time.Minute)) {
			return fmt.Errorf("ID token issued in the future: %v", claims.IssuedAt.Time)
		}
	}

	// 6. Validate not before (nbf) if present
	if claims.NotBefore != nil {
		// Allow 5 minutes clock skew
		if claims.NotBefore.Time.After(now.Add(5 * time.Minute)) {
			return fmt.Errorf("ID token not yet valid: not before %v", claims.NotBefore.Time)
		}
	}

	// 7. Validate nonce if present (for replay attack prevention)
	// Note: Nonce validation requires storing expected nonce in session
	// This is typically handled by the relying party (client application)

	return nil
}

// CreateIDTokenFromIdentity creates ID token from GAuth identity proof
// This bridges GAuth identity structures to OIDC ID tokens
func (s *IDTokenService) CreateIDTokenFromIdentity(
	ctx context.Context,
	subjectID string,
	audience []string,
	identityType string,
	trustLevel string,
	additionalClaims map[string]interface{},
) (string, error) {
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  subjectID,
			Audience: audience,
		},
	}

	// Map GAuth trust level to OIDC ACR
	claims.ACR = s.mapTrustLevelToACR(trustLevel)

	// Set GAuth entity type
	claims.EntityType = identityType

	// Add additional claims from proof data
	if name, ok := additionalClaims["name"].(string); ok {
		claims.Name = name
	}
	if email, ok := additionalClaims["email"].(string); ok {
		claims.Email = email
	}
	if entityID, ok := additionalClaims["entity_id"].(string); ok {
		claims.EntityID = entityID
	}
	if legalEntityName, ok := additionalClaims["legal_entity_name"].(string); ok {
		claims.LegalEntityName = legalEntityName
	}
	if jurisdiction, ok := additionalClaims["jurisdiction"].(string); ok {
		claims.Jurisdiction = jurisdiction
	}
	if tspName, ok := additionalClaims["tsp_name"].(string); ok {
		claims.TSPName = tspName
	}
	if tspID, ok := additionalClaims["tsp_id"].(string); ok {
		claims.TSPID = tspID
	}

	return s.IssueIDToken(ctx, claims)
}

// mapTrustLevelToACR maps GAuth trust level to OIDC ACR value
func (s *IDTokenService) mapTrustLevelToACR(trustLevel string) string {
	switch trustLevel {
	case "low":
		return "1"
	case "substantial":
		return "substantial"
	case "high":
		return "high"
	default:
		return "0"
	}
}

// GetSigningKeyID returns the current signing key ID
func (s *IDTokenService) GetSigningKeyID() string {
	return s.signingKeyID
}

// GetSigningAlgorithm returns the current signing algorithm
func (s *IDTokenService) GetSigningAlgorithm() string {
	return s.signingMethod.Alg()
}

// GetDefaultExpiry returns the default token expiry duration
func (s *IDTokenService) GetDefaultExpiry() time.Duration {
	return s.defaultExpiry
}
