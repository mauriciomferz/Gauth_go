package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

// Package crypto provides signing & verification abstractions for Milestone 2B.
// Initial implementation targets Ed25519; future milestones may add threshold / multi-sig.

// Signer produces digital signatures over message digests or direct messages.
type Signer interface {
	KeyID() string
	Algorithm() string
	Sign(msg []byte) ([]byte, error)
}

// Verifier checks signatures and returns error if invalid.
type Verifier interface {
	Algorithm() string
	Verify(msg, sig []byte, keyID string) error
}

// KeyProvider returns the active signer and allows lookup of public keys by ID.
type KeyProvider interface {
	ActiveSigner() (Signer, error)
	PublicKey(keyID string) (keyBytes []byte, algo string, err error)
	VerifyWith(msg, sig []byte, keyID string) error
}

// KMS abstracts a key management system capable of returning an active signer,
// looking up public keys, listing metadata, and performing rotation. This allows
// future Vault/HSM/backing service integration without changing higher layers.
// All methods are designed to degrade gracefully so demo/tests can use the
// in-memory implementation.
type KMS interface {
	ActiveSigner() (Signer, error)
	PublicKey(keyID string) ([]byte, string, error)
	Rotate() (string, error)              // optional; implementations may return an error if unsupported
	ListKeys() ([]KeyMetadata, error)     // metadata enumeration
}

// KeyMetadata captures descriptive information about a managed key.
type KeyMetadata struct {
	ID        string `json:"id"`
	Algorithm string `json:"algorithm"`
	CreatedAt int64  `json:"created_at_unix"`
	Active    bool   `json:"active"`
}

const (
	AlgoEd25519 = "ed25519"
)

// ErrUnknownKey indicates a key id was not found.
var ErrUnknownKey = errors.New("crypto: unknown key id")

// ed25519Signer implements Signer using an ed25519 private key.
type ed25519Signer struct {
	keyID string
	priv  ed25519.PrivateKey
	pub   ed25519.PublicKey
	algo  string
}

func (s *ed25519Signer) KeyID() string     { return s.keyID }
func (s *ed25519Signer) Algorithm() string { return s.algo }
func (s *ed25519Signer) Sign(msg []byte) ([]byte, error) {
	// ed25519 signs the raw message; upstream caller should supply canonical digest if desired.
	sig := ed25519.Sign(s.priv, msg)
	return sig, nil
}

// (Verifier implementation is provided indirectly via InMemoryKeyProvider.VerifyWith for now.)

// InMemoryKeyProvider is a simple key provider holding one active and a map of public keys.
type InMemoryKeyProvider struct {
	mu      sync.RWMutex
	active  *ed25519Signer
	publics map[string]ed25519.PublicKey
}

// NewInMemoryEd25519Provider generates a new ed25519 key pair and returns a provider.
func NewInMemoryEd25519Provider() (*InMemoryKeyProvider, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519: %w", err)
	}
	keyID := deriveKeyID(pub)
	signer := &ed25519Signer{keyID: keyID, priv: priv, pub: pub, algo: AlgoEd25519}
	return &InMemoryKeyProvider{active: signer, publics: map[string]ed25519.PublicKey{keyID: pub}}, nil
}

// deriveKeyID derives a short stable identifier from the public key (first 12 hex of SHA256(pub)).
func deriveKeyID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:6]) // 12 hex chars
}

func (p *InMemoryKeyProvider) ActiveSigner() (Signer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.active == nil {
		return nil, errors.New("no active signer")
	}
	return p.active, nil
}

func (p *InMemoryKeyProvider) PublicKey(keyID string) ([]byte, string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pk, ok := p.publics[keyID]
	if !ok {
		return nil, "", ErrUnknownKey
	}
	return pk, AlgoEd25519, nil
}

// Rotate generates a new key pair and sets it active while retaining previous public key.
func (p *InMemoryKeyProvider) Rotate() (newKeyID string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("rotate generate: %w", err)
	}
	keyID := deriveKeyID(pub)
	signer := &ed25519Signer{keyID: keyID, priv: priv, pub: pub, algo: AlgoEd25519}
	p.mu.Lock()
	if p.publics == nil {
		p.publics = make(map[string]ed25519.PublicKey)
	}
	p.publics[keyID] = pub
	p.active = signer
	p.mu.Unlock()
	return keyID, nil
}

// ListKeys returns metadata for all known public keys (active marked Active=true).
func (p *InMemoryKeyProvider) ListKeys() ([]KeyMetadata, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []KeyMetadata
	for id := range p.publics {
		km := KeyMetadata{ID: id, Algorithm: AlgoEd25519, CreatedAt: 0, Active: p.active != nil && p.active.keyID == id}
		out = append(out, km)
	}
	return out, nil
}

// InMemoryKeyProvider implements KMS implicitly (ActiveSigner/PublicKey/Rotate/ListKeys).

// VerifyWith verifies a signature using the stored public key for keyID.
func (p *InMemoryKeyProvider) VerifyWith(msg, sig []byte, keyID string) error {
	pkBytes, algo, err := p.PublicKey(keyID)
	if err != nil {
		return err
	}
	if algo != AlgoEd25519 {
		return fmt.Errorf("unsupported algo %s", algo)
	}
	if len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature length")
	}
	if !ed25519.Verify(ed25519.PublicKey(pkBytes), msg, sig) {
		return errors.New("invalid signature")
	}
	return nil
}

// VerifierFunc used in algorithm registry
type VerifierFunc func(canonical []byte, sigBase64 string, keyID string, kp KeyProvider) error

// Algorithm describes a signature verification handler
type Algorithm struct {
	Name   string
	Verify VerifierFunc
}

var algoRegistry = map[string]Algorithm{}

// RegisterAlgorithm registers a signature algorithm
func RegisterAlgorithm(a Algorithm) {
	if a.Name != "" && a.Verify != nil {
		algoRegistry[a.Name] = a
	}
}

// GetAlgorithm retrieves registered algorithm
func GetAlgorithm(name string) *Algorithm {
	if a, ok := algoRegistry[name]; ok {
		return &a
	}
	return nil
}

// VerifyAlgorithm dispatches verification via registry
func VerifyAlgorithm(algo string, canonical []byte, sigBase64, keyID string, kp KeyProvider) error {
	a := GetAlgorithm(algo)
	if a == nil {
		return fmt.Errorf("unknown signature algorithm: %s", algo)
	}
	return a.Verify(canonical, sigBase64, keyID, kp)
}

func init() {
	// Register ed25519
	RegisterAlgorithm(Algorithm{Name: AlgoEd25519, Verify: func(canonical []byte, sigBase64 string, keyID string, kp KeyProvider) error {
		if kp == nil {
			return errors.New("ed25519: missing key provider")
		}
		sigBytes, err := base64.StdEncoding.DecodeString(sigBase64)
		if err != nil {
			return err
		}
		return kp.VerifyWith(canonical, sigBytes, keyID)
	}})
}
