package crypto

// In-memory ECDSA (P-256) provider and signer implementation to extend algorithm agility.
// Follows same patterns as the Ed25519 provider: short key id derivation and base64 DER encoding
// of signatures performed by higher layers"AAP001 service). The Sign method returns raw DER bytes.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	AlgoECDSAP256 = "ecdsa-p256"
)

// ecdsaSigner implements Signer for P-256 ECDSA with DER encoded signatures.
type ecdsaSigner struct {
	keyID string
	priv  *ecdsa.PrivateKey
	pub   *ecdsa.PublicKey
	algo  string
}

func (s *ecdsaSigner) KeyID() string { return s.keyID }

func (s *ecdsaSigner) Algorithm() string { return s.algo }
func (s *ecdsaSigner) Public() []byte {
	// Uncompressed form: 0x04 || X || Y (32 bytes each for P-256)
	byteLen := (s.pub.Curve.Params().BitSize + 7) / 8
	ret := make([]byte, 1+2*byteLen)
	ret[0] = 0x04
	s.pub.X.FillBytes(ret[1 : 1+byteLen])
	s.pub.Y.FillBytes(ret[1+byteLen:])
	return ret
}
func (s *ecdsaSigner) Verify(msg, sig []byte) bool {
	// Verify implementation for Signer interface
	// Note: Signer interface expects Verify(msg, sig) bool
	// But ecdsaSigner logic was in VerifyWith.
	// We should adapt it.
	// For now, let's just implement Algo() and see if Verify is needed.
	// Signer interface has Verify(msg, sig []byte) bool.
	// ecdsaSigner didn't implement Verify before?
	// Wait, ecdsa_provider.go didn't implement Verify for ecdsaSigner?
	// Let's check signer.go Signer interface again.
	// It has Verify(msg, sig []byte) bool.
	// So ecdsaSigner MUST implement it.
	// I'll add it.
	h := sha256.Sum256(msg)
	r, sv, err := decodeDERSignature(sig)
	if err != nil {
		return false
	}
	if isHighS(sv, s.pub.Params().N) {
		return false
	}
	return ecdsa.Verify(s.pub, h[:], r, sv)
}
func (s *ecdsaSigner) Sign(msg []byte) ([]byte, error) {
	if s.priv == nil {
		return nil, errors.New("ecdsa: no private key")
	}
	// Hash then sign (ECDSA requires hashing; we standardize on SHA-256 for canonical bytes)
	h := sha256.Sum256(msg)
	r, sv, err := ecdsa.Sign(rand.Reader, s.priv, h[:])
	if err != nil {
		return nil, err
	}
	// Low-S normalization (anti-malleability) – mirror enforcement in internal crypto
	sNorm := normalizeLowS(sv, s.priv.Params().N)
	return encodeDERSignature(r, sNorm), nil
}

// InMemoryECDSAProvider manages a single active key and lookup map of public keys.
type InMemoryECDSAProvider struct {
	active  *ecdsaSigner
	publics map[string]*ecdsa.PublicKey
}

// NewInMemoryECDSAProvider generates a new P-256 key pair.
func NewInMemoryECDSAProvider() (*InMemoryECDSAProvider, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ecdsa p256: %w", err)
	}
	keyID := deriveECDSAKeyID(&priv.PublicKey)
	signer := &ecdsaSigner{keyID: keyID, priv: priv, pub: &priv.PublicKey, algo: AlgoECDSAP256}
	return &InMemoryECDSAProvider{active: signer, publics: map[string]*ecdsa.PublicKey{keyID: &priv.PublicKey}}, nil
}

// deriveECDSAKeyID derives short identifier from uncompressed public key (first 12 hex of SHA256(pub)).
func deriveECDSAKeyID(pub *ecdsa.PublicKey) string {
	if pub == nil {
		return ""
	}
	// Uncompressed form: 0x04 || X || Y (32 bytes each for P-256)
	byteLen := (pub.Curve.Params().BitSize + 7) / 8
	uncompressed := make([]byte, 1+2*byteLen)
	uncompressed[0] = 0x04
	pub.X.FillBytes(uncompressed[1 : 1+byteLen])
	pub.Y.FillBytes(uncompressed[1+byteLen:])
	h := sha256.Sum256(uncompressed)
	return hex.EncodeToString(h[:6])
}

func (p *InMemoryECDSAProvider) ActiveSigner() (Signer, error) {
	if p.active == nil {
		return nil, errors.New("no active signer")
	}
	return p.active, nil
}

// PublicKey returns uncompressed form of the P-256 public key.
func (p *InMemoryECDSAProvider) PublicKey(keyID string) ([]byte, string, error) {
	pk, ok := p.publics[keyID]
	if !ok {
		return nil, "", ErrUnknownKey
	}
	// Uncompressed form: 0x04 || X || Y (32 bytes each for P-256)
	byteLen := (pk.Curve.Params().BitSize + 7) / 8
	ret := make([]byte, 1+2*byteLen)
	ret[0] = 0x04
	pk.X.FillBytes(ret[1 : 1+byteLen])
	pk.Y.FillBytes(ret[1+byteLen:])
	return ret, AlgoECDSAP256, nil
}

// VerifyWith verifies base64 DER signature against canonical bytes (after SHA-256 hashing).
func (p *InMemoryECDSAProvider) VerifyWith(msg, sig []byte, keyID string) error {
	pk, ok := p.publics[keyID]
	if !ok {
		return ErrUnknownKey
	}
	// msg is canonical bytes; hash then verify
	h := sha256.Sum256(msg)
	// DER decode
	r, sv, derr := decodeDERSignature(sig)
	if derr != nil {
		return derr
	}
	// Reject high-S (malleable) signatures
	if isHighS(sv, pk.Params().N) {
		return errors.New("ecdsa: high-s signature rejected")
	}
	if !ecdsa.Verify(pk, h[:], r, sv) {
		return errors.New("ecdsa: invalid signature")
	}
	return nil
}

// Rotate generates and sets a new active key pair.
func (p *InMemoryECDSAProvider) Rotate() (string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("rotate generate ecdsa: %w", err)
	}
	keyID := deriveECDSAKeyID(&priv.PublicKey)
	p.publics[keyID] = &priv.PublicKey
	p.active = &ecdsaSigner{keyID: keyID, priv: priv, pub: &priv.PublicKey, algo: AlgoECDSAP256}
	return keyID, nil
}

// ListKeys enumerates public keys.
func (p *InMemoryECDSAProvider) ListKeys() ([]KeyMetadata, error) {
	var out []KeyMetadata
	for id := range p.publics {
		out = append(out, KeyMetadata{ID: id, Algorithm: AlgoECDSAP256, Active: p.active != nil && p.active.keyID == id})
	}
	return out, nil
}

// Register ECDSA algorithm with registry on package init.
//
//nolint:gochecknoinits
func init() {
	RegisterAlgorithm(Algorithm{
		Name: AlgoECDSAP256,
		Verify: func(canonical []byte, sigBase64 string, keyID string, kp KeyProvider) error {
			if kp == nil {
				return errors.New("ecdsa-p256: missing key provider")
			}
			sigBytes, err := base64.StdEncoding.DecodeString(sigBase64)
			if err != nil {
				return err
			}
			// Delegates to provider's VerifyWith (which performs hash + DER decode + low-S enforcement)
			return kp.VerifyWith(canonical, sigBytes, keyID)
		},
	})
}
