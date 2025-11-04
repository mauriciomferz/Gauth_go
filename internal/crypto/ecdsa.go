package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
)

// ECDSAKey represents an ECDSA key pair
// Curve: P-256, P-384, P-521
type ECDSAKey struct {
	Private *ecdsa.PrivateKey
	Public  *ecdsa.PublicKey
	Curve   string
}

// GenerateECDSAKey generates a new ECDSA key pair for the given curve
func GenerateECDSAKey(curve string) (*ECDSAKey, error) {
	var c elliptic.Curve
	switch curve {
	case "P-256":
		c = elliptic.P256()
	case "P-384":
		c = elliptic.P384()
	case "P-521":
		c = elliptic.P521()
	default:
		return nil, ErrUnsupportedCurve
	}
	priv, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ECDSAKey{Private: priv, Public: &priv.PublicKey, Curve: curve}, nil
}

// SignECDSA signs a message using the ECDSA private key
func SignECDSA(key *ECDSAKey, message []byte) (string, string, error) {
	hash := sha256.Sum256(message)
	r, s, err := ecdsa.Sign(rand.Reader, key.Private, hash[:])
	if err != nil {
		return "", "", err
	}
	return base64.RawURLEncoding.EncodeToString(r.Bytes()), base64.RawURLEncoding.EncodeToString(s.Bytes()), nil
}

// VerifyECDSA verifies an ECDSA signature
func VerifyECDSA(key *ECDSAKey, message []byte, rB64, sB64 string) bool {
	rBytes, err := base64.RawURLEncoding.DecodeString(rB64)
	if err != nil {
		return false
	}
	sBytes, err := base64.RawURLEncoding.DecodeString(sB64)
	if err != nil {
		return false
	}
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)
	hash := sha256.Sum256(message)
	return ecdsa.Verify(key.Public, hash[:], r, s)
}

var ErrUnsupportedCurve = fmt.Errorf("unsupported ECDSA curve")
