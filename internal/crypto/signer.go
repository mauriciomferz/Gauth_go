package crypto

// Unified signing interfaces and adapters for multiple algorithms.
// Phase 1 introduces Ed25519, ECDSA (P-256) and BLS wrappers without refactoring
// existing rotation manager; adapters can operate in verify-only mode if private
// key material is absent.

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"
)

// Algorithm name constants
const (
	AlgoEd25519 = "Ed25519"
)

// Signer exposes a unified interface across supported algorithms.
// Sign may return error if operating in verify-only (no private key).
type Signer interface {
	Algo() string
	Public() []byte
	Sign(msg []byte) ([]byte, error)
	Verify(msg, sig []byte) bool
}

// -------------------- Ed25519 --------------------
type Ed25519Signer struct {
	priv ed25519.PrivateKey // optional
	pub  ed25519.PublicKey
}

func NewEd25519Signer(priv ed25519.PrivateKey, pub ed25519.PublicKey) *Ed25519Signer {
	if pub == nil && len(priv) == ed25519.PrivateKeySize {
		pub = priv.Public().(ed25519.PublicKey)
	}
	return &Ed25519Signer{priv: priv, pub: pub}
}

func (s *Ed25519Signer) Algo() string   { return AlgoEd25519 }
func (s *Ed25519Signer) Public() []byte { return append([]byte(nil), s.pub...) }
func (s *Ed25519Signer) Sign(msg []byte) ([]byte, error) {
	if len(s.priv) != ed25519.PrivateKeySize {
		return nil, errors.New("ed25519_signer_no_private")
	}
	sig := ed25519.Sign(s.priv, msg)
	return sig, nil
}
func (s *Ed25519Signer) Verify(msg, sig []byte) bool {
	return len(s.pub) == ed25519.PublicKeySize && ed25519.Verify(s.pub, msg, sig)
}

// -------------------- RotatingEd25519Signer (RB6 dynamic agility) --------------------
// Provides a Signer backed by the key Manager enabling transparent rotation without
// changing downstream signing call sites. Exposes KeyID via optional interface assertion.
type RotatingEd25519Signer struct {
	M *Manager
}

func (s *RotatingEd25519Signer) Algo() string { return AlgoEd25519 }
func (s *RotatingEd25519Signer) Public() []byte {
	if s == nil || s.M == nil || s.M.Active() == nil {
		return nil
	}
	return append([]byte(nil), s.M.Active().Public...)
}
func (s *RotatingEd25519Signer) Sign(msg []byte) ([]byte, error) {
	if s == nil || s.M == nil || s.M.Active() == nil || len(s.M.Active().Private) != ed25519.PrivateKeySize {
		return nil, errors.New("ed25519_rotating_signer_no_private")
	}
	return ed25519.Sign(s.M.Active().Private, msg), nil
}
func (s *RotatingEd25519Signer) Verify(msg, sig []byte) bool {
	if s == nil || s.M == nil || s.M.Active() == nil {
		return false
	}
	k := s.M.Active()
	return ed25519.Verify(k.Public, msg, sig)
}

// KeyID returns the active key identifier; not part of Signer interface to avoid widening existing contract.
func (s *RotatingEd25519Signer) KeyID() string {
	if s == nil || s.M == nil || s.M.Active() == nil {
		return ""
	}
	return s.M.Active().ID
}

// GlobalRotatingSigner returns a Signer backed by GlobalEdDSARegistry if available.
func GlobalRotatingSigner() Signer {
	if GlobalEdDSARegistry == nil {
		return nil
	}
	return &RotatingEd25519Signer{M: GlobalEdDSARegistry}
}

// -------------------- ECDSA (P-256 only Phase 1) --------------------
type ECDSASigner struct {
	priv *ecdsa.PrivateKey // optional
	pub  *ecdsa.PublicKey
}

func NewP256Signer(priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) *ECDSASigner {
	if pub == nil && priv != nil {
		pub = &priv.PublicKey
	}
	return &ECDSASigner{priv: priv, pub: pub}
}

func (s *ECDSASigner) Algo() string { return "ECDSA-P256" }
func (s *ECDSASigner) Public() []byte {
	if s.pub == nil {
		return nil
	}
	// Uncompressed form: 0x04 || X || Y (32 bytes each for P-256)
	byteLen := (s.pub.Curve.Params().BitSize + 7) / 8
	ret := make([]byte, 1+2*byteLen)
	ret[0] = 0x04 // uncompressed point
	s.pub.X.FillBytes(ret[1 : 1+byteLen])
	s.pub.Y.FillBytes(ret[1+byteLen:])
	return ret
}
func (s *ECDSASigner) Sign(msg []byte) ([]byte, error) {
	if s.priv == nil {
		return nil, errors.New("ecdsa_signer_no_private")
	}
	h := sha256.Sum256(msg)
	r, vS, err := ecdsa.Sign(rand.Reader, s.priv, h[:])
	if err != nil {
		return nil, err
	}
	sNorm := normalizeLowS(vS, s.priv.Params().N)
	return encodeDERSignature(r, sNorm), nil
}
func (s *ECDSASigner) Verify(msg, sig []byte) bool {
	if s.pub == nil {
		return false
	}
	r, vS, err := decodeDERSignature(sig)
	if err != nil {
		return false
	}
	// Reject high-S (malleable) signatures proactively
	if isHighS(vS, s.pub.Params().N) {
		return false
	}
	h := sha256.Sum256(msg)
	return ecdsa.Verify(s.pub, h[:], r, vS)
}

// -------------------- BLS --------------------
// Uses existing bls.go helpers. Private key optional (verify-only mode permitted).
type BLSSigner struct {
	priv *BLSKey // if nil, verification only (public must be set)
	pub  *BLSKey // public-only wrapper when priv nil
}

func NewBLSSigner(priv *BLSKey) *BLSSigner {
	if priv == nil {
		return &BLSSigner{}
	}
	return &BLSSigner{priv: priv}
}

func (s *BLSSigner) Algo() string { return "BLS12-381" }
func (s *BLSSigner) Public() []byte {
	if s.priv != nil {
		return s.priv.Public.Serialize()
	}
	if s.pub != nil {
		return s.pub.Public.Serialize()
	}
	return nil
}
func (s *BLSSigner) Sign(msg []byte) ([]byte, error) {
	if s.priv == nil {
		return nil, errors.New("bls_signer_no_private")
	}
	return BLSSign(s.priv, msg)
}
func (s *BLSSigner) Verify(msg, sig []byte) bool {
	var k *BLSKey
	if s.priv != nil {
		k = s.priv
	} else {
		k = s.pub
	}
	if k == nil {
		return false
	}
	return BLSVerify(k, msg, sig)
}

// -------------------- ECDSA helpers --------------------
// DER encoding: SEQUENCE { r INTEGER, s INTEGER }
// Minimal encoder (no negative values expected; ensures r,s positive).
func encodeDERSignature(r, s *big.Int) []byte {
	rb := r.Bytes()
	sb := s.Bytes()
	// Add leading zero if high bit set (avoid negative interpretation)
	if len(rb) > 0 && rb[0]&0x80 != 0 {
		rb = append([]byte{0x00}, rb...)
	}
	if len(sb) > 0 && sb[0]&0x80 != 0 {
		sb = append([]byte{0x00}, sb...)
	}
	// 0x30 len 0x02 rlen r 0x02 slen s
	// Compute total length of payload after initial sequence tag & len
	total := 2 + len(rb) + 2 + len(sb)
	out := make([]byte, 0, 2+total)
	out = append(out, 0x30, byte(total))
	out = append(out, 0x02, byte(len(rb)))
	out = append(out, rb...)
	out = append(out, 0x02, byte(len(sb)))
	out = append(out, sb...)
	return out
}

func decodeDERSignature(der []byte) (*big.Int, *big.Int, error) {
	if len(der) < 8 || der[0] != 0x30 {
		return nil, nil, errors.New("invalid_der_prefix")
	}
	// Basic length check
	seqLen := int(der[1])
	if seqLen+2 != len(der) {
		return nil, nil, errors.New("invalid_der_length")
	}
	// Expect INTEGER r
	if der[2] != 0x02 {
		return nil, nil, errors.New("missing_r_integer")
	}
	rLen := int(der[3])
	if 4+rLen+2 >= len(der) {
		return nil, nil, errors.New("r_length_out_of_bounds")
	}
	rStart := 4
	rEnd := rStart + rLen
	// INTEGER s
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
	// if s > n/2 then s = n - s
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
