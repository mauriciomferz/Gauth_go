package crypto

// BLS (BLS12-381) provider and registry integration for algorithm agility.
// Relies on internal/crypto BLS primitives (GenerateBLSKey, BLSSign, BLSVerify).

import (
	"encoding/base64"
	"errors"

	blsbin "github.com/herumi/bls-eth-go-binary/bls"
)

const AlgoBLS12381 = "bls12-381"

// blsSigner implements Signer for BLS12-381.
type blsSigner struct {
    keyID string
    priv  blsbin.SecretKey
    pub   blsbin.PublicKey
    havePriv bool
}

func (s *blsSigner) KeyID() string     { return s.keyID }
func (s *blsSigner) Algorithm() string { return AlgoBLS12381 }
func (s *blsSigner) Sign(msg []byte) ([]byte, error) {
    if !s.havePriv { return nil, errors.New("bls signer: no private key") }
    sig := s.priv.SignByte(msg)
    return sig.Serialize(), nil
}

// InMemoryBLSProvider stores one active key and public key map.
type InMemoryBLSProvider struct {
    active  *blsSigner
    publics map[string]blsbin.PublicKey
}

// NewInMemoryBLSProvider creates a new BLS key pair and provider.
func NewInMemoryBLSProvider() (*InMemoryBLSProvider, error) {
    if err := blsbin.Init(blsbin.BLS12_381); err != nil { return nil, err }
    var sk blsbin.SecretKey
    sk.SetByCSPRNG()
    pk := sk.GetPublicKey()
    kid := deriveKeyID(pk.Serialize()) // reuse first 12 hex of SHA256(pub) derivation
    signer := &blsSigner{keyID: kid, priv: sk, pub: *pk, havePriv: true}
    return &InMemoryBLSProvider{active: signer, publics: map[string]blsbin.PublicKey{kid: *pk}}, nil
}

func (p *InMemoryBLSProvider) ActiveSigner() (Signer, error) {
    if p.active == nil { return nil, errors.New("no active bls signer") }
    return p.active, nil
}

func (p *InMemoryBLSProvider) PublicKey(keyID string) ([]byte, string, error) {
    pk, ok := p.publics[keyID]
    if !ok { return nil, "", ErrUnknownKey }
    return pk.Serialize(), AlgoBLS12381, nil
}

func (p *InMemoryBLSProvider) VerifyWith(msg, sig []byte, keyID string) error {
    pk, ok := p.publics[keyID]
    if !ok { return ErrUnknownKey }
    var sigObj blsbin.Sign
    if err := sigObj.Deserialize(sig); err != nil { return err }
    if !sigObj.VerifyByte(&pk, msg) { return errors.New("bls: invalid signature") }
    return nil
}

// Rotate creates a new key pair and sets it active.
func (p *InMemoryBLSProvider) Rotate() (string, error) {
    if err := blsbin.Init(blsbin.BLS12_381); err != nil { return "", err }
    var sk blsbin.SecretKey
    sk.SetByCSPRNG()
    pk := sk.GetPublicKey()
    kid := deriveKeyID(pk.Serialize())
    p.publics[kid] = *pk
    p.active = &blsSigner{keyID: kid, priv: sk, pub: *pk, havePriv: true}
    return kid, nil
}

func (p *InMemoryBLSProvider) ListKeys() ([]KeyMetadata, error) {
    var out []KeyMetadata
    for id := range p.publics {
        out = append(out, KeyMetadata{ID: id, Algorithm: AlgoBLS12381, Active: p.active != nil && p.active.keyID == id})
    }
    return out, nil
}

// Register algorithm on init (single signature only; aggregated variant TBD).
func init() {
    RegisterAlgorithm(Algorithm{Name: AlgoBLS12381, Verify: func(canonical []byte, sigBase64 string, keyID string, kp KeyProvider) error {
        if kp == nil { return errors.New("bls12-381: missing key provider") }
        sigBytes, err := base64.StdEncoding.DecodeString(sigBase64)
        if err != nil { return err }
        return kp.VerifyWith(canonical, sigBytes, keyID)
    }})
}
