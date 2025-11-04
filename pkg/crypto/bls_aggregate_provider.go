package crypto

// Aggregated BLS (BLS12-381) algorithm registration and helper provider.
// This exposes algorithm name `bls12-381-agg` for multi-signature aggregation
// over IDENTICAL canonical message bytes. All participants must have produced
// valid individual BLS signatures on the same message. Aggregation compresses
// N signatures/public keys to one aggregated signature + list of key IDs.
//
// Verification contract (AggregatedVerify):
// - messages: must contain exactly one element (the canonical message)
// - signatures: must contain exactly one element (the aggregated signature base64)
// - keyIDs: list of participant key IDs (>=1)
// - key provider: must return public keys whose algorithm is AlgoBLS12381
//   for each key ID.
// Errors returned when any structural or cryptographic check fails.

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"

	bls "github.com/herumi/bls-eth-go-binary/bls"
)

const AlgoBLS12381Agg = "bls12-381-agg"

// multiBLSKeyProvider is a minimal KeyProvider implementation supporting
// multiple BLS public keys (verify-only). Used by tests and callers wanting
// an in-memory collection of participant keys.
// It satisfies KeyProvider but ActiveSigner/VerifyWith are no-ops for aggregation.
// PublicKey returns AlgoBLS12381 for each stored key.
// NOTE: This is separate from InMemoryBLSProvider (single active key) to allow
// multi-key scenarios needed for aggregated verification tests.

type multiBLSKeyProvider struct {
	publics map[string][]byte // raw serialized BLS public keys
}

func newMultiBLSKeyProvider() *multiBLSKeyProvider {
	return &multiBLSKeyProvider{publics: make(map[string][]byte)}
}

func (m *multiBLSKeyProvider) ActiveSigner() (Signer, error) {
	return nil, errors.New("no active signer for aggregate provider")
}
func (m *multiBLSKeyProvider) PublicKey(keyID string) ([]byte, string, error) {
	pk, ok := m.publics[keyID]
	if !ok {
		return nil, "", ErrUnknownKey
	}
	return pk, AlgoBLS12381, nil
}
func (m *multiBLSKeyProvider) VerifyWith(msg, sig []byte, keyID string) error {
	return errors.New("single verify unsupported for aggregate provider")
}

// AddPublic inserts a BLS public key using derived key ID.
func (m *multiBLSKeyProvider) AddPublic(raw []byte) (string, error) {
	// deriveKeyID expects ed25519.PublicKey (alias []byte); safe conversion
	kid := deriveKeyID(ed25519.PublicKey(raw))
	m.publics[kid] = append([]byte(nil), raw...)
	return kid, nil
}

// GenerateBLSKeys creates n new BLS key pairs and returns their key IDs and private keys.
// The public keys are stored for future verification. Private keys returned for signing.
func (m *multiBLSKeyProvider) GenerateBLSKeys(n int) (keyIDs []string, privs []bls.SecretKey, err error) {
	if n <= 0 {
		return nil, nil, errors.New("n must be >0")
	}
	if err := bls.Init(bls.BLS12_381); err != nil {
		return nil, nil, err
	}
	for i := 0; i < n; i++ {
		var sk bls.SecretKey
		sk.SetByCSPRNG()
		pk := sk.GetPublicKey()
		kid := deriveKeyID(ed25519.PublicKey(pk.Serialize()))
		m.publics[kid] = pk.Serialize()
		keyIDs = append(keyIDs, kid)
		privs = append(privs, sk)
	}
	return keyIDs, privs, nil
}

// AggregateSign signs canonical message with provided secret keys and aggregates resulting signatures.
func AggregateSign(message []byte, privs []bls.SecretKey) ([]byte, error) {
	if len(privs) == 0 {
		return nil, errors.New("no private keys")
	}
	// Aggregate by group addition of individual signatures.
	var agg bls.Sign
	for i, sk := range privs {
		sig := sk.SignByte(message)
		if i == 0 {
			agg = *sig
		} else {
			agg.Add(sig)
		}
	}
	return agg.Serialize(), nil
}

// aggregateVerify reproduces internal aggregation verification: aggregates public keys then verifies.
func aggregateVerify(pubKeys []bls.PublicKey, message, aggSig []byte) bool {
	if len(pubKeys) == 0 {
		return false
	}
	var aggPK bls.PublicKey
	for i, pk := range pubKeys {
		if i == 0 {
			aggPK = pk
		} else {
			aggPK.Add(&pk)
		}
	}
	var sig bls.Sign
	if err := sig.Deserialize(aggSig); err != nil {
		return false
	}
	return sig.VerifyByte(&aggPK, message)
}

func init() {
	// Register aggregated BLS algorithm with AggregatedVerify handler.
	RegisterAlgorithm(Algorithm{Name: AlgoBLS12381Agg, Verify: func(canonical []byte, sigBase64 string, keyID string, kp KeyProvider) error {
		return errors.New("bls12-381-agg: single signature verify unsupported; use aggregated path")
	}, AggregatedVerify: func(messages [][]byte, signatures []string, keyIDs []string, kp KeyProvider) error {
		// Structural checks
		if len(messages) != 1 {
			return fmt.Errorf("%s: expected exactly 1 message", AlgoBLS12381Agg)
		}
		if len(signatures) != 1 {
			return fmt.Errorf("%s: expected exactly 1 aggregated signature", AlgoBLS12381Agg)
		}
		if len(keyIDs) == 0 {
			return fmt.Errorf("%s: no key IDs provided", AlgoBLS12381Agg)
		}
		if kp == nil {
			return fmt.Errorf("%s: missing key provider", AlgoBLS12381Agg)
		}
		msg := messages[0]
		aggSigBytes, err := base64.StdEncoding.DecodeString(signatures[0])
		if err != nil {
			return fmt.Errorf("%s: base64 decode aggregated signature: %w", AlgoBLS12381Agg, err)
		}
		// Collect and deserialize public keys
		var pubKeys []bls.PublicKey
		for _, kid := range keyIDs {
			pkRaw, algo, err := kp.PublicKey(kid)
			if err != nil {
				return fmt.Errorf("%s: public key lookup failed for %s: %w", AlgoBLS12381Agg, kid, err)
			}
			if algo != AlgoBLS12381 {
				return fmt.Errorf("%s: key %s algorithm mismatch: %s", AlgoBLS12381Agg, kid, algo)
			}
			var pk bls.PublicKey
			if err := pk.Deserialize(pkRaw); err != nil {
				return fmt.Errorf("%s: deserialize public key %s failed", AlgoBLS12381Agg, kid)
			}
			pubKeys = append(pubKeys, pk)
		}
		if !aggregateVerify(pubKeys, msg, aggSigBytes) {
			return errors.New("aggregated signature verification failed")
		}
		return nil
	}})
}
