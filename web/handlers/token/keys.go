package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var (
	rsaOnce sync.Once
	rsaKey  *rsa.PrivateKey
	rsaErr  error
)

// loadOrGenerateRSAKey loads an RSA private key from GAUTH_JWT_PRIVKEY_PATH or generates and persists one.
// Key size: 2048 bits (sufficient for demo). For production use 3072/4096.
func LoadOrGenerateRSAKey() (*rsa.PrivateKey, error) {
	rsaOnce.Do(func() {
		path := os.Getenv("GAUTH_JWT_PRIVKEY_PATH")
		if path == "" {
			path = ".keys/jwt_rsa.pem"
		}
		// Try load
		if b, err := os.ReadFile(path); err == nil {
			block, _ := pem.Decode(b)
			if block != nil && block.Type == "RSA PRIVATE KEY" {
				if pk, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
					rsaKey = pk
					return
				}
			}
		}
		// Generate new key
		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			rsaErr = err
			return
		}
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			rsaErr = err
			return
		}
		pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
		if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
			rsaErr = err
			return
		}
		rsaKey = pk
	})
	if rsaErr != nil {
		return nil, rsaErr
	}
	if rsaKey == nil {
		return nil, errors.New("rsa key not available")
	}
	return rsaKey, nil
}

// rsaPublicJWK returns a JWK map for the active RSA key (kty: RSA, alg RS256, kid from env or fingerprint).
func rsaPublicJWK() (map[string]any, error) {
	pk, err := LoadOrGenerateRSAKey()
	if err != nil {
		return nil, err
	}
	kid := os.Getenv("GAUTH_JWT_KID")
	if kid == "" {
		// Simple fingerprint: base64url of first 8 bytes of modulus
		modBytes := pk.PublicKey.N.Bytes()
		if len(modBytes) >= 8 {
			kid = base64.RawURLEncoding.EncodeToString(modBytes[:8])
		} else {
			kid = "rsa-demo-key"
		}
	}
	n := base64.RawURLEncoding.EncodeToString(pk.PublicKey.N.Bytes())
	eBytes := []byte{byte(pk.PublicKey.E >> 16), byte(pk.PublicKey.E >> 8), byte(pk.PublicKey.E)}
	// Trim leading zeros
	i := 0
	for i < len(eBytes)-1 && eBytes[i] == 0 {
		i++
	}
	e := base64.RawURLEncoding.EncodeToString(eBytes[i:])
	return map[string]any{"kty": "RSA", "alg": "RS256", "kid": kid, "use": "sig", "n": n, "e": e}, nil
}
