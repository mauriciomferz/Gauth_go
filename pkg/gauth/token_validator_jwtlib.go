package gauth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// JWTLibValidator validates tokens using the golang-jwt/jwt library
// Implements TokenSignatureValidator interface
type JWTLibValidator struct {
	signingKey  []byte
	expectedAlg string
}

// NewJWTLibValidator creates a new JWT library validator
func NewJWTLibValidator(signingKey []byte, expectedAlg string) *JWTLibValidator {
	if expectedAlg == "" {
		expectedAlg = "HS256"
	}
	return &JWTLibValidator{
		signingKey:  signingKey,
		expectedAlg: expectedAlg,
	}
}

// ValidateSignature validates JWT signature using jwt library and extracts claims
func (v *JWTLibValidator) ValidateSignature(token string) (map[string]any, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != v.expectedAlg {
			return nil, fmt.Errorf("unexpected alg %s", t.Method.Alg())
		}
		return v.signingKey, nil
	})

	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Convert jwt.MapClaims to map[string]any
	result := make(map[string]any)
	for k, v := range claims {
		result[k] = v
	}

	return result, nil
}

// Name returns the validator name
func (v *JWTLibValidator) Name() string {
	return "JWTLib"
}
