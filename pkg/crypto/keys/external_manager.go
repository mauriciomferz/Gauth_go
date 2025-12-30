package keys

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"
	"sync"
)

// KMSClient represents a client to an external Key Management Service.
type KMSClient interface {
	// SignDigest signs the given digest. The mechanism implies the private key is remote.
	SignDigest(ctx context.Context, digest []byte) ([]byte, error)
	// PublicKey returns the public key associated with the remote private key.
	PublicKey(ctx context.Context) (crypto.PublicKey, error)
	// KeyID returns the identifier of the key.
	KeyID() string
	// LookupPublicKey returns the public key for a specific ID (active or previous).
	LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error)
}

// ExternalKeyManager implements KeyManager using an external KMS client.
type ExternalKeyManager struct {
	client KMSClient
}

// NewExternalKeyManager creates a new key manager backed by a KMS client.
func NewExternalKeyManager(client KMSClient) *ExternalKeyManager {
	return &ExternalKeyManager{client: client}
}

func (m *ExternalKeyManager) Sign(ctx context.Context, data []byte) ([]byte, error) {
	// For high-level Sign, we hash locally then ask KMS to sign digest.
	// Assuming RS256 by default for now (or whatever KMS is configured for).
	// AgentAuth historically uses RSA.
	h := sha256.Sum256(data)
	return m.client.SignDigest(ctx, h[:])
}

func (m *ExternalKeyManager) GetPublicKey(ctx context.Context) (crypto.PublicKey, error) {
	return m.client.PublicKey(ctx)
}

func (m *ExternalKeyManager) GetKeyID(ctx context.Context) (string, error) {
	return m.client.KeyID(), nil
}

func (m *ExternalKeyManager) LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	return m.client.LookupPublicKey(ctx, kid)
}

func (m *ExternalKeyManager) CryptoSigner(ctx context.Context) (crypto.Signer, error) {
	pub, err := m.client.PublicKey(ctx)
	if err != nil {
		return nil, err
	}
	return &externalSigner{
		client: m.client,
		pub:    pub,
		ctx:    ctx,
	}, nil
}

// externalSigner adapts KMSClient to crypto.Signer interface.
type externalSigner struct {
	client KMSClient
	pub    crypto.PublicKey
	ctx    context.Context // Context captured from CryptoSigner call
}

func (s *externalSigner) Public() crypto.PublicKey {
	return s.pub
}

func (s *externalSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	// Ignore rand, ignore opts (assuming KMS has fixed algo or opts are compatible).
	// In real world, we might check opts.HashFunc() match.
	return s.client.SignDigest(s.ctx, digest)
}

// --- Simulated KMS ---

// SimulatedKMS is an in-memory simulation of a KMS.
type SimulatedKMS struct {
	privKey *rsa.PrivateKey
	kid     string
	mu      sync.Mutex
}

func NewSimulatedKMS(kid string) (*SimulatedKMS, error) {
	// Generate a key that stays inside this struct
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &SimulatedKMS{
		privKey: pk,
		kid:     kid,
	}, nil
}

func (kms *SimulatedKMS) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()
	// Simulate network latency if desired, but not needed for logic test.
	return rsa.SignPKCS1v15(rand.Reader, kms.privKey, crypto.SHA256, digest)
}

func (kms *SimulatedKMS) PublicKey(ctx context.Context) (crypto.PublicKey, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()
	return &kms.privKey.PublicKey, nil
}

func (kms *SimulatedKMS) KeyID() string {
	return kms.kid
}

func (kms *SimulatedKMS) LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()
	if kid == kms.kid {
		return &kms.privKey.PublicKey, nil
	}
	return nil, fmt.Errorf("key not found: %s", kid)
}
