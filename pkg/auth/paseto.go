package auth

import (
	"context"
	"errors"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// newPasetoAuthenticator is a stub for PASETO support.
func newPasetoAuthenticator(_ Config) (Authenticator, error) {
	return nil, errors.New("PASETO authenticator not implemented")
}

// PASETOConfig extends the base Config with PASETO-specific settings
type PASETOConfig struct {
	Config
	Version       string // v1 or v2
	Purpose       string // local or public
	SymmetricKey  []byte // for local purpose
	PublicKey     []byte // for public purpose
	PrivateKey    []byte // for public purpose
	TokenStore    token.EnhancedStore
	Footer        string
	Audience      string
	Issuer        string
	TokenValidity time.Duration
}

// pasetoAuthenticator implements the Authenticator interface for PASETO
//
//nolint:unused // reserved for PASETO token implementation
type pasetoAuthenticator struct {
	config PASETOConfig
}

// All PASETO methods are currently stubs. Uncomment and implement as needed.

//nolint:unused // reserved for PASETO token implementation
func newPASETOAuthenticator(_ Config) (Authenticator, error) {
	return nil, errors.New("PASETO authenticator not implemented")
}

//nolint:unused // reserved for PASETO token implementation
func (a *pasetoAuthenticator) Initialize(ctx context.Context) error {
	return nil
}

//nolint:unused // reserved for PASETO token implementation
func (a *pasetoAuthenticator) Close() error {
	return nil
}

// The following methods are intentionally left unimplemented for now.
// Uncomment and implement as needed.
//
//nolint:unused // reserved for PASETO token implementation
func (a *pasetoAuthenticator) GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	return nil, errors.New("not implemented")
}

// func (a *pasetoAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
//     return nil, errors.New("not implemented")
// }

// func (a *pasetoAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
//     return errors.New("not implemented")
// }
