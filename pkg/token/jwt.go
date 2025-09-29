//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package token

import (
	"crypto"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSigner handles JWT token signing and verification
type JWTSigner struct {
	signingKey crypto.Signer
	verifyKey  crypto.PublicKey
	signingAlg Algorithm
	keyID      string
}

// NewJWTSigner creates a new JWT token signer
func NewJWTSigner(signingKey crypto.Signer, alg Algorithm) *JWTSigner {
	return &JWTSigner{
		signingKey: signingKey,
		verifyKey:  signingKey.Public(),
		signingAlg: alg,
	}
}

// WithKeyID sets the key ID used in the JWT header
func (s *JWTSigner) WithKeyID(kid string) *JWTSigner {
	s.keyID = kid
	return s
}

// SignToken signs a token using JWT
func (s *JWTSigner) SignToken(token *Token) (string, error) {

	claims := jwt.MapClaims{
		"jti": token.ID,
		"typ": string(token.Type),
		"sub": token.Subject,
		"iss": token.Issuer,
		"aud": token.Audience,
		"iat": token.IssuedAt.Unix(),
		"nbf": token.NotBefore.Unix(),
		"exp": token.ExpiresAt.Unix(),
		"scp": token.Scopes,
	}

	// Add metadata as a single 'meta' claim where appropriate
	if token.Metadata != nil {
		if token.Metadata.AppData != nil {
			claims["meta"] = token.Metadata.AppData
		}
	}

	jwtToken := jwt.NewWithClaims(jwtSigningMethod(s.signingAlg), claims)

	if s.keyID != "" {
		jwtToken.Header["kid"] = s.keyID
	}

	signedToken, err := jwtToken.SignedString(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

// VerifyToken verifies and parses a JWT token string
func (s *JWTSigner) VerifyToken(tokenString string) (*Token, error) {
	jwtToken, err := s.parseJWTToken(tokenString)
	if err != nil {
		return nil, err
	}

	claims, err := s.extractClaims(jwtToken)
	if err != nil {
		return nil, err
	}

	token := s.createTokenFromClaims(tokenString, claims)
	return token, nil
}

func (s *JWTSigner) parseJWTToken(tokenString string) (*jwt.Token, error) {
	jwtToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwtSigningMethod(s.signingAlg) {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.verifyKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if !jwtToken.Valid {
		return nil, ErrInvalidToken
	}

	return jwtToken, nil
}

func (s *JWTSigner) extractClaims(jwtToken *jwt.Token) (jwt.MapClaims, error) {
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}

func (s *JWTSigner) createTokenFromClaims(tokenString string, claims jwt.MapClaims) *Token {
	token := &Token{
		Algorithm: s.signingAlg,
		Value:     tokenString,
		Metadata:  &Metadata{},
	}

	s.parseStandardClaims(token, claims)
	s.parseCustomClaims(token, claims)

	return token
}

func (s *JWTSigner) parseStandardClaims(token *Token, claims jwt.MapClaims) {
	if id, ok := claims["jti"].(string); ok {
		token.ID = id
	}
	if typ, ok := claims["typ"].(string); ok {
		token.Type = Type(typ)
	}
	if sub, ok := claims["sub"].(string); ok {
		token.Subject = sub
	}
	if iss, ok := claims["iss"].(string); ok {
		token.Issuer = iss
	}

	s.parseAudienceClaim(token, claims)
	s.parseTimeClaims(token, claims)
	s.parseScopesClaim(token, claims)
}

func (s *JWTSigner) parseAudienceClaim(token *Token, claims jwt.MapClaims) {
	if aud, ok := claims["aud"].([]interface{}); ok {
		token.Audience = make([]string, len(aud))
		for i, a := range aud {
			if s, ok := a.(string); ok {
				token.Audience[i] = s
			}
		}
	}
}

func (s *JWTSigner) parseTimeClaims(token *Token, claims jwt.MapClaims) {
	if iat, ok := claims["iat"].(float64); ok {
		token.IssuedAt = time.Unix(int64(iat), 0)
	}
	if nbf, ok := claims["nbf"].(float64); ok {
		token.NotBefore = time.Unix(int64(nbf), 0)
	}
	if exp, ok := claims["exp"].(float64); ok {
		token.ExpiresAt = time.Unix(int64(exp), 0)
	}
}

func (s *JWTSigner) parseScopesClaim(token *Token, claims jwt.MapClaims) {
	if scp, ok := claims["scp"].([]interface{}); ok {
		token.Scopes = make([]string, len(scp))
		for i, s := range scp {
			if scope, ok := s.(string); ok {
				token.Scopes[i] = scope
			}
		}
	}
}

func (s *JWTSigner) parseCustomClaims(token *Token, claims jwt.MapClaims) {
	if metaVal, ok := claims["meta"].(map[string]interface{}); ok {
		appData := make(map[string]string, len(metaVal))
		for k, v := range metaVal {
			if s, ok := v.(string); ok {
				appData[k] = s
			} else if b, err := json.Marshal(v); err == nil {
				appData[k] = string(b)
			}
		}
		token.Metadata.AppData = appData
	}
}

// jwtSigningMethod converts our Algorithm type to jwt.SigningMethod
func jwtSigningMethod(alg Algorithm) jwt.SigningMethod {
	switch alg {
	case RS256:
		return jwt.SigningMethodRS256
	case ES256:
		return jwt.SigningMethodES256
	case HS256:
		return jwt.SigningMethodHS256
	case PS256:
		return jwt.SigningMethodPS256
	default:
		return jwt.SigningMethodRS256
	}
}

// helper function for future JWT validation
func _isStandardClaim(claim string) bool {
	standardClaims := map[string]bool{
		"jti": true, "sub": true, "iss": true, "typ": true,
		"iat": true, "nbf": true, "exp": true, "scp": true,
	}
	return standardClaims[claim]
}
