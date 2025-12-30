package crypto

import (
	"crypto/rand"
	"fmt"

	bls12381 "github.com/kilic/bls12-381"
)

// BLSProvider implements AggregationProvider using BLS12-381 curve (MinSignature variant).
// Signatures are points on G1 (48 bytes compressed), Public Keys are points on G2 (96 bytes compressed).
// Note: kilic/bls12-381 defaults: G1 is 48 bytes, G2 is 96 bytes.
// Standard usually puts signatures on G2 (malleability check simpler) and PK on G1 (smaller),
// or vice versa (MinSig vs MinPk).
// This implementation assumes:
// - Signatures on G1
// - Public Keys on G2
type BLSProvider struct {
	domain []byte
}

// NewBLSProvider creates a new provider with a domain separation tag.
func NewBLSProvider(dst string) *BLSProvider {
	if dst == "" {
		dst = "AGENTAUTH_BLS_SIG_V1"
	}
	return &BLSProvider{
		domain: []byte(dst),
	}
}

// Aggregate combines multiple compressed signatures into one.
func (p *BLSProvider) Aggregate(signatures [][]byte) ([]byte, error) {
	if len(signatures) == 0 {
		return nil, ErrAggregateFail
	}

	g1 := bls12381.NewG1()

	// Deserialize first signature to start accumulator
	current, err := g1.FromCompressed(signatures[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse signature 0: %w", err)
	}

	// Add remaining signatures
	for i := 1; i < len(signatures); i++ {
		nextSig, err := g1.FromCompressed(signatures[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse signature %d: %w", i, err)
		}
		g1.Add(current, current, nextSig)
	}

	return g1.ToCompressed(current), nil
}

// VerifyAggregated verifies a single aggregated signature against multiple public keys for the SAME message.
// This implements BLS Multi-Signature validation: e(g1_agg, proof_of_possession?) == e(H(m), pk_agg)
// Wait, simple aggregation requires protection against rogue key attacks (Proof of Possession - PoP).
// We assume here that public keys are verified/trusted (registered in system).
// If keys are distinct and message is same, we can aggregate public keys and verify against aggregated signature.
// Verification: e(sig_agg, g2) == e(H(m), pk_agg)
func (p *BLSProvider) VerifyAggregated(pubKeys [][]byte, message []byte, signature []byte) error {
	if len(pubKeys) == 0 {
		return ErrInvalidKey
	}

	// 1. Aggregate Public Keys
	g2 := bls12381.NewG2()
	aggPK := g2.New() // Identity

	// We need to handle identity properly, but kilic libs usually initialize zero point.
	// Actually better to parse first and start.

	// Parse signature
	g1 := bls12381.NewG1()
	sigPoint, err := g1.FromCompressed(signature)
	if err != nil {
		return ErrInvalidSignature
	}

	// Aggregate PKs
	// Start with first
	firstPK, err := g2.FromCompressed(pubKeys[0])
	if err != nil {
		return fmt.Errorf("invalid pubkey 0: %w", err)
	}
	// Copy firstPK to aggPK
	// g2.Set undefined, use Add with Zero
	g2.Add(aggPK, firstPK, g2.Zero())

	for i := 1; i < len(pubKeys); i++ {
		nextPK, err := g2.FromCompressed(pubKeys[i])
		if err != nil {
			return fmt.Errorf("invalid pubkey %d: %w", i, err)
		}
		g2.Add(aggPK, aggPK, nextPK)
	}

	// 2. Perform Pairing Check
	// e(sig, g2_generator) == e(hash(msg), pk_agg) ??
	// No, pairing equation is e(sig, g2) = e(H(m), pk) traditionally?
	// kilic library helper usually Check?
	// Let's use manual pairing engine.

	engine := bls12381.NewEngine()

	// H(m)
	// MapToG1 for H(m) since signatures are on G1
	hm, err := g1.HashToCurve(message, p.domain)
	if err != nil {
		return fmt.Errorf("hash to curve failed: %w", err)
	}

	// Equation: e(sig, One) == e(hm, pk_agg)
	// => e(sig, One) * e(hm, pk_agg)^-1 == 1
	// => e(sig, One) + e(hm, -pk_agg) == 0 (in additive notation loop)
	// kilic engine AddPair(g1, g2) and Check() checks if sum is Zero/Unity.
	// Note: G2 is usually PK side.

	// We need e(sig, g2_generator) == e(hm, pk_agg) ??
	// Standard BLS: verify(pk, m, sig): e(sig, g2) == e(H(m), pk)
	// Here g2 is Generator point on G2. pk is public key on G2.

	// We need negation of one element to check equality via pairing product = 1
	// engine.AddPair(sig, g2.One())
	// engine.AddPair(hm, g2.Neg(pk_agg))

	one := g2.One()

	// Negate PK Agg
	negAggPK := g2.New()
	g2.Neg(negAggPK, aggPK)

	engine.AddPair(sigPoint, one)
	engine.AddPair(hm, negAggPK)

	if !engine.Check() {
		// Let's try pairing check configuration
		// e(sig, -One) * e(hm, aggPK) == 1

		negOne := g2.New()
		g2.Neg(negOne, one)

		engine.Reset()
		return checkPairing(engine, sigPoint, negOne, hm, aggPK)
	}

	return nil
}

func checkPairing(engine *bls12381.Engine, p1 *bls12381.PointG1, q1 *bls12381.PointG2, p2 *bls12381.PointG1, q2 *bls12381.PointG2) error {
	engine.AddPair(p1, q1)
	engine.AddPair(p2, q2)
	if !engine.Check() {
		return ErrInvalidSignature
	}
	return nil
}

func (p *BLSProvider) VerifyBatch(publicKeys []interface{}, messages [][]byte, signatures [][]byte) error {
	if len(publicKeys) != len(messages) || len(publicKeys) != len(signatures) {
		return fmt.Errorf("batch length mismatch")
	}
	for i, pk := range publicKeys {
		if err := p.Verify(pk, messages[i], signatures[i]); err != nil {
			return err
		}
	}
	return nil
}

// Sign generates a BLS signature over the message.
func (p *BLSProvider) Sign(privateKey interface{}, message []byte) ([]byte, error) {
	sk, ok := privateKey.(*bls12381.Fr)
	if !ok {
		return nil, fmt.Errorf("invalid key type: expected *bls12381.Fr, got %T", privateKey)
	}

	g1 := bls12381.NewG1()
	hm, err := g1.HashToCurve(message, p.domain)
	if err != nil {
		return nil, fmt.Errorf("hash to curve failed: %w", err)
	}

	sig := g1.New()
	g1.MulScalar(sig, hm, sk)
	return g1.ToCompressed(sig), nil
}

// Verify checks a BLS signature against the message.
func (p *BLSProvider) Verify(publicKey interface{}, message, signature []byte) error {
	pk, ok := publicKey.(*bls12381.PointG2)
	if !ok {
		// Might be raw bytes?
		if pkBytes, ok := publicKey.([]byte); ok {
			g2 := bls12381.NewG2()
			var err error
			pk, err = g2.FromCompressed(pkBytes)
			if err != nil {
				return fmt.Errorf("failed to parse pubkey bytes: %w", err)
			}
		} else {
			return fmt.Errorf("invalid key type: expected *bls12381.PointG2, got %T", publicKey)
		}
	}

	g1 := bls12381.NewG1()
	sigPoint, err := g1.FromCompressed(signature)
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	// Verify pairing: e(sig, g2.One) == e(H(m), pk)
	g2 := bls12381.NewG2()
	engine := bls12381.NewEngine()

	hm, err := g1.HashToCurve(message, p.domain)
	if err != nil {
		return fmt.Errorf("hash to curve failed: %w", err)
	}

	// e(sig, g2.One) + e(hm, -pk) == 0 (Unity in multiplicative target group)
	one := g2.One()
	negPk := g2.New()
	g2.Neg(negPk, pk)

	engine.AddPair(sigPoint, one)
	engine.AddPair(hm, negPk)

	if !engine.Check() {
		return fmt.Errorf("invalid BLS signature")
	}
	return nil
}

// AlgorithmID returns the identifier.
func (p *BLSProvider) AlgorithmID() string {
	return "BLS12-381"
}

// KeyType returns the private key type.
func (p *BLSProvider) KeyType() string {
	return "*bls12381.Fr"
}

// GenerateKey generates a new key pair.
func (p *BLSProvider) GenerateKey() (interface{}, interface{}, error) {
	sk := bls12381.NewFr()
	_, err := sk.Rand(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	g2 := bls12381.NewG2()
	pk := g2.New()
	g2.MulScalar(pk, g2.One(), sk)
	return sk, pk, nil
}

// MarshalPrivateKey (Stub - minimal impl)
func (p *BLSProvider) MarshalPrivateKey(privateKey interface{}) ([]byte, error) {
	sk, ok := privateKey.(*bls12381.Fr)
	if !ok {
		return nil, fmt.Errorf("invalid key type")
	}
	return sk.ToBytes(), nil
}

// UnmarshalPrivateKey (Stub)
func (p *BLSProvider) UnmarshalPrivateKey(pemData []byte) (interface{}, error) {
	// Not full PEM, just bytes for skeleton
	sk := bls12381.NewFr()
	sk.FromBytes(pemData)
	return sk, nil
}

// MarshalPublicKey (Stub)
func (p *BLSProvider) MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	pk, ok := publicKey.(*bls12381.PointG2)
	if !ok {
		return nil, fmt.Errorf("invalid key type")
	}
	g2 := bls12381.NewG2()
	return g2.ToCompressed(pk), nil
}

// UnmarshalPublicKey (Stub)
func (p *BLSProvider) UnmarshalPublicKey(pemData []byte) (interface{}, error) {
	g2 := bls12381.NewG2()
	return g2.FromCompressed(pemData)
}
