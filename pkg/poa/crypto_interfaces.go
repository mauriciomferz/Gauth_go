// Package poa provides crypto interface abstractions for module independence.
// These interfaces decouple pkg/poa from the main AgentAuth pkg/crypto package,
// enabling standalone module extraction.
package poa

// KeyManager provides signing and verification capabilities.
// This interface abstracts the crypto.Manager functionality needed by PoA.
type KeyManager interface {
	// Sign signs a message using the active key.
	Sign(msg []byte) ([]byte, error)
	// Verify verifies a signature against a message.
	Verify(msg, sig []byte) bool
	// KeyID returns the identifier of the active key.
	KeyID() string
}

// KeyProvider provides access to key material and verification.
// This mirrors crypto.KeyProvider for PoA operations.
type KeyProvider interface {
	// ActiveSigner returns a signer for the currently active key.
	ActiveSigner() (Signer, error)
	// PublicKey retrieves the public key bytes for a given key ID.
	PublicKey(keyID string) ([]byte, string, error)
	// VerifyWith verifies a signature using a specific key.
	VerifyWith(msg, sig []byte, keyID string) error
}

// Signer provides signing operations.
type Signer interface {
	// KeyID returns the key identifier.
	KeyID() string
	// Algorithm returns the algorithm name.
	Algorithm() string
	// Sign signs a message.
	Sign(msg []byte) ([]byte, error)
}
