package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken indicates the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired indicates the token has expired
	ErrTokenExpired = errors.New("token expired")
	// ErrInvalidSignature indicates the token signature is invalid
	ErrInvalidSignature = errors.New("invalid signature")
	// ErrInvalidClaims indicates the token claims are invalid
	ErrInvalidClaims = errors.New("invalid claims")
)

// jwtAuthenticator implements the Authenticator interface using JWT
type jwtAuthenticator struct {
	config    Config
	signKey   interface{}
	verifyKey interface{}
}

// jwtClaims represents JWT claims
type jwtClaims struct {
	jwt.RegisteredClaims
	Scope  []string               `json:"scope,omitempty"`
	Claims map[string]interface{} `json:"claims,omitempty"`
}

// newJWTAuthenticator creates a new JWT authenticator
func newJWTAuthenticator(config Config) (Authenticator, error) {
	// For now using HMAC, but can be extended to support RSA
	key := []byte(config.ClientSecret)
	return &jwtAuthenticator{
		config:    config,
		signKey:   key,
		verifyKey: key,
	}, nil
}

func (a *jwtAuthenticator) Initialize(ctx context.Context) error {
	// No initialization needed for now
	return nil
}

func (a *jwtAuthenticator) Close() error {
	// No cleanup needed for now
	return nil
}

func (a *jwtAuthenticator) ValidateCredentials(ctx context.Context, creds interface{}) error {
	// JWT authenticator doesn't handle credentials directly
	return errors.New("credential validation not supported by JWT authenticator")
}

func (a *jwtAuthenticator) GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	now := time.Now()
	expiresIn := a.config.AccessTokenExpiry
	if req.ExpiresIn > 0 {
		expiresIn = req.ExpiresIn
	}

	claims := &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.config.ClientID,
			Subject:   req.Subject,
			Audience:  jwt.ClaimStrings{req.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		Scope:  req.Scopes,
		Claims: req.Metadata,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.signKey)
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	if a.config.AuditLogger != nil {
		entry := &audit.Entry{
			Type:    audit.TypeToken,
			Action:  audit.ActionTokenGenerate,
			ActorID: req.Subject,
			Result:  audit.ResultSuccess,
			Metadata: audit.Metadata{
				"audience": req.Audience,
				"scope":    fmt.Sprintf("%v", req.Scopes),
			},
		}
		a.config.AuditLogger.Log(ctx, entry)
	}

	return &TokenResponse{
		Token:     signed,
		TokenType: "Bearer",
		ExpiresIn: int64(expiresIn.Seconds()),
		Scope:     req.Scopes,
		Claims:    req.Metadata,
	}, nil
}

func (a *jwtAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
	token, err := a.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}

	claims, err := a.extractTokenClaims(token)
	if err != nil {
		return nil, err
	}

	if err := a.validateTokenClaims(claims); err != nil {
		return nil, err
	}

	a.logTokenValidation(ctx, claims)

	return a.createTokenData(claims), nil
}

func (a *jwtAuthenticator) parseToken(tokenStr string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return a.verifyKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return token, nil
}

func (a *jwtAuthenticator) extractTokenClaims(token *jwt.Token) (*jwtClaims, error) {
	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}
	return claims, nil
}

func (a *jwtAuthenticator) validateTokenClaims(claims *jwtClaims) error {
	if err := a.validateIssuerClaim(claims); err != nil {
		return err
	}

	if err := a.validateAudienceClaim(claims); err != nil {
		return err
	}

	if err := a.validateScopesClaim(claims); err != nil {
		return err
	}

	return a.validateCustomClaims(claims)
}

func (a *jwtAuthenticator) validateIssuerClaim(claims *jwtClaims) error {
	if len(a.config.TokenValidation.AllowedIssuers) == 0 {
		return nil
	}

	for _, issuer := range a.config.TokenValidation.AllowedIssuers {
		if claims.Issuer == issuer {
			return nil
		}
	}
	return fmt.Errorf("invalid issuer: %s", claims.Issuer)
}

func (a *jwtAuthenticator) validateAudienceClaim(claims *jwtClaims) error {
	if len(a.config.TokenValidation.AllowedAudiences) == 0 {
		return nil
	}

	for _, aud := range a.config.TokenValidation.AllowedAudiences {
		if contains(claims.Audience, aud) {
			return nil
		}
	}
	return fmt.Errorf("invalid audience: %v", claims.Audience)
}

func (a *jwtAuthenticator) validateScopesClaim(claims *jwtClaims) error {
	for _, scope := range a.config.TokenValidation.RequiredScopes {
		if !contains(claims.Scope, scope) {
			return fmt.Errorf("missing required scope: %s", scope)
		}
	}
	return nil
}

func (a *jwtAuthenticator) validateCustomClaims(claims *jwtClaims) error {
	for claim, value := range a.config.TokenValidation.RequiredClaims {
		if claims.Claims[claim] != value {
			return fmt.Errorf("invalid claim value for %s", claim)
		}
	}
	return nil
}

func (a *jwtAuthenticator) logTokenValidation(ctx context.Context, claims *jwtClaims) {
	if a.config.AuditLogger == nil {
		return
	}

	entry := &audit.Entry{
		Type:    audit.TypeToken,
		Action:  audit.ActionTokenValidate,
		ActorID: claims.Subject,
		Result:  audit.ResultSuccess,
		Metadata: audit.Metadata{
			"issuer":   claims.Issuer,
			"audience": fmt.Sprintf("%v", claims.Audience),
			"scope":    fmt.Sprintf("%v", claims.Scope),
		},
	}
	a.config.AuditLogger.Log(ctx, entry)
}

func (a *jwtAuthenticator) createTokenData(claims *jwtClaims) *TokenData {
	return &TokenData{
		Valid:     true,
		Subject:   claims.Subject,
		Issuer:    claims.Issuer,
		Audience:  claims.Audience[0],
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: claims.ExpiresAt.Time,
		Scope:     claims.Scope,
		Claims:    claims.Claims,
	}
}

func (a *jwtAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
	// For a simple JWT implementation, we don't handle revocation
	// In a real implementation, we would add the token to a blacklist
	// or use a distributed token store for revocation
	return errors.New("token revocation not supported by simple JWT authenticator")
}

// Utility functions
func generateTokenID() string {
	// Use a proper ID generation in production
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
