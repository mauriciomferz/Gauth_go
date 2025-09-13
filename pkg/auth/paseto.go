package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
	"github.com/o1egl/paseto"
)

// PASETOConfig extends the base Config with PASETO-specific settings
type PASETOConfig struct {
	Config
	Version       string // v1 or v2
	Purpose       string // local or public
	SymmetricKey  []byte // for local purpose
	PublicKey     []byte // for public purpose
	PrivateKey    []byte // for public purpose
	TokenStore    TokenStore
	Footer        string
	Audience      string
	Issuer        string
	TokenValidity time.Duration
}

// pasetoAuthenticator implements the Authenticator interface for PASETO
type pasetoAuthenticator struct {
	config     PASETOConfig
	tokenMaker *paseto.V2
}

func newPASETOAuthenticator(config Config) (Authenticator, error) {
	pConfig, ok := config.ExtraConfig.(PASETOConfig)
	if !ok {
		return nil, errors.New("invalid PASETO config")
	}

	switch pConfig.Version {
	case "v2":
		if pConfig.Purpose == "local" && len(pConfig.SymmetricKey) == 0 {
			return nil, errors.New("symmetric key required for local purpose")
		}
		if pConfig.Purpose == "public" && (len(pConfig.PublicKey) == 0 || len(pConfig.PrivateKey) == 0) {
			return nil, errors.New("public/private key pair required for public purpose")
		}
	default:
		return nil, fmt.Errorf("unsupported PASETO version: %s", pConfig.Version)
	}

	return &pasetoAuthenticator{
		config:     pConfig,
		tokenMaker: paseto.NewV2(),
	}, nil
}

func (a *pasetoAuthenticator) Initialize(ctx context.Context) error {
	if a.config.TokenStore != nil {
		return a.config.TokenStore.Initialize(ctx)
	}
	return nil
}

func (a *pasetoAuthenticator) Close() error {
	if a.config.TokenStore != nil {
		return a.config.TokenStore.Close()
	}
	return nil
}

func (a *pasetoAuthenticator) ValidateCredentials(ctx context.Context, creds interface{}) error {
	// PASETO doesn't handle credential validation directly
	// This should be handled by the application
	return fmt.Errorf("credential validation not supported by PASETO authenticator")
}

func (a *pasetoAuthenticator) GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	now := time.Now()
	claims := map[string]interface{}{
		"sub": req.Subject,
		"iat": now.Unix(),
		"exp": now.Add(a.config.TokenValidity).Unix(),
	}

	if a.config.Audience != "" {
		claims["aud"] = a.config.Audience
	}
	if a.config.Issuer != "" {
		claims["iss"] = a.config.Issuer
	}
	if len(req.Scopes) > 0 {
		claims["scope"] = req.Scopes
	}

	var token string
	var err error

	switch a.config.Purpose {
	case "local":
		token, err = a.tokenMaker.Encrypt(a.config.SymmetricKey, claims, a.config.Footer)
	case "public":
		token, err = a.tokenMaker.Sign(a.config.PrivateKey, claims, a.config.Footer)
	default:
		return nil, fmt.Errorf("unsupported purpose: %s", a.config.Purpose)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate PASETO token: %w", err)
	}

	tokenResp := &TokenResponse{
		Token:     token,
		TokenType: "PASETO",
		ExpiresIn: int64(a.config.TokenValidity.Seconds()),
		Scope:     req.Scopes,
		Claims:    claims,
	}

	if a.config.TokenStore != nil {
		if err := a.config.TokenStore.Store(ctx, tokenResp); err != nil {
			return nil, fmt.Errorf("failed to store token: %w", err)
		}
	}

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(audit.Event{
			Type:    audit.TypeToken,
			Action:  audit.ActionTokenGenerate,
			ActorID: req.Subject,
			Result:  audit.ResultSuccess,
			Metadata: map[string]string{
				"token_type": "PASETO",
				"version":    a.config.Version,
				"purpose":    a.config.Purpose,
			},
		})
	}

	return tokenResp, nil
}

func (a *pasetoAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
	if a.config.TokenStore != nil {
		// Check if token is in the store first
		data, err := a.config.TokenStore.Get(ctx, tokenStr)
		if err != nil {
			return nil, fmt.Errorf("token not found in store: %w", err)
		}
		return data, nil
	}

	var claims map[string]interface{}
	var err error

	switch a.config.Purpose {
	case "local":
		err = a.tokenMaker.Decrypt(tokenStr, a.config.SymmetricKey, &claims, a.config.Footer)
	case "public":
		err = a.tokenMaker.Verify(tokenStr, a.config.PublicKey, &claims, a.config.Footer)
	default:
		return nil, fmt.Errorf("unsupported purpose: %s", a.config.Purpose)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to validate PASETO token: %w", err)
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid expiration claim")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, errors.New("invalid issued at claim")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid subject claim")
	}

	var scope []string
	if s, ok := claims["scope"].([]interface{}); ok {
		scope = make([]string, len(s))
		for i, v := range s {
			scope[i] = v.(string)
		}
	}

	tokenData := &TokenData{
		Valid:     true,
		Subject:   sub,
		IssuedAt:  time.Unix(int64(iat), 0),
		ExpiresAt: time.Unix(int64(exp), 0),
		Scope:     scope,
	}

	if iss, ok := claims["iss"].(string); ok {
		tokenData.Issuer = iss
	}

	if aud, ok := claims["aud"].(string); ok {
		tokenData.Audience = aud
	}

	return tokenData, nil
}

func (a *pasetoAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
	if a.config.TokenStore != nil {
		if err := a.config.TokenStore.Remove(ctx, tokenStr); err != nil {
			return fmt.Errorf("failed to remove token from store: %w", err)
		}
	}

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(audit.Event{
			Type:   audit.TypeToken,
			Action: audit.ActionTokenRevoke,
			Result: audit.ResultSuccess,
		})
	}

	return nil
}
