package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"math/big"
	"testing"
)

// TestPhase1Vectors iterates dynamically generated Phase 1 vectors
// ensuring verification outcome matches the declared Valid flag.
func TestPhase1Vectors(t *testing.T) {
	vectors, err := GeneratePhase1Vectors()
	if err != nil {
		t.Fatalf("generate vectors: %v", err)
	}
	for i, v := range vectors {
		var ok bool
		switch v.Alg {
		case "Ed25519":
			if len(v.Public) == ed25519.PublicKeySize {
				ok = ed25519.Verify(ed25519.PublicKey(v.Public), v.Message, v.Signature)
			}
		case "ECDSA-P256":
			// Manual uncompressed SEC1 decode (0x04 || X || Y)
			if len(v.Public) == 1+32+32 && v.Public[0] == 0x04 {
				x := new(big.Int).SetBytes(v.Public[1:33])
				y := new(big.Int).SetBytes(v.Public[33:65])
				pub := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
				r, s, derr := decodeDERSignature(v.Signature)
				if derr == nil && !isHighS(s, pub.Params().N) {
					// Hash message the same way signer does (sha256)
					// signer.go does hash inside Sign/Verify; we rely on Verify logic indirectly via ecdsa.Verify here
					// but to avoid reimplement hashing normalization we reuse ecdsa.Verify by reconstructing digest
					// We'll simply call a Signer instance for consistency.
					signer := NewP256Signer(nil, pub)
					ok = signer.Verify(v.Message, v.Signature)
					// Double-check r,s path (redundant but ensures decode applied)
					_ = r
					_ = s
				}
			}
		case "BLS12-381":
			pubKey, perr := NewBLSPublicKey(v.Public)
			if perr == nil {
				ok = BLSVerify(pubKey, v.Message, v.Signature)
			}
		default:
			t.Fatalf("unknown algorithm in vector: %s", v.Alg)
		}
		if ok != v.Valid {
			t.Fatalf("vector %d (%s) expected valid=%v got %v", i, v.Alg, v.Valid, ok)
		}
	}
}
