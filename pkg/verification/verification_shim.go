package verification

// Shim exposing a narrow read-only projection of the global EdDSA registry.

import (
	"crypto/ed25519"

	cryptoInt "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
)

func findPublicKey(kid string) *publicKeyWrapper {
	km := cryptoInt.GlobalEdDSARegistry
	if km == nil {
		return nil
	}
	k := km.FindByID(kid)
	if k == nil || len(k.Public) != ed25519.PublicKeySize {
		return nil
	}
	return &publicKeyWrapper{Public: k.Public}
}
