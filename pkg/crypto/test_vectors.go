package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

// TestVector represents a cross-language test vector for cryptographic primitives.
type TestVector struct {
	Alg       string // "Ed25519", "ECDSA-P256", "BLS12-381"
	Curve     string // e.g. "P-256" for ECDSA
	Message   []byte
	Private   []byte // optional (may be absent for verify-only vectors)
	Public    []byte
	Signature []byte
	Valid     bool // expected verification result
}

// EnforcementFlags holds compliance/authenticity enforcement settings.
type EnforcementFlags struct {
	RequireSignature         bool
	RequireBatchVerification bool
	RequireKeyRotation       bool
}

// GeneratePhase1Vectors creates a slice of positive and negative test vectors
// for Ed25519, ECDSA-P256 (including high-S rejection), and BLS12-381.
// NOTE: ECDSA signatures are non-deterministic (nonce) in Phase 1; vectors are
// generated dynamically at test time rather than static fixtures.
func GeneratePhase1Vectors() ([]TestVector, error) {
	vectors := make([]TestVector, 0, 10)

	// --- Ed25519 ---
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	edMsg := []byte("tv-ed25519-1")
	edSig := ed25519.Sign(edPriv, edMsg)
	vectors = append(vectors, TestVector{Alg: "Ed25519", Message: edMsg, Private: edPriv, Public: edPub, Signature: edSig, Valid: true})
	// Negative: mutate one byte
	badEd := append([]byte(nil), edSig...)
	badEd[0] ^= 0xFF
	vectors = append(vectors, TestVector{Alg: "Ed25519", Message: edMsg, Public: edPub, Signature: badEd, Valid: false})

	// --- ECDSA P-256 ---
	ecdsaPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	eMsg := []byte("tv-ecdsa-1")
	h := sha256.Sum256(eMsg)
	r, s, err := ecdsa.Sign(rand.Reader, ecdsaPriv, h[:])
	if err != nil {
		return nil, err
	}
	// Canonical low-S form
	lowS := normalizeLowS(s, ecdsaPriv.Params().N)
	derLow := encodeDERSignature(r, lowS)
	// Uncompressed form: 0x04 || X || Y (32 bytes each for P-256)
	byteLen := (ecdsaPriv.PublicKey.Curve.Params().BitSize + 7) / 8
	pubBytes := make([]byte, 1+2*byteLen)
	pubBytes[0] = 0x04
	ecdsaPriv.PublicKey.X.FillBytes(pubBytes[1 : 1+byteLen])
	ecdsaPriv.PublicKey.Y.FillBytes(pubBytes[1+byteLen:])
	vectors = append(vectors, TestVector{Alg: "ECDSA-P256", Curve: "P-256", Message: eMsg, Public: pubBytes, Signature: derLow, Valid: true})
	// High-S variant (malleable) - should be invalid
	highS := new(big.Int).Sub(ecdsaPriv.Params().N, lowS)
	half := new(big.Int).Rsh(ecdsaPriv.Params().N, 1)
	if highS.Cmp(half) == 1 { // Only append if truly high
		derHigh := encodeDERSignature(r, highS)
		vectors = append(vectors, TestVector{Alg: "ECDSA-P256", Curve: "P-256", Message: eMsg, Public: pubBytes, Signature: derHigh, Valid: false})
	}
	// Truncated signature
	truncated := derLow[:len(derLow)/2]
	vectors = append(vectors, TestVector{Alg: "ECDSA-P256", Curve: "P-256", Message: eMsg, Public: pubBytes, Signature: truncated, Valid: false})

	// --- BLS12-381 ---
	blsKey, err := GenerateBLSKey()
	if err != nil {
		return nil, err
	}
	bMsg := []byte("tv-bls-1")
	bSig, err := BLSSign(blsKey, bMsg)
	if err != nil {
		return nil, err
	}
	pubBLS := blsKey.Public.Serialize()
	vectors = append(vectors, TestVector{Alg: "BLS12-381", Message: bMsg, Public: pubBLS, Signature: bSig, Valid: true})
	// Negative: mutate
	badBLS := append([]byte(nil), bSig...)
	if len(badBLS) > 0 {
		badBLS[0] ^= 0xAA
	}
	vectors = append(vectors, TestVector{Alg: "BLS12-381", Message: bMsg, Public: pubBLS, Signature: badBLS, Valid: false})

	return vectors, nil
}
