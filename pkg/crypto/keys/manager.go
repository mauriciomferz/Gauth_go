package keys

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// KeyManager defines the interface for cryptographic key operations.
// This abstraction allows swapping local file-based keys with HSMs or Cloud KMS.
type KeyManager interface {
	// Sign signs the data using the managed private key.
	// The mechanism (PSS vs PKCS1v1.5) depends on implementation configuration.
	Sign(ctx context.Context, data []byte) ([]byte, error)

	// GetPublicKey returns the public component of the key.
	GetPublicKey(ctx context.Context) (crypto.PublicKey, error)

	// GetKeyID returns the specific key identifier (kid) for the active key.
	GetKeyID(ctx context.Context) (string, error)

	// CryptoSigner returns the specific key material as a crypto.Signer.
	// This is useful for integration with libraries that expect a crypto.Signer (e.g. jwt-go).
	CryptoSigner(ctx context.Context) (crypto.Signer, error)

	// LookupPublicKey returns the public key associated with the given key ID (kid).
	// It checks both the active key and the previous key (if configured) to enable zero-downtime rotation.
	LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error)
}

// LocalKeyManager implements KeyManager using a local RSA private key file.
type LocalKeyManager struct {
	mu          sync.RWMutex
	privateKey  *rsa.PrivateKey
	previousKey *rsa.PrivateKey // RR-005: Support for previous key
	keyID       string
	prevKeyID   string
	path        string
}

// NewLocalKeyManager creates a new local key manager.
// If path is empty, it defaults to GAUTH_JWT_PRIVKEY_PATH or .keys/jwt_rsa.pem.
// It loads existing keys or generates a new one if missing.
func NewLocalKeyManager(path string) (*LocalKeyManager, error) {
	if path == "" {
		path = os.Getenv("GAUTH_JWT_PRIVKEY_PATH")
	}
	if path == "" {
		path = ".keys/jwt_rsa.pem"
	}

	km := &LocalKeyManager{
		path: path,
	}

	if err := km.loadOrGenerate(); err != nil {
		return nil, err
	}

	// Load Previous Key (Optional)
	km.loadPrevious()

	return km, nil
}

func (km *LocalKeyManager) loadPrevious() {
	prevPath := os.Getenv("GAUTH_JWT_PREV_PRIVKEY_PATH")
	if prevPath == "" {
		return
	}
	// #nosec G304
	b, err := os.ReadFile(prevPath)
	if err != nil {
		return
	}
	block, _ := pem.Decode(b)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return
	}
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return
	}
	km.previousKey = pk

	// Derive previous KID
	prevKid := os.Getenv("GAUTH_JWT_PREV_KID")
	if prevKid != "" {
		km.prevKeyID = prevKid
	} else {
		modBytes := pk.PublicKey.N.Bytes()
		if len(modBytes) >= 8 {
			km.prevKeyID = base64.RawURLEncoding.EncodeToString(modBytes[:8])
		} else {
			km.prevKeyID = "prev-rsa-key"
		}
	}
}

// NewKeyManager creates a KeyManager based on environment configuration.
// Supports "aws" or "local" (default) keys.
func NewKeyManager(ctx context.Context) (KeyManager, error) {
	provider := os.Getenv("GAUTH_KMS_PROVIDER")
	switch provider {
	case "aws":
		keyID := os.Getenv("GAUTH_KMS_KEY_ID")
		region := os.Getenv("GAUTH_KMS_REGION")
		client, err := NewAWSKMSClient(ctx, keyID, region)
		if err != nil {
			return nil, fmt.Errorf("aws kms init: %w", err)
		}
		return NewExternalKeyManager(client), nil
	default:
		return NewLocalKeyManager("")
	}
}

func (km *LocalKeyManager) loadOrGenerate() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Try load
	// #nosec G304
	if b, err := os.ReadFile(km.path); err == nil {
		block, _ := pem.Decode(b)
		if block != nil && block.Type == "RSA PRIVATE KEY" {
			if pk, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
				km.privateKey = pk
				km.deriveKeyID()
				return nil
			}
		}
	}

	// Generate new key
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	// Ensure directory exists
	// #nosec G301
	if err := os.MkdirAll(filepath.Dir(km.path), 0o700); err != nil {
		return fmt.Errorf("mkdir keys: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
	if err := os.WriteFile(km.path, pemBytes, 0o600); err != nil {
		return fmt.Errorf("write key file: %w", err)
	}

	km.privateKey = pk
	km.deriveKeyID()
	return nil
}

func (km *LocalKeyManager) deriveKeyID() {
	// Simple fingerprint: base64url of first 8 bytes of modulus
	// If env GAUTH_JWT_KID is set, use that, otherwise derived.
	kid := os.Getenv("GAUTH_JWT_KID")
	if kid != "" {
		km.keyID = kid
		return
	}

	if km.privateKey != nil {
		modBytes := km.privateKey.PublicKey.N.Bytes()
		if len(modBytes) >= 8 {
			km.keyID = base64.RawURLEncoding.EncodeToString(modBytes[:8])
			return
		}
	}
	km.keyID = "rsa-demo-key"
}

// LookupPublicKey returns the public key matching the kid.
func (km *LocalKeyManager) LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	// Check Active
	if km.keyID == kid && km.privateKey != nil {
		return &km.privateKey.PublicKey, nil
	}

	// Check Previous (RR-005)
	if km.prevKeyID == kid && km.previousKey != nil {
		return &km.previousKey.PublicKey, nil
	}

	// Fallback: If kid is empty, return active (legacy behavior support)
	if kid == "" && km.privateKey != nil {
		return &km.privateKey.PublicKey, nil
	}

	return nil, fmt.Errorf("key not found: %s", kid)
}

// Sign signs data using RSA PKCS1v1.5 (for JWT compatibility default).
func (km *LocalKeyManager) Sign(ctx context.Context, data []byte) ([]byte, error) {
	km.mu.RLock()
	pk := km.privateKey
	km.mu.RUnlock()

	if pk == nil {
		return nil, errors.New("key not loaded")
	}

	// Hash the data first? JWT libraries usually pass the digest or the payload?
	// crypto.Signer Sign method expects digest.
	// BUT, if this is for JWT signing, standard library usually does the hashing inside if we use golang-jwt/jwt.
	// Wait, if we are implementing a signer for `jwt.SigningMethodRSA`, we usually provide the *rsa.PrivateKey to the library.
	// But here we are abstracting it.
	// If we are abstracting, we might want to return a `crypto.Signer`?
	// `rsa.PrivateKey` implements `crypto.Signer`.
	// For this interface, let's assume we are signing a digest (SHA256).

	// WARNING: If this method is called with raw payload, we must hash it.
	// If it's called with digest, we sign it.
	// To be safe and standard with `crypto.Signer`, let's assume `data` is the digest if options say so,
	// or we hash it here?
	// Let's implement it as `crypto.Signer.Sign` expects: digest.
	// Usually `rand` is required.

	// Actually, simpler: return the `crypto.Signer` is usually the best interface for go.
	// But `KeyManager` is our custom one.
	// Let's stick to simple Sign with SHA256 for now as default for GAuth.

	hashed := crypto.SHA256.New()
	hashed.Write(data)
	digest := hashed.Sum(nil)

	return rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, digest)
}

// GetPublicKey returns the public key.
func (km *LocalKeyManager) GetPublicKey(ctx context.Context) (crypto.PublicKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	if km.privateKey == nil {
		return nil, errors.New("key not loaded")
	}
	return &km.privateKey.PublicKey, nil
}

// GetKeyID returns the key ID.
func (km *LocalKeyManager) GetKeyID(ctx context.Context) (string, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.keyID, nil
}

// CryptoSigner returns the underlying private key as crypto.Signer if possible.
// This is useful for integration with libraries like golang-jwt.
func (km *LocalKeyManager) CryptoSigner(ctx context.Context) (crypto.Signer, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	if km.privateKey == nil {
		return nil, errors.New("key not loaded")
	}
	return km.privateKey, nil
}

// PublicJWK generates a JWK map for the active key in the manager.
// Currently supports RSA keys.
func PublicJWK(km KeyManager) (map[string]any, error) {
	pub, err := km.GetPublicKey(context.Background())
	if err != nil {
		return nil, err
	}
	kid, err := km.GetKeyID(context.Background())
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("unsupported key type for JWK (only RSA supported)")
	}

	n := base64.RawURLEncoding.EncodeToString(rsaPub.N.Bytes())
	eBytes := []byte{byte(rsaPub.E >> 16), byte(rsaPub.E >> 8), byte(rsaPub.E)}
	// Trim leading zeros
	i := 0
	for i < len(eBytes)-1 && eBytes[i] == 0 {
		i++
	}
	e := base64.RawURLEncoding.EncodeToString(eBytes[i:])

	return map[string]any{"kty": "RSA", "alg": "RS256", "kid": kid, "use": "sig", "n": n, "e": e}, nil
}
