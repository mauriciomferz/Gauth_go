package signalgo

// Package signalgo provides a lightweight signature algorithm registry to enable
// future cryptographic agility (RFC0111:6). The initial implementation registers
// Ed25519 as the default algorithm. A stub ECDSA P256 implementation is provided
// behind an environment flag for forward compatibility but is not used by core
// signing flows yet. Canonical digests remain unchanged; this layer only abstracts
// signing and verification primitives.

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"sync"
)

// PublicKey represents a generic public key byte slice.
type PublicKey []byte

// PrivateKey represents a generic private key byte slice.
type PrivateKey []byte

// SignatureAlgorithm defines pluggable signing primitives.
type SignatureAlgorithm interface {
    Name() string
    KeyGen() (PublicKey, PrivateKey, error)
    Sign(priv PrivateKey, msg []byte) ([]byte, error)
    Verify(pub PublicKey, msg []byte, sig []byte) bool
}

// registry maintains a map of supported algorithms.
type registry struct {
    mu    sync.RWMutex
    algos map[string]SignatureAlgorithm
}

// AlgorithmRegistry is the global default registry instance.
var AlgorithmRegistry = &registry{algos: make(map[string]SignatureAlgorithm)}

// Register adds a new algorithm if not already present.
func Register(algo SignatureAlgorithm) error {
    AlgorithmRegistry.mu.Lock()
    defer AlgorithmRegistry.mu.Unlock()
    if _, exists := AlgorithmRegistry.algos[algo.Name()]; exists {
        return errors.New("signalgo: algorithm already registered: " + algo.Name())
    }
    AlgorithmRegistry.algos[algo.Name()] = algo
    return nil
}

// Get retrieves an algorithm by name.
func Get(name string) (SignatureAlgorithm, bool) {
    AlgorithmRegistry.mu.RLock()
    defer AlgorithmRegistry.mu.RUnlock()
    a, ok := AlgorithmRegistry.algos[name]
    return a, ok
}

// List returns the names of registered algorithms.
func List() []string {
    AlgorithmRegistry.mu.RLock()
    defer AlgorithmRegistry.mu.RUnlock()
    out := make([]string, 0, len(AlgorithmRegistry.algos))
    for k := range AlgorithmRegistry.algos {
        out = append(out, k)
    }
    return out
}

// ed25519Algorithm implements SignatureAlgorithm for Ed25519.
type ed25519Algorithm struct{}

func (e ed25519Algorithm) Name() string { return "Ed25519" }

func (e ed25519Algorithm) KeyGen() (PublicKey, PrivateKey, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, nil, err
    }
    return PublicKey(pub), PrivateKey(priv), nil
}

func (e ed25519Algorithm) Sign(priv PrivateKey, msg []byte) ([]byte, error) {
    if len(priv) != ed25519.PrivateKeySize {
        return nil, errors.New("signalgo: invalid ed25519 private key length")
    }
    return ed25519.Sign(ed25519.PrivateKey(priv), msg), nil
}

func (e ed25519Algorithm) Verify(pub PublicKey, msg []byte, sig []byte) bool {
    if len(pub) != ed25519.PublicKeySize {
        return false
    }
    return ed25519.Verify(ed25519.PublicKey(pub), msg, sig)
}

// init registers default algorithms.
func init() {
    _ = Register(ed25519Algorithm{})
}
