package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RFC 9068: JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens
//
// This implementation provides JWT-formatted access tokens instead of opaque tokens.
// JWT access tokens are self-contained and can be validated by resource servers without
// introspection, improving performance and scalability.
//
// Key Features:
// - JWT format with "at+jwt" type
// - Required claims: iss, exp, aud, sub, client_id, iat, jti
// - Optional claims: scope, auth_time, acr, amr
// - RS256 or ES256 signatures
// - Resource server validation without AS communication

// JWTAccessTokenClaims represents the claims in a JWT access token per RFC 9068.
type JWTAccessTokenClaims struct {
	// Standard JWT claims
	Issuer    string   `json:"iss"`           // Authorization server identifier
	Subject   string   `json:"sub"`           // Subject (typically user ID)
	Audience  []string `json:"aud"`           // Intended audience (resource servers)
	ExpiresAt int64    `json:"exp"`           // Expiration time (Unix timestamp)
	IssuedAt  int64    `json:"iat"`           // Issued at time (Unix timestamp)
	JWTID     string   `json:"jti"`           // Unique token identifier
	NotBefore int64    `json:"nbf,omitempty"` // Not before time (optional)

	// OAuth 2.0 specific claims (RFC 9068 Section 2.2)
	ClientID string `json:"client_id"`       // Client identifier
	Scope    string `json:"scope,omitempty"` // Space-separated scope values

	// Authentication context
	AuthTime int64    `json:"auth_time,omitempty"` // Authentication timestamp
	ACR      string   `json:"acr,omitempty"`       // Authentication Context Class Reference
	AMR      []string `json:"amr,omitempty"`       // Authentication Methods References

	// Additional claims
	Username     string                 `json:"username,omitempty"` // Username (informational)
	Groups       []string               `json:"groups,omitempty"`   // User groups (optional)
	CustomClaims map[string]interface{} `json:"-"`                  // Extension claims
}

// JWTAccessTokenConfig configures JWT access token generation.
type JWTAccessTokenConfig struct {
	// Issuer is the authorization server identifier
	Issuer string

	// SigningKey is the private key for signing tokens (RSA or ECDSA)
	SigningKey interface{}

	// SigningMethod is the JWT signing algorithm (RS256, ES256)
	SigningMethod jwt.SigningMethod

	// DefaultAudience is the default audience if not specified
	DefaultAudience []string

	// TokenLifetime is the default token lifetime
	TokenLifetime time.Duration

	// IncludeGroups indicates whether to include user groups in token
	IncludeGroups bool

	// IncludeUsername indicates whether to include username in token
	IncludeUsername bool
}

// DefaultJWTAccessTokenConfig returns default JWT access token configuration.
func DefaultJWTAccessTokenConfig(issuer string, signingKey *rsa.PrivateKey) *JWTAccessTokenConfig {
	return &JWTAccessTokenConfig{
		Issuer:          issuer,
		SigningKey:      signingKey,
		SigningMethod:   jwt.SigningMethodRS256,
		DefaultAudience: []string{issuer},
		TokenLifetime:   3600 * time.Second, // 1 hour
		IncludeGroups:   false,
		IncludeUsername: true,
	}
}

// JWTAccessTokenService manages JWT access token operations.
type JWTAccessTokenService struct {
	config *JWTAccessTokenConfig
}

// NewJWTAccessTokenService creates a new JWT access token service.
func NewJWTAccessTokenService(config *JWTAccessTokenConfig) (*JWTAccessTokenService, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Issuer == "" {
		return nil, fmt.Errorf("issuer cannot be empty")
	}

	if config.SigningKey == nil {
		return nil, fmt.Errorf("signing key cannot be nil")
	}

	if config.SigningMethod == nil {
		return nil, fmt.Errorf("signing method cannot be nil")
	}

	return &JWTAccessTokenService{
		config: config,
	}, nil
}

// GenerateAccessToken generates a new JWT access token.
//
// Parameters:
//   - ctx: Context for the operation
//   - subject: Subject identifier (typically user ID)
//   - clientID: Client identifier
//   - scopes: Space-separated scope values
//   - audience: Intended audience (resource servers)
//   - authTime: Authentication timestamp (optional, 0 for none)
//   - acr: Authentication Context Class Reference (optional)
//   - amr: Authentication Methods References (optional)
//
// Returns:
//   - string: JWT access token
//   - error: Generation error
func (s *JWTAccessTokenService) GenerateAccessToken(
	ctx context.Context,
	subject string,
	clientID string,
	scopes string,
	audience []string,
	authTime int64,
	acr string,
	amr []string,
) (string, error) {
	if subject == "" {
		return "", fmt.Errorf("subject cannot be empty")
	}

	if clientID == "" {
		return "", fmt.Errorf("client ID cannot be empty")
	}

	// Use default audience if not specified
	if len(audience) == 0 {
		audience = s.config.DefaultAudience
	}

	now := time.Now()
	exp := now.Add(s.config.TokenLifetime)

	// Generate unique token ID
	jti, err := s.generateJTI()
	if err != nil {
		return "", fmt.Errorf("failed to generate JTI: %w", err)
	}

	// Build claims
	claims := JWTAccessTokenClaims{
		Issuer:    s.config.Issuer,
		Subject:   subject,
		Audience:  audience,
		ExpiresAt: exp.Unix(),
		IssuedAt:  now.Unix(),
		JWTID:     jti,
		ClientID:  clientID,
		Scope:     scopes,
	}

	if authTime > 0 {
		claims.AuthTime = authTime
	}

	if acr != "" {
		claims.ACR = acr
	}

	if len(amr) > 0 {
		claims.AMR = amr
	}

	// Create JWT token with "at+jwt" type
	token := jwt.NewWithClaims(s.config.SigningMethod, s.claimsToMapClaims(claims))
	token.Header["typ"] = "at+jwt" // RFC 9068 Section 2.1

	// Sign token
	tokenString, err := token.SignedString(s.config.SigningKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken validates a JWT access token.
//
// Parameters:
//   - ctx: Context for the operation
//   - tokenString: JWT access token to validate
//   - expectedAudience: Expected audience (resource server identifier)
//
// Returns:
//   - *JWTAccessTokenClaims: Validated claims
//   - error: Validation error
func (s *JWTAccessTokenService) ValidateAccessToken(
	ctx context.Context,
	tokenString string,
	expectedAudience string,
) (*JWTAccessTokenClaims, error) {
	if tokenString == "" {
		return nil, &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "token is empty",
		}
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// Validate signing method
		if t.Method.Alg() != s.config.SigningMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
		}

		// Validate typ header
		if typ, ok := t.Header["typ"].(string); !ok || typ != "at+jwt" {
			return nil, fmt.Errorf("invalid 'typ' header, must be 'at+jwt'")
		}

		// Return public key for verification
		return s.getPublicKey(), nil
	})

	if err != nil {
		return nil, &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: fmt.Sprintf("token validation failed: %v", err),
		}
	}

	if !token.Valid {
		return nil, &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "token is invalid",
		}
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "invalid claims format",
		}
	}

	claims := s.mapClaimsToClaims(mapClaims)

	// Validate required claims
	if err := s.validateClaims(claims, expectedAudience); err != nil {
		return nil, err
	}

	return claims, nil
}

// validateClaims validates JWT access token claims per RFC 9068.
func (s *JWTAccessTokenService) validateClaims(claims *JWTAccessTokenClaims, expectedAudience string) error {
	// Validate issuer
	if claims.Issuer != s.config.Issuer {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: fmt.Sprintf("invalid issuer: expected %s, got %s", s.config.Issuer, claims.Issuer),
		}
	}

	// Validate audience
	if expectedAudience != "" {
		found := false
		for _, aud := range claims.Audience {
			if aud == expectedAudience {
				found = true
				break
			}
		}
		if !found {
			return &OIDCError{
				ErrorCode:        ErrorInvalidToken,
				ErrorDescription: fmt.Sprintf("audience mismatch: expected %s", expectedAudience),
			}
		}
	}

	// Validate expiration
	now := time.Now().Unix()
	if claims.ExpiresAt <= now {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "token has expired",
		}
	}

	// Validate not before (if present)
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "token not yet valid",
		}
	}

	// Validate required claims
	if claims.Subject == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "missing 'sub' claim",
		}
	}

	if claims.ClientID == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "missing 'client_id' claim",
		}
	}

	if claims.JWTID == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "missing 'jti' claim",
		}
	}

	return nil
}

// IntrospectToken provides introspection-compatible response for JWT token.
//
// This allows JWT tokens to be used with introspection endpoints while
// maintaining backward compatibility.
func (s *JWTAccessTokenService) IntrospectToken(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	claims, err := s.ValidateAccessToken(ctx, tokenString, "")
	if err != nil {
		// Return inactive response on error
		return map[string]interface{}{
			"active": false,
		}, nil
	}

	// Build introspection response
	response := map[string]interface{}{
		"active":    true,
		"sub":       claims.Subject,
		"client_id": claims.ClientID,
		"exp":       claims.ExpiresAt,
		"iat":       claims.IssuedAt,
		"iss":       claims.Issuer,
		"jti":       claims.JWTID,
	}

	if claims.Scope != "" {
		response["scope"] = claims.Scope
	}

	if len(claims.Audience) > 0 {
		response["aud"] = claims.Audience
	}

	if claims.AuthTime > 0 {
		response["auth_time"] = claims.AuthTime
	}

	if claims.ACR != "" {
		response["acr"] = claims.ACR
	}

	if len(claims.AMR) > 0 {
		response["amr"] = claims.AMR
	}

	if claims.Username != "" {
		response["username"] = claims.Username
	}

	return response, nil
}

// claimsToMapClaims converts JWTAccessTokenClaims to jwt.MapClaims.
func (s *JWTAccessTokenService) claimsToMapClaims(claims JWTAccessTokenClaims) jwt.MapClaims {
	mc := jwt.MapClaims{
		"iss":       claims.Issuer,
		"sub":       claims.Subject,
		"aud":       claims.Audience,
		"exp":       claims.ExpiresAt,
		"iat":       claims.IssuedAt,
		"jti":       claims.JWTID,
		"client_id": claims.ClientID,
	}

	if claims.NotBefore > 0 {
		mc["nbf"] = claims.NotBefore
	}

	if claims.Scope != "" {
		mc["scope"] = claims.Scope
	}

	if claims.AuthTime > 0 {
		mc["auth_time"] = claims.AuthTime
	}

	if claims.ACR != "" {
		mc["acr"] = claims.ACR
	}

	if len(claims.AMR) > 0 {
		mc["amr"] = claims.AMR
	}

	if claims.Username != "" {
		mc["username"] = claims.Username
	}

	if len(claims.Groups) > 0 {
		mc["groups"] = claims.Groups
	}

	// Add custom claims
	for k, v := range claims.CustomClaims {
		mc[k] = v
	}

	return mc
}

// mapClaimsToClaims converts jwt.MapClaims to JWTAccessTokenClaims.
func (s *JWTAccessTokenService) mapClaimsToClaims(mc jwt.MapClaims) *JWTAccessTokenClaims {
	claims := &JWTAccessTokenClaims{}

	if iss, ok := mc["iss"].(string); ok {
		claims.Issuer = iss
	}

	if sub, ok := mc["sub"].(string); ok {
		claims.Subject = sub
	}

	// Handle audience (can be string or []string)
	if aud, ok := mc["aud"].([]interface{}); ok {
		claims.Audience = make([]string, len(aud))
		for i, a := range aud {
			if audStr, ok := a.(string); ok {
				claims.Audience[i] = audStr
			}
		}
	} else if aud, ok := mc["aud"].(string); ok {
		claims.Audience = []string{aud}
	}

	if exp, ok := mc["exp"].(float64); ok {
		claims.ExpiresAt = int64(exp)
	}

	if iat, ok := mc["iat"].(float64); ok {
		claims.IssuedAt = int64(iat)
	}

	if nbf, ok := mc["nbf"].(float64); ok {
		claims.NotBefore = int64(nbf)
	}

	if jti, ok := mc["jti"].(string); ok {
		claims.JWTID = jti
	}

	if clientID, ok := mc["client_id"].(string); ok {
		claims.ClientID = clientID
	}

	if scope, ok := mc["scope"].(string); ok {
		claims.Scope = scope
	}

	if authTime, ok := mc["auth_time"].(float64); ok {
		claims.AuthTime = int64(authTime)
	}

	if acr, ok := mc["acr"].(string); ok {
		claims.ACR = acr
	}

	if amr, ok := mc["amr"].([]interface{}); ok {
		claims.AMR = make([]string, len(amr))
		for i, a := range amr {
			if amrStr, ok := a.(string); ok {
				claims.AMR[i] = amrStr
			}
		}
	}

	if username, ok := mc["username"].(string); ok {
		claims.Username = username
	}

	if groups, ok := mc["groups"].([]interface{}); ok {
		claims.Groups = make([]string, len(groups))
		for i, g := range groups {
			if groupStr, ok := g.(string); ok {
				claims.Groups[i] = groupStr
			}
		}
	}

	return claims
}

// generateJTI generates a unique JWT ID.
func (s *JWTAccessTokenService) generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// getPublicKey extracts the public key from the signing key.
func (s *JWTAccessTokenService) getPublicKey() interface{} {
	switch key := s.config.SigningKey.(type) {
	case *rsa.PrivateKey:
		return &key.PublicKey
	default:
		// For other key types (ECDSA, etc.)
		return s.config.SigningKey
	}
}

// ParseScopes parses a space-separated scope string into a slice.
func ParseScopes(scopes string) []string {
	if scopes == "" {
		return []string{}
	}
	return strings.Fields(scopes)
}

// HasScope checks if a scope string contains a specific scope.
func HasScope(scopes string, requiredScope string) bool {
	scopeList := ParseScopes(scopes)
	for _, s := range scopeList {
		if s == requiredScope {
			return true
		}
	}
	return false
}

// HasAnyScope checks if a scope string contains any of the required scopes.
func HasAnyScope(scopes string, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		if HasScope(scopes, required) {
			return true
		}
	}
	return false
}

// HasAllScopes checks if a scope string contains all required scopes.
func HasAllScopes(scopes string, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		if !HasScope(scopes, required) {
			return false
		}
	}
	return true
}
