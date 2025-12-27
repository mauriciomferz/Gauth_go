package crypto

import "errors"

// AggregationProvider defines methods for signature aggregation and verification.
// It supports schemes like BLS that allow compressing multiple signatures into one.
type AggregationProvider interface {
	// Aggregate combines multiple signatures into a single aggregated signature.
	// All signatures must be over the same message distinct messages depending on the scheme.
	// For BLS basic scheme: same message, distinct signers (aggregated public key).
	// For BLS aggregation scheme: distinct messages, distinct signers.
	// This interface assumes the common use case needed for PoA:
	// "n-of-m" signers signing the SAME message (Multi-Signature).
	Aggregate(signatures [][]byte) ([]byte, error)

	// VerifyAggregated verifies an aggregated signature against a list of public keys and a message.
	// The provider implementation handles key aggregation internally.
	VerifyAggregated(pubKeys [][]byte, message []byte, signature []byte) error

	// VerifyBatch verified multiple distinct (pubKey, message, signature) triples efficiently.
	// This is slightly different from Aggregate verification but useful for high throughput.
	VerifyBatch(pubKeys [][]byte, messages [][]byte, signatures [][]byte) error
}

// ErrInvalidSignature is returned when verification fails.
var ErrInvalidSignature = errors.New("invalid signature")

// ErrInvalidKey is returned when a public key cannot be parsed.
var ErrInvalidKey = errors.New("invalid public key")

// ErrAggregateFail is returned when aggregation fails.
var ErrAggregateFail = errors.New("aggregation failed")
