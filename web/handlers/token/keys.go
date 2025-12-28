package token

import (
	"context"
	"crypto/rsa"
	"errors"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto/keys"
)

// LoadOrGenerateRSAKey is DEPRECATED. Use keys.NewLocalKeyManager instead.
// It is kept here for backward compatibility with existing tests that need raw key access.
func LoadOrGenerateRSAKey() (*rsa.PrivateKey, error) {
	// We use the default path logic from LocalKeyManager
	km, err := keys.NewLocalKeyManager("")
	if err != nil {
		return nil, err
	}

	signer, err := km.CryptoSigner(context.Background())
	if err != nil {
		return nil, err
	}

	rsaKey, ok := signer.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("underlying key is not RSA")
	}
	return rsaKey, nil
}

// rsaPublicJWK is DEPRECATED. The handler now constructs JWKs from the injected KeyManager.
// Keeping it temporarily if needed, but optimally we remove its usage.
// Assuming we refactor the usage in api.go, we can remove this or make it use the shim.
func rsaPublicJWK() (map[string]any, error) {
	km, err := keys.NewLocalKeyManager("")
	if err != nil {
		return nil, err
	}
	return keys.PublicJWK(km)
}
