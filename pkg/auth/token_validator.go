package auth
p
import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

// TokenValidator handles token validation logic
type TokenValidator struct {
	config *Config
}

// NewTokenValidator creates a new token validator
func NewTokenValidator(config *Config) *TokenValidator {
	return &TokenValidator{config: config}
}

// ValidationResult represents the result of token validation
type ValidationResult struct {
	Valid    bool
	Claims   map[string]interface{}
	Subject  string
	ExpireAt time.Time
	Error    error
}

// ValidateToken performs comprehensive token validation
func (v *TokenValidator) ValidateToken(ctx context.Context, tokenStr string) *ValidationResult {
	result := &ValidationResult{
		Valid: false,
	}

	// Parse token without validating signature first
	token, _ := jwt.Parse(tokenStr, nil)
	if token == nil {
		result.Error = ErrInvalidToken
		return result
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		result.Claims = claims
	} else {
		result.Error = ErrInvalidClaims
		return result
	}

	// Validate signature
	if err := v.validateSignature(token); err != nil {
		result.Error = err
		return result
	}

	// Validate expiration
	if err := v.validateExpiration(result.Claims); err != nil {
		result.Error = err
		return result
	}

	// Validate issuer if configured
	if err := v.validateIssuer(result.Claims); err != nil {
		result.Error = err
		return result
	}

	// Extract standard claims
	result.Subject = v.extractSubject(result.Claims)
	result.ExpireAt = v.extractExpiration(result.Claims)
	result.Valid = true

	return result
}

func (v *TokenValidator) validateSignature(token *jwt.Token) error {
	if token.Method.Alg() != v.config.SigningMethod {
		return ErrInvalidSigningMethod
	}

	if _, err := token.Method.Verify(token.Raw, token.Signature, v.config.SigningKey); err != nil {
		return ErrInvalidSignature
	}

	return nil
}

func (v *TokenValidator) validateExpiration(claims map[string]interface{}) error {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return ErrMissingExpiration
	}

	if time.Unix(int64(exp), 0).Before(time.Now()) {
		return ErrTokenExpired
	}

	return nil
}

func (v *TokenValidator) validateIssuer(claims map[string]interface{}) error {
	if v.config.RequireIssuer == "" {
		return nil
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != v.config.RequireIssuer {
		return ErrInvalidIssuer
	}

	return nil
}

func (v *TokenValidator) extractSubject(claims map[string]interface{}) string {
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

func (v *TokenValidator) extractExpiration(claims map[string]interface{}) time.Time {
	if exp, ok := claims["exp"].(float64); ok {
		return time.Unix(int64(exp), 0)
	}
	return time.Time{}
}

// Common token validation errors
var (
	ErrInvalidToken         = errors.New("invalid token format")
	ErrInvalidClaims       = errors.New("invalid token claims")
	ErrInvalidSignature    = errors.New("invalid token signature")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
	ErrTokenExpired        = errors.New("token has expired")
	ErrMissingExpiration   = errors.New("token missing expiration claim")
	ErrInvalidIssuer       = errors.New("invalid token issuer")
)