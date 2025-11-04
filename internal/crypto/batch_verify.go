package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"math/big"

	blslib "github.com/herumi/bls-eth-go-binary/bls"
)

// BatchVerifyEd25519 performs naive batch verification (per-signature) over Ed25519.
// A future optimization can use cofactoring techniques; current implementation short-circuits on first failure.
func BatchVerifyEd25519(publicKeys []interface{}, messages [][]byte, signatures [][]byte) bool {
	n := len(publicKeys)
	if n == 0 || len(messages) != n || len(signatures) != n {
		return false
	}
	for i := 0; i < n; i++ {
		pk, ok := publicKeys[i].(ed25519.PublicKey)
		if !ok || len(pk) != ed25519.PublicKeySize {
			return false
		}
		if !ed25519.Verify(pk, messages[i], signatures[i]) {
			return false
		}
	}
	return true
}

// BatchVerifyECDSA performs naive per-signature verification using SHA-256.
// Signatures provided as concatenated r||s (fixed length cannot be assumed across curves; we parse midpoint).
func BatchVerifyECDSA(publicKeys []interface{}, messages [][]byte, signatures [][]byte) bool {
	n := len(publicKeys)
	if n == 0 || len(messages) != n || len(signatures) != n {
		return false
	}
	for i := 0; i < n; i++ {
		pub, ok := publicKeys[i].(*ecdsa.PublicKey)
		if !ok {
			return false
		}
		sig := signatures[i]
		// Basic split r||s (heuristic: half length)
		if len(sig) < 2 || len(sig)%2 != 0 {
			return false
		}
		mid := len(sig) / 2
		r := new(big.Int).SetBytes(sig[:mid])
		s := new(big.Int).SetBytes(sig[mid:])
		h := sha256.Sum256(messages[i])
		if !ecdsa.Verify(pub, h[:], r, s) {
			return false
		}
	}
	return true
}

// BatchVerifyBLS performs naive verification of individual signatures; if all pass returns true.
func BatchVerifyBLS(publicKeys []interface{}, messages [][]byte, signatures [][]byte) bool {
	n := len(publicKeys)
	if n == 0 || len(messages) != n || len(signatures) != n {
		return false
	}
	for i := 0; i < n; i++ {
		pk, ok := publicKeys[i].(blslib.PublicKey)
		if !ok {
			return false
		}
		if !BLSVerify(&BLSKey{Public: pk}, messages[i], signatures[i]) {
			return false
		}
	}
	return true
}
