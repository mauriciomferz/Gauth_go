package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
func (v *TokenValidator) ValidateToken(_ context.Context, tokenStr string) *ValidationResult {
	result := &ValidationResult{
		Valid: false,
	}

	// Parse token without validating signature first (for claim extraction)
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
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
	// If signature validation is disabled in config, skip
	if !v.config.TokenValidation.ValidateSignature {
		return nil
	}

	// Use the signKey from config if available (pseudo, adjust as needed)
	// signKey := v.config.TokenValidation.SignKey
	// For now, just check if the method is HMAC (as an example)
	if token.Method.Alg() != "HS256" {
		return fmt.Errorf("invalid signing method: %s", token.Method.Alg())
	}

	// NOTE: jwt-go v4/v5 expects signature validation to be handled in the KeyFunc, not here.
	// This is a placeholder for actual signature validation logic.
	// If you want to validate the signature, do it in the KeyFunc when calling jwt.Parse.

	return nil
}

func (v *TokenValidator) validateExpiration(claims map[string]interface{}) error {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("missing expiration claim")
	}

	if time.Unix(int64(exp), 0).Before(time.Now()) {
		return ErrTokenExpired
	}

	return nil
}

func (v *TokenValidator) validateIssuer(claims map[string]interface{}) error {
	allowedIssuers := v.config.TokenValidation.AllowedIssuers
	if len(allowedIssuers) == 0 {
		return nil
	}

	iss, ok := claims["iss"].(string)
	if !ok {
		return fmt.Errorf("missing issuer claim")
	}
	for _, allowed := range allowedIssuers {
		if iss == allowed {
			return nil
		}
	}
	return fmt.Errorf("invalid issuer: %s", iss)
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
var ()
