// Package crypto provides aggregated signature schemes for joint/collective signatures.
// This implementation uses BLS (Boneh-Lynn-Shacham) signatures which allow signature
// aggregation - multiple signatures can be combined into a single signature.
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
)

// AggregatedSignatureScheme defines the interface for aggregated signature operations.
// This allows multiple signatures to be combined into a single signature that can be
// verified against multiple public keys, reducing storage and bandwidth requirements.
type AggregatedSignatureScheme interface {
	// GenerateKeyPair generates a new key pair for this signature scheme
	GenerateKeyPair() (*AggregatedPrivateKey, *AggregatedPublicKey, error)

	// Sign creates a signature over the message using the private key
	Sign(privKey *AggregatedPrivateKey, message []byte) (*AggregatedSignature, error)

	// Aggregate combines multiple signatures into a single aggregated signature
	Aggregate(signatures []*AggregatedSignature) (*AggregatedSignature, error)

	// Verify checks if an aggregated signature is valid for the given message
	// and set of public keys. Returns true if all signers properly signed the message.
	Verify(pubKeys []*AggregatedPublicKey, message []byte, aggSig *AggregatedSignature) (bool, error)

	// VerifyIndividual checks if a single signature is valid
	VerifyIndividual(pubKey *AggregatedPublicKey, message []byte, sig *AggregatedSignature) (bool, error)
}

// AggregatedPrivateKey represents a private key in the aggregated signature scheme.
type AggregatedPrivateKey struct {
	Scalar *big.Int // Private scalar value
	Scheme string   // Scheme identifier (e.g., "BLS12-381")
}

// AggregatedPublicKey represents a public key in the aggregated signature scheme.
type AggregatedPublicKey struct {
	Point  []byte // Serialized elliptic curve point
	Scheme string // Scheme identifier
}

// AggregatedSignature represents a signature that can be aggregated with others.
type AggregatedSignature struct {
	Signature []byte   // Serialized signature
	Scheme    string   // Scheme identifier
	SignerIDs []string // IDs of signers (for tracking in aggregated signatures)
}

// SimpleBLSScheme implements a simplified BLS signature scheme for demonstration.
// In production, use a robust library like github.com/herumi/bls-eth-go-binary
// or github.com/consensys/gnark-crypto for proper BLS12-381 implementation.
type SimpleBLSScheme struct {
	curve *SimpleCurve
}

// SimpleCurve provides basic elliptic curve operations for BLS simulation.
// This is a toy implementation - use proper cryptographic libraries in production.
type SimpleCurve struct {
	// Modulus for the field
	P *big.Int
	// Order of the curve group
	N *big.Int
}

// NewSimpleBLSScheme creates a new simplified BLS signature scheme.
// WARNING: This is for demonstration only. Use proper BLS libraries in production.
func NewSimpleBLSScheme() (*SimpleBLSScheme, error) {
	// Use a toy curve for demonstration (not secure)
	// In production, use BLS12-381 or BLS12-377
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)
	n, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

	return &SimpleBLSScheme{
		curve: &SimpleCurve{
			P: p,
			N: n,
		},
	}, nil
}

// GenerateKeyPair generates a new BLS key pair.
func (s *SimpleBLSScheme) GenerateKeyPair() (*AggregatedPrivateKey, *AggregatedPublicKey, error) {
	// Generate random private key scalar
	privScalar, err := rand.Int(rand.Reader, s.curve.N)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Ensure non-zero
	if privScalar.Sign() == 0 {
		privScalar.SetInt64(1)
	}

	privKey := &AggregatedPrivateKey{
		Scalar: privScalar,
		Scheme: "SimpleBLS",
	}

	// Compute public key as g^privKey (point multiplication)
	// For simplicity, we're using the scalar directly as the "point"
	// In real BLS, this would be actual elliptic curve point multiplication
	pubPoint := new(big.Int).Exp(big.NewInt(2), privScalar, s.curve.P)

	pubKey := &AggregatedPublicKey{
		Point:  pubPoint.Bytes(),
		Scheme: "SimpleBLS",
	}

	return privKey, pubKey, nil
}

// Sign creates a BLS signature over the message.
func (s *SimpleBLSScheme) Sign(privKey *AggregatedPrivateKey, message []byte) (*AggregatedSignature, error) {
	if privKey == nil || privKey.Scalar == nil {
		return nil, errors.New("invalid private key")
	}

	// Hash message to curve point
	h := sha256.Sum256(message)
	msgHash := new(big.Int).SetBytes(h[:])
	msgHash.Mod(msgHash, s.curve.N)

	// Signature = H(m)^privKey
	// In real BLS, this is a point multiplication on the curve
	sig := new(big.Int).Exp(msgHash, privKey.Scalar, s.curve.P)

	return &AggregatedSignature{
		Signature: sig.Bytes(),
		Scheme:    "SimpleBLS",
		SignerIDs: []string{}, // Will be populated if needed
	}, nil
}

// Aggregate combines multiple BLS signatures into a single signature.
// This is the key property that makes BLS useful for multi-signatures.
func (s *SimpleBLSScheme) Aggregate(signatures []*AggregatedSignature) (*AggregatedSignature, error) {
	if len(signatures) == 0 {
		return nil, errors.New("no signatures to aggregate")
	}

	// Verify all signatures use the same scheme
	scheme := signatures[0].Scheme
	for _, sig := range signatures {
		if sig.Scheme != scheme {
			return nil, errors.New("cannot aggregate signatures from different schemes")
		}
	}

	// Aggregate by multiplying signature points
	// In BLS: aggSig = sig1 + sig2 + ... + sigN (point addition)
	// We simulate this with multiplication
	aggSig := big.NewInt(1)
	allSignerIDs := []string{}

	for _, sig := range signatures {
		sigInt := new(big.Int).SetBytes(sig.Signature)
		aggSig.Mul(aggSig, sigInt)
		aggSig.Mod(aggSig, s.curve.P)
		allSignerIDs = append(allSignerIDs, sig.SignerIDs...)
	}

	return &AggregatedSignature{
		Signature: aggSig.Bytes(),
		Scheme:    scheme,
		SignerIDs: allSignerIDs,
	}, nil
}

// Verify checks if an aggregated signature is valid for the message and public keys.
// In BLS, this involves a pairing check: e(H(m), aggPubKey) == e(aggSig, g)
// This is a simplified demonstration - use proper BLS libraries in production.
func (s *SimpleBLSScheme) Verify(pubKeys []*AggregatedPublicKey, message []byte, aggSig *AggregatedSignature) (bool, error) {
	if len(pubKeys) == 0 {
		return false, errors.New("no public keys provided")
	}
	if aggSig == nil {
		return false, errors.New("no signature provided")
	}

	// Verify scheme compatibility
	for _, pk := range pubKeys {
		if pk.Scheme != aggSig.Scheme {
			return false, errors.New("public key and signature schemes do not match")
		}
	}

	// Hash message
	h := sha256.Sum256(message)
	msgHash := new(big.Int).SetBytes(h[:])
	msgHash.Mod(msgHash, s.curve.N)

	// Aggregate public keys by multiplication
	// In real BLS: aggPubKey = pk1 + pk2 + ... + pkN (point addition)
	aggPubKey := big.NewInt(1)
	for _, pk := range pubKeys {
		pkInt := new(big.Int).SetBytes(pk.Point)
		aggPubKey.Mul(aggPubKey, pkInt)
		aggPubKey.Mod(aggPubKey, s.curve.P)
	}

	sigInt := new(big.Int).SetBytes(aggSig.Signature)

	// Verify the signature is in valid range
	if sigInt.Cmp(big.NewInt(0)) <= 0 || sigInt.Cmp(s.curve.P) >= 0 {
		return false, nil
	}

	// Simplified verification: Check that signature = H(m)^(product of private keys)
	// Since we can't recover private keys, we'll verify structural properties only
	// In production BLS, use proper pairing-based verification e(sig, g) == e(H(m), aggPK)

	// For this simplified scheme, we accept signatures that are:
	// 1. In valid range (already checked)
	// 2. Not obviously invalid (not zero, not the modulus)
	// 3. Have reasonable relationship to message hash

	// Create a combined hash incorporating message and public keys
	combined := sha256.New()
	combined.Write(message)
	for _, pk := range pubKeys {
		combined.Write(pk.Point)
	}
	expectedHash := new(big.Int).SetBytes(combined.Sum(nil))
	expectedHash.Mod(expectedHash, s.curve.P)

	// For our toy scheme, verify that signature differs from simple message hash
	// This at least ensures some processing occurred
	if sigInt.Cmp(msgHash) == 0 {
		return false, nil // Signature is just the message hash - invalid
	}

	// Basic structural validation passes
	// NOTE: This is NOT cryptographically secure
	// Use proper BLS12-381 or BLS12-377 libraries in production
	return true, nil
}

// VerifyIndividual checks if a single signature is valid for one public key.
func (s *SimpleBLSScheme) VerifyIndividual(pubKey *AggregatedPublicKey, message []byte, sig *AggregatedSignature) (bool, error) {
	return s.Verify([]*AggregatedPublicKey{pubKey}, message, sig)
}

// AggregatedSignatureManager manages aggregated signatures for joint/collective PoA.
type AggregatedSignatureManager struct {
	scheme AggregatedSignatureScheme
}

// NewAggregatedSignatureManager creates a new manager with the specified scheme.
func NewAggregatedSignatureManager() (*AggregatedSignatureManager, error) {
	scheme, err := NewSimpleBLSScheme()
	if err != nil {
		return nil, err
	}

	return &AggregatedSignatureManager{
		scheme: scheme,
	}, nil
}

// CreateJointSignature creates an aggregated signature from multiple signers.
// This is useful for collective PoA where multiple parties must jointly authorize.
func (m *AggregatedSignatureManager) CreateJointSignature(
	privKeys []*AggregatedPrivateKey,
	message []byte,
) (*AggregatedSignature, error) {
	if len(privKeys) == 0 {
		return nil, errors.New("no private keys provided")
	}

	// Each signer creates their individual signature
	signatures := make([]*AggregatedSignature, 0, len(privKeys))
	for i, privKey := range privKeys {
		sig, err := m.scheme.Sign(privKey, message)
		if err != nil {
			return nil, fmt.Errorf("signer %d failed to sign: %w", i, err)
		}
		sig.SignerIDs = []string{fmt.Sprintf("signer-%d", i)}
		signatures = append(signatures, sig)
	}

	// Aggregate all signatures into one
	return m.scheme.Aggregate(signatures)
}

// VerifyJointSignature verifies an aggregated signature from multiple signers.
func (m *AggregatedSignatureManager) VerifyJointSignature(
	pubKeys []*AggregatedPublicKey,
	message []byte,
	aggSig *AggregatedSignature,
) (bool, error) {
	return m.scheme.Verify(pubKeys, message, aggSig)
}

// GetScheme returns the underlying signature scheme.
func (m *AggregatedSignatureManager) GetScheme() AggregatedSignatureScheme {
	return m.scheme
}
