package auth

import (
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



// All PASETO methods are currently stubs. Uncomment and implement as needed.



// PASETO implementation reserved for future development

// func (a *pasetoAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
//     return nil, errors.New("not implemented")
// }

// func (a *pasetoAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
//     return errors.New("not implemented")
// }
