package crypto

import (
	"encoding/base64"
	"testing"
)

// Fuzz test for Ed25519 signature verification with malformed inputs
func FuzzEd25519SignatureVerification(f *testing.F) {
	// Seed with valid cases
	f.Add([]byte("valid message"), []byte("valid signature base64"), "key123")
	f.Add([]byte(""), []byte("ZmFrZV9zaWduYXR1cmVfZGF0YQ=="), "testkey")
	f.Add([]byte("test"), []byte(""), "k1")

	f.Fuzz(func(t *testing.T, msg []byte, sigB64Bytes []byte, keyID string) {
		// Create provider
		prov, err := NewInMemoryEd25519Provider()
		if err != nil {
			t.Skip("provider init failed")
		}

		// Attempt to verify with fuzzed inputs
		// Should never panic, only return error for invalid inputs
		_ = VerifyAlgorithm(AlgoEd25519, msg, string(sigB64Bytes), keyID, prov)
	})
}

// Fuzz test for ECDSA P-256 signature verification with malformed inputs
func FuzzECDSASignatureVerification(f *testing.F) {
	// Seed corpus
	f.Add([]byte("canonical"), []byte("TWFsbGVhYmxlRVJERVJERVJERVI="), "eckey")
	f.Add([]byte("x"), []byte("!@#$%^&*()"), "bad")
	f.Add(make([]byte, 1024), []byte(""), "")

	f.Fuzz(func(t *testing.T, msg []byte, sigB64Bytes []byte, keyID string) {
		prov, err := NewInMemoryECDSAProvider()
		if err != nil {
			t.Skip("ecdsa provider init failed")
		}

		// Fuzzed verification should not panic
		_ = VerifyAlgorithm(AlgoECDSAP256, msg, string(sigB64Bytes), keyID, prov)
	})
}

// Fuzz test for BLS12-381 signature verification with malformed inputs
func FuzzBLSSignatureVerification(f *testing.F) {
	// Seed corpus
	f.Add([]byte("bls message"), []byte("Qkxex1NJR19EQVRB"), "blskey")
	f.Add([]byte(""), []byte(""), "")

	f.Fuzz(func(t *testing.T, msg []byte, sigB64Bytes []byte, keyID string) {
		prov, err := NewInMemoryBLSProvider()
		if err != nil {
			t.Skip("bls provider init failed")
		}

		// Should handle gracefully
		_ = VerifyAlgorithm(AlgoBLS12381, msg, string(sigB64Bytes), keyID, prov)
	})
}

// Fuzz test for signature round-trip stability with arbitrary messages
func FuzzSignatureRoundTrip_Ed25519(f *testing.F) {
	// Seed with various message types
	f.Add([]byte("normal text"))
	f.Add([]byte{0x00, 0xFF, 0xAA, 0x55}) // binary
	f.Add([]byte(""))                     // empty
	f.Add(make([]byte, 1024))             // large

	f.Fuzz(func(t *testing.T, msg []byte) {
		prov, err := NewInMemoryEd25519Provider()
		if err != nil {
			t.Skip()
		}
		signer, err := prov.ActiveSigner()
		if err != nil {
			t.Skip()
		}

		// Sign arbitrary message
		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("sign failed for valid key: %v", err)
		}

		// Verify must succeed
		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoEd25519, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("round-trip verification failed: %v", err)
		}

		// Tampered message must fail
		if len(msg) > 0 {
			tampered := make([]byte, len(msg))
			copy(tampered, msg)
			tampered[0] ^= 0x01
			if err := VerifyAlgorithm(AlgoEd25519, tampered, b64, signer.KeyID(), prov); err == nil {
				t.Fatalf("tampered message should fail verification")
			}
		}
	})
}

// Fuzz test for ECDSA signature round-trip with arbitrary messages
func FuzzSignatureRoundTrip_ECDSA(f *testing.F) {
	f.Add([]byte("ecdsa test"))
	f.Add(make([]byte, 512))

	f.Fuzz(func(t *testing.T, msg []byte) {
		prov, err := NewInMemoryECDSAProvider()
		if err != nil {
			t.Skip()
		}
		signer, _ := prov.ActiveSigner()

		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("ecdsa sign failed: %v", err)
		}

		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoECDSAP256, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("ecdsa round-trip failed: %v", err)
		}
	})
}

// Fuzz test for BLS signature round-trip with arbitrary messages
func FuzzSignatureRoundTrip_BLS(f *testing.F) {
	f.Add([]byte("bls canonical"))
	f.Add([]byte{0xDE, 0xAD, 0xBE, 0xEF})

	f.Fuzz(func(t *testing.T, msg []byte) {
		prov, err := NewInMemoryBLSProvider()
		if err != nil {
			t.Skip()
		}
		signer, _ := prov.ActiveSigner()

		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("bls sign failed: %v", err)
		}

		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoBLS12381, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("bls round-trip failed: %v", err)
		}
	})
}

// Fuzz test for algorithm registry lookup with arbitrary algorithm names
func FuzzAlgorithmRegistryLookup(f *testing.F) {
	f.Add("ed25519")
	f.Add("ecdsa-p256")
	f.Add("bls12-381")
	f.Add("unknown-algo")
	f.Add("")
	f.Add("SQL INJECTION'; DROP TABLE algorithms; --")

	f.Fuzz(func(t *testing.T, algoName string) {
		// GetAlgorithm should never panic
		_ = GetAlgorithm(algoName)

		// VerifyAlgorithm with unknown algorithm should return error, not panic
		prov, _ := NewInMemoryEd25519Provider()
		err := VerifyAlgorithm(algoName, []byte("msg"), "c2ln", "kid", prov)
		if algoName != AlgoEd25519 && algoName != AlgoECDSAP256 && algoName != AlgoBLS12381 {
			if err == nil {
				t.Fatalf("unknown algorithm should return error")
			}
		}
	})
}

// Fuzz test for base64 decoding edge cases in signature verification
func FuzzBase64DecodingInVerification(f *testing.F) {
	f.Add("VmFsaWRCYXNlNjQ=")
	f.Add("!!!invalid!!!")
	f.Add("")
	f.Add("SGVsbG8gV29ybGQ=")
	f.Add("padding==")

	f.Fuzz(func(t *testing.T, b64Input string) {
		prov, err := NewInMemoryEd25519Provider()
		if err != nil {
			t.Skip()
		}
		signer, _ := prov.ActiveSigner()

		// Should handle invalid base64 gracefully (return error, not panic)
		_ = VerifyAlgorithm(AlgoEd25519, []byte("msg"), b64Input, signer.KeyID(), prov)
	})
}

// Fuzz test for key ID lookup edge cases
func FuzzKeyIDLookup(f *testing.F) {
	f.Add("validkey123")
	f.Add("")
	f.Add("unknown")
	f.Add("../../etc/passwd")
	f.Add(string(make([]byte, 10000))) // very long

	f.Fuzz(func(t *testing.T, keyID string) {
		prov, err := NewInMemoryEd25519Provider()
		if err != nil {
			t.Skip()
		}

		// PublicKey lookup should not panic
		_, _, _ = prov.PublicKey(keyID)

		// VerifyWith should not panic
		_ = prov.VerifyWith([]byte("msg"), []byte("sig"), keyID)
	})
}

// Fuzz test for signature length edge cases
func FuzzSignatureLengthVariations(f *testing.F) {
	f.Add(make([]byte, 0))   // empty
	f.Add(make([]byte, 64))  // Ed25519 size
	f.Add(make([]byte, 100)) // arbitrary
	f.Add(make([]byte, 1))   // too small

	f.Fuzz(func(t *testing.T, sigBytes []byte) {
		prov, err := NewInMemoryEd25519Provider()
		if err != nil {
			t.Skip()
		}
		signer, _ := prov.ActiveSigner()

		// Verification with arbitrary length signatures should not panic
		b64 := base64.StdEncoding.EncodeToString(sigBytes)
		_ = VerifyAlgorithm(AlgoEd25519, []byte("msg"), b64, signer.KeyID(), prov)
	})
}
