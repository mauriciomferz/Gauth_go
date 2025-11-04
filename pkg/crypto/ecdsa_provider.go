package crypto

// In-memory ECDSA (P-256) provider and signer implementation to extend algorithm agility.
// Follows same patterns as the Ed25519 provider: short key id derivation and base64 DER encoding
// of signatures performed by higher layers (rfc0111 service). The Sign method returns raw DER bytes.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
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

func (s *ecdsaSigner) KeyID() string     { return s.keyID }
func (s *ecdsaSigner) Algorithm() string { return s.algo }
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
	uncompressed := elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
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
	return elliptic.Marshal(elliptic.P256(), pk.X, pk.Y), AlgoECDSAP256, nil
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
func init() {
	RegisterAlgorithm(Algorithm{Name: AlgoECDSAP256, Verify: func(canonical []byte, sigBase64 string, keyID string, kp KeyProvider) error {
		if kp == nil {
			return errors.New("ecdsa-p256: missing key provider")
		}
		sigBytes, err := base64.StdEncoding.DecodeString(sigBase64)
		if err != nil {
			return err
		}
		// Delegates to provider's VerifyWith (which performs hash + DER decode + low-S enforcement)
		return kp.VerifyWith(canonical, sigBytes, keyID)
	}})
}

// --- ECDSA helpers (DER encoding & S normalization) ---
// encodeDERSignature encodes r and s as a minimal DER sequence.
func encodeDERSignature(r, s *big.Int) []byte {
	rb := r.Bytes()
	sb := s.Bytes()
	if len(rb) > 0 && rb[0]&0x80 != 0 {
		rb = append([]byte{0x00}, rb...)
	}
	if len(sb) > 0 && sb[0]&0x80 != 0 {
		sb = append([]byte{0x00}, sb...)
	}
	total := 2 + len(rb) + 2 + len(sb)
	out := make([]byte, 0, 2+total)
	out = append(out, 0x30, byte(total))
	out = append(out, 0x02, byte(len(rb)))
	out = append(out, rb...)
	out = append(out, 0x02, byte(len(sb)))
	out = append(out, sb...)
	return out
}

// decodeDERSignature parses a DER encoded ECDSA signature.
func decodeDERSignature(der []byte) (*big.Int, *big.Int, error) {
	if len(der) < 8 || der[0] != 0x30 {
		return nil, nil, errors.New("invalid_der_prefix")
	}
	seqLen := int(der[1])
	if seqLen+2 != len(der) {
		return nil, nil, errors.New("invalid_der_length")
	}
	if der[2] != 0x02 {
		return nil, nil, errors.New("missing_r_integer")
	}
	rLen := int(der[3])
	if 4+rLen+2 >= len(der) {
		return nil, nil, errors.New("r_length_out_of_bounds")
	}
	rStart := 4
	rEnd := rStart + rLen
	if der[rEnd] != 0x02 {
		return nil, nil, errors.New("missing_s_integer")
	}
	sLen := int(der[rEnd+1])
	sStart := rEnd + 2
	sEnd := sStart + sLen
	if sEnd != len(der) {
		return nil, nil, errors.New("s_length_out_of_bounds")
	}
	r := new(big.Int).SetBytes(der[rStart:rEnd])
	s := new(big.Int).SetBytes(der[sStart:sEnd])
	return r, s, nil
}

func normalizeLowS(s, n *big.Int) *big.Int {
	half := new(big.Int).Rsh(n, 1)
	if s.Cmp(half) == 1 {
		return new(big.Int).Sub(n, s)
	}
	return s
}

func isHighS(s, n *big.Int) bool {
	half := new(big.Int).Rsh(n, 1)
	return s.Cmp(half) == 1
}
