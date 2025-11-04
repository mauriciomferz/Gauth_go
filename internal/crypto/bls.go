package crypto

import (
	"errors"

	bls "github.com/herumi/bls-eth-go-binary/bls"
)

// BLSKey represents a BLS key pair.
type BLSKey struct {
	Private bls.SecretKey
	Public  bls.PublicKey
}

// GenerateBLSKey generates a new BLS key pair.
func GenerateBLSKey() (*BLSKey, error) {
	err := bls.Init(bls.BLS12_381)
	if err != nil {
		return nil, err
	}
	var sk bls.SecretKey
	sk.SetByCSPRNG()
	pk := sk.GetPublicKey()
	return &BLSKey{Private: sk, Public: *pk}, nil
}

// NewBLSPublicKey constructs a public-only BLSKey from serialized public key bytes.
// This allows verify-only vector tests without private key material.
func NewBLSPublicKey(pubBytes []byte) (*BLSKey, error) {
	if err := bls.Init(bls.BLS12_381); err != nil {
		return nil, err
	}
	var pk bls.PublicKey
	if err := pk.Deserialize(pubBytes); err != nil {
		return nil, err
	}
	return &BLSKey{Public: pk}, nil
}

// BLSSign signs a message with the BLS private key.
func BLSSign(key *BLSKey, message []byte) ([]byte, error) {
	sig := key.Private.SignByte(message)
	return sig.Serialize(), nil
}

// BLSVerify verifies a BLS signature.
func BLSVerify(key *BLSKey, message, signature []byte) bool {
	var sig bls.Sign
	if err := sig.Deserialize(signature); err != nil {
		return false
	}
	return sig.VerifyByte(&key.Public, message)
}

// BLSAggregate aggregates multiple BLS signatures over the SAME message.
// All signatures must be valid individually; aggregation is performed by group addition.
func BLSAggregate(signatures [][]byte) ([]byte, error) {
	if len(signatures) == 0 {
		return nil, errors.New("no signatures to aggregate")
	}
	var agg bls.Sign
	for i, s := range signatures {
		var sig bls.Sign
		if err := sig.Deserialize(s); err != nil {
			return nil, errors.New("invalid signature at index ")
		}
		if i == 0 {
			agg = sig
		} else {
			agg.Add(&sig)
		}
	}
	return agg.Serialize(), nil
}

// BLSAggregateVerify verifies an aggregated signature against aggregated public keys for SAME message.
// Assumes all participants signed identical message.
func BLSAggregateVerify(pubKeys []bls.PublicKey, message, aggSig []byte) bool {
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
