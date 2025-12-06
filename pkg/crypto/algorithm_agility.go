// Package crypto provides cryptographic signature algorithm abstraction for GAuth.
//
// This package implements algorithm agility, allowing GAuth to support multiple
// signature algorithms (Ed25519, RSA-PSS, ECDSA P-256) for both signing and
// verification operations. This enables gradual migration between algorithms
// and interoperability with systems using different cryptographic standards.
package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
)

// Algorithm identifier constants following IANA JOSE registry conventions
const (
	AlgorithmEd25519   = "EdDSA" // RFC 8032 - Ed25519 signature algorithm
	AlgorithmRSAPSS    = "PS256" // RFC 8017 - RSA-PSS with SHA-256
	AlgorithmECDSAP256 = "ES256" // RFC 7518 - ECDSA with P-256 and SHA-256
)

// AlgorithmSignerVerifier provides core signing and verification operations for algorithm providers.
// Consumers that only need to sign or verify should depend on this interface.
type AlgorithmSignerVerifier interface {
	// Sign generates a signature over the provided message using the private key
	Sign(privateKey interface{}, message []byte) ([]byte, error)

	// Verify checks the signature against the message using the public key
	Verify(publicKey interface{}, message, signature []byte) error

	// AlgorithmID returns the IANA JOSE algorithm identifier (e.g., "EdDSA", "PS256", "ES256")
	AlgorithmID() string
}

// KeyFactory generates new key pairs for a specific algorithm.
type KeyFactory interface {
	// GenerateKey creates a new key pair for this algorithm
	GenerateKey() (privateKey interface{}, publicKey interface{}, err error)

	// KeyType returns the expected Go type for private keys (e.g., "ed25519.PrivateKey")
	KeyType() string
}

// KeySerializer handles PEM encoding and decoding of keys.
// JWKS endpoints and key export features should depend on this interface.
type KeySerializer interface {
	// MarshalPrivateKey encodes the private key to PEM format
	MarshalPrivateKey(privateKey interface{}) ([]byte, error)

	// UnmarshalPrivateKey decodes a private key from PEM format
	UnmarshalPrivateKey(pemData []byte) (interface{}, error)

	// MarshalPublicKey encodes the public key to PEM format
	MarshalPublicKey(publicKey interface{}) ([]byte, error)

	// UnmarshalPublicKey decodes a public key from PEM format
	UnmarshalPublicKey(pemData []byte) (interface{}, error)
}

// SignatureAlgorithm defines the complete interface for cryptographic signature operations.
// Implementations must provide key generation, signing, and verification for
// specific algorithm types.
//
// SignatureAlgorithm composes the segregated interfaces for backward compatibility.
// Consumers should prefer depending on the smaller interfaces when full
// functionality is not required.
type SignatureAlgorithm interface {
	AlgorithmSignerVerifier
	KeyFactory
	KeySerializer
}

// Ed25519Provider implements SignatureAlgorithm for Ed25519 (EdDSA).
// This is the default algorithm for GAuth, providing fast signatures with
// small key sizes (32 bytes) and signatures (64 bytes).
type Ed25519Provider struct{}

// Sign generates an Ed25519 signature over the message.
func (p *Ed25519Provider) Sign(privateKey interface{}, message []byte) ([]byte, error) {
	privKey, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected ed25519.PrivateKey, got %T", privateKey)
	}
	return ed25519.Sign(privKey, message), nil
}

// Verify checks an Ed25519 signature against the message.
func (p *Ed25519Provider) Verify(publicKey interface{}, message, signature []byte) error {
	pubKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("invalid key type: expected ed25519.PublicKey, got %T", publicKey)
	}
	if !ed25519.Verify(pubKey, message, signature) {
		return errors.New("signature verification failed")
	}
	return nil
}

// KeyType returns the Go type name for Ed25519 private keys.
func (p *Ed25519Provider) KeyType() string {
	return "ed25519.PrivateKey"
}

// AlgorithmID returns the IANA JOSE algorithm identifier for Ed25519.
func (p *Ed25519Provider) AlgorithmID() string {
	return AlgorithmEd25519
}

// GenerateKey creates a new Ed25519 key pair.
func (p *Ed25519Provider) GenerateKey() (interface{}, interface{}, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}
	return privKey, pubKey, nil
}

// MarshalPrivateKey encodes an Ed25519 private key to PEM format.
func (p *Ed25519Provider) MarshalPrivateKey(privateKey interface{}) ([]byte, error) {
	privKey, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected ed25519.PrivateKey, got %T", privateKey)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), nil
}

// UnmarshalPrivateKey decodes an Ed25519 private key from PEM format.
func (p *Ed25519Provider) UnmarshalPrivateKey(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	privKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type in PEM: expected ed25519.PrivateKey, got %T", key)
	}
	return privKey, nil
}

// MarshalPublicKey encodes an Ed25519 public key to PEM format.
func (p *Ed25519Provider) MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	pubKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected ed25519.PublicKey, got %T", publicKey)
	}
	pkix, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}), nil
}

// UnmarshalPublicKey decodes an Ed25519 public key from PEM format.
func (p *Ed25519Provider) UnmarshalPublicKey(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	pubKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type in PEM: expected ed25519.PublicKey, got %T", key)
	}
	return pubKey, nil
}

// RSAPSSProvider implements SignatureAlgorithm for RSA-PSS with SHA-256.
// This provides compatibility with systems requiring RSA signatures and
// offers stronger security guarantees than traditional RSA PKCS#1 v1.5.
type RSAPSSProvider struct {
	keySize int // RSA key size in bits (default: 2048, recommended: 3072+)
}

// NewRSAPSSProvider creates an RSA-PSS provider with the specified key size.
// Recommended key sizes: 2048 (minimum), 3072 (standard), 4096 (high security).
func NewRSAPSSProvider(keySize int) *RSAPSSProvider {
	if keySize < 2048 {
		keySize = 2048 // Enforce minimum security level
	}
	return &RSAPSSProvider{keySize: keySize}
}

// Sign generates an RSA-PSS signature over the message using SHA-256.
func (p *RSAPSSProvider) Sign(privateKey interface{}, message []byte) ([]byte, error) {
	privKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *rsa.PrivateKey, got %T", privateKey)
	}

	hash := sha256.Sum256(message)
	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}

	signature, err := rsa.SignPSS(rand.Reader, privKey, crypto.SHA256, hash[:], opts)
	if err != nil {
		return nil, fmt.Errorf("RSA-PSS signing failed: %w", err)
	}
	return signature, nil
}

// Verify checks an RSA-PSS signature against the message using SHA-256.
func (p *RSAPSSProvider) Verify(publicKey interface{}, message, signature []byte) error {
	pubKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid key type: expected *rsa.PublicKey, got %T", publicKey)
	}

	hash := sha256.Sum256(message)
	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}

	err := rsa.VerifyPSS(pubKey, crypto.SHA256, hash[:], signature, opts)
	if err != nil {
		return fmt.Errorf("RSA-PSS verification failed: %w", err)
	}
	return nil
}

// KeyType returns the Go type name for RSA private keys.
func (p *RSAPSSProvider) KeyType() string {
	return "*rsa.PrivateKey"
}

// AlgorithmID returns the IANA JOSE algorithm identifier for RSA-PSS with SHA-256.
func (p *RSAPSSProvider) AlgorithmID() string {
	return AlgorithmRSAPSS
}

// GenerateKey creates a new RSA key pair with the configured key size.
func (p *RSAPSSProvider) GenerateKey() (interface{}, interface{}, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, p.keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	return privKey, &privKey.PublicKey, nil
}

// MarshalPrivateKey encodes an RSA private key to PEM format.
func (p *RSAPSSProvider) MarshalPrivateKey(privateKey interface{}) ([]byte, error) {
	privKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *rsa.PrivateKey, got %T", privateKey)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), nil
}

// UnmarshalPrivateKey decodes an RSA private key from PEM format.
func (p *RSAPSSProvider) UnmarshalPrivateKey(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	privKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type in PEM: expected *rsa.PrivateKey, got %T", key)
	}
	return privKey, nil
}

// MarshalPublicKey encodes an RSA public key to PEM format.
func (p *RSAPSSProvider) MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	pubKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *rsa.PublicKey, got %T", publicKey)
	}
	pkix, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}), nil
}

// UnmarshalPublicKey decodes an RSA public key from PEM format.
func (p *RSAPSSProvider) UnmarshalPublicKey(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	pubKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type in PEM: expected *rsa.PublicKey, got %T", key)
	}
	return pubKey, nil
}

// ECDSAP256Provider implements SignatureAlgorithm for ECDSA with P-256 curve and SHA-256.
// This provides a balance between security and performance, with smaller key sizes
// than RSA (256-bit keys) and faster operations than RSA-PSS.
type ECDSAP256Provider struct{}

// Sign generates an ECDSA signature over the message using P-256 and SHA-256.
func (p *ECDSAP256Provider) Sign(privateKey interface{}, message []byte) ([]byte, error) {
	privKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *ecdsa.PrivateKey, got %T", privateKey)
	}

	hash := sha256.Sum256(message)
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("ECDSA signing failed: %w", err)
	}
	return signature, nil
}

// Verify checks an ECDSA signature against the message using P-256 and SHA-256.
func (p *ECDSAP256Provider) Verify(publicKey interface{}, message, signature []byte) error {
	pubKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid key type: expected *ecdsa.PublicKey, got %T", publicKey)
	}

	hash := sha256.Sum256(message)
	valid := ecdsa.VerifyASN1(pubKey, hash[:], signature)
	if !valid {
		return errors.New("ECDSA signature verification failed")
	}
	return nil
}

// KeyType returns the Go type name for ECDSA private keys.
func (p *ECDSAP256Provider) KeyType() string {
	return "*ecdsa.PrivateKey"
}

// AlgorithmID returns the IANA JOSE algorithm identifier for ECDSA P-256.
func (p *ECDSAP256Provider) AlgorithmID() string {
	return AlgorithmECDSAP256
}

// GenerateKey creates a new ECDSA P-256 key pair.
func (p *ECDSAP256Provider) GenerateKey() (interface{}, interface{}, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}
	return privKey, &privKey.PublicKey, nil
}

// MarshalPrivateKey encodes an ECDSA private key to PEM format.
func (p *ECDSAP256Provider) MarshalPrivateKey(privateKey interface{}) ([]byte, error) {
	privKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *ecdsa.PrivateKey, got %T", privateKey)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), nil
}

// UnmarshalPrivateKey decodes an ECDSA private key from PEM format.
func (p *ECDSAP256Provider) UnmarshalPrivateKey(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	privKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type in PEM: expected *ecdsa.PrivateKey, got %T", key)
	}
	return privKey, nil
}

// MarshalPublicKey encodes an ECDSA public key to PEM format.
func (p *ECDSAP256Provider) MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	pubKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *ecdsa.PublicKey, got %T", publicKey)
	}
	pkix, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}), nil
}

// UnmarshalPublicKey decodes an ECDSA public key from PEM format.
func (p *ECDSAP256Provider) UnmarshalPublicKey(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	pubKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type in PEM: expected *ecdsa.PublicKey, got %T", key)
	}
	return pubKey, nil
}

// AlgorithmRegistry manages available signature algorithms and provides
// algorithm selection based on identifier strings.
type AlgorithmRegistry struct {
	algorithms map[string]SignatureAlgorithm
	mu         sync.RWMutex
}

// NewAlgorithmRegistry creates a registry with default algorithms registered.
func NewAlgorithmRegistry() *AlgorithmRegistry {
	registry := &AlgorithmRegistry{
		algorithms: make(map[string]SignatureAlgorithm),
	}

	// Register default algorithms
	registry.Register(AlgorithmEd25519, &Ed25519Provider{})
	registry.Register(AlgorithmRSAPSS, NewRSAPSSProvider(2048))
	registry.Register(AlgorithmECDSAP256, &ECDSAP256Provider{})

	return registry
}

// Register adds or updates an algorithm in the registry.
func (r *AlgorithmRegistry) Register(algorithmID string, provider SignatureAlgorithm) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.algorithms[algorithmID] = provider
}

// Get retrieves an algorithm provider by its identifier.
func (r *AlgorithmRegistry) Get(algorithmID string) (SignatureAlgorithm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.algorithms[algorithmID]
	if !exists {
		return nil, fmt.Errorf("unknown algorithm: %s", algorithmID)
	}
	return provider, nil
}

// ListAlgorithms returns all registered algorithm identifiers.
func (r *AlgorithmRegistry) ListAlgorithms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.algorithms))
	for id := range r.algorithms {
		ids = append(ids, id)
	}
	return ids
}

// DefaultRegistry is a global registry with standard algorithms pre-registered.
var DefaultRegistry = NewAlgorithmRegistry()
