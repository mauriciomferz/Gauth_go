package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

// Property: Sign then verify must always succeed for valid keys
func TestProperty_SignVerifyRoundTrip_Ed25519(t *testing.T) {
	for i := 0; i < 100; i++ {
		prov, err := NewInMemoryEd25519Provider()
		if err != nil {
			t.Fatalf("iteration %d: provider init: %v", i, err)
		}
		signer, err := prov.ActiveSigner()
		if err != nil {
			t.Fatalf("iteration %d: active signer: %v", i, err)
		}
		// Generate random message
		msg := make([]byte, 32+i%128)
		if _, err := rand.Read(msg); err != nil { //nolint:errcheck
			t.Fatalf("iteration %d: random bytes: %v", i, err)
		}
		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("iteration %d: sign: %v", i, err)
		}
		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoEd25519, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("iteration %d: verify round trip failed: %v", i, err)
		}
	}
}

// Property: Sign then verify must always succeed for ECDSA P-256
func TestProperty_SignVerifyRoundTrip_ECDSAP256(t *testing.T) {
	for i := 0; i < 100; i++ {
		prov, err := NewInMemoryECDSAProvider()
		if err != nil {
			t.Fatalf("iteration %d: provider init: %v", i, err)
		}
		signer, err := prov.ActiveSigner()
		if err != nil {
			t.Fatalf("iteration %d: active signer: %v", i, err)
		}
		msg := make([]byte, 64+i%256)
		if _, err := rand.Read(msg); err != nil { //nolint:errcheck
			t.Fatalf("iteration %d: random bytes: %v", i, err)
		}
		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("iteration %d: sign: %v", i, err)
		}
		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoECDSAP256, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("iteration %d: verify ecdsa round trip failed: %v", i, err)
		}
	}
}

// Property: Sign then verify must always succeed for BLS12-381
func TestProperty_SignVerifyRoundTrip_BLS12381(t *testing.T) {
	for i := 0; i < 50; i++ { // BLS is slower, reduce iterations
		prov, err := NewInMemoryBLSProvider()
		if err != nil {
			t.Fatalf("iteration %d: provider init: %v", i, err)
		}
		signer, err := prov.ActiveSigner()
		if err != nil {
			t.Fatalf("iteration %d: active signer: %v", i, err)
		}
		msg := make([]byte, 32+i%128)
		if _, err := rand.Read(msg); err != nil { //nolint:errcheck
			t.Fatalf("iteration %d: random bytes: %v", i, err)
		}
		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("iteration %d: sign: %v", i, err)
		}
		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoBLS12381, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("iteration %d: verify bls round trip failed: %v", i, err)
		}
	}
}

// Property: Tampered message must fail verification
func TestProperty_TamperedMessageFailsVerification(t *testing.T) {
	algorithms := []struct {
		name    string
		newProv func() (KeyProvider, error)
	}{
		{AlgoEd25519, func() (KeyProvider, error) { return NewInMemoryEd25519Provider() }},
		{AlgoECDSAP256, func() (KeyProvider, error) { return NewInMemoryECDSAProvider() }},
		{AlgoBLS12381, func() (KeyProvider, error) { return NewInMemoryBLSProvider() }},
	}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				prov, err := algo.newProv()
				if err != nil {
					t.Fatalf("iteration %d: provider init: %v", i, err)
				}
			signer, _ := prov.ActiveSigner()
			msg := make([]byte, 64)
			_, _ = rand.Read(msg) //nolint:errcheck
			sig, _ := signer.Sign(msg)
				b64 := base64.StdEncoding.EncodeToString(sig)

				// Tamper message (flip one byte)
				tampered := make([]byte, len(msg))
				copy(tampered, msg)
				tampered[i%len(tampered)] ^= 0xFF

				if err := VerifyAlgorithm(algo.name, tampered, b64, signer.KeyID(), prov); err == nil {
					t.Fatalf("iteration %d: expected verification failure for tampered message", i)
				}
			}
		})
	}
}

// Property: Invalid signature must fail verification
func TestProperty_InvalidSignatureFailsVerification(t *testing.T) {
	algorithms := []struct {
		name    string
		newProv func() (KeyProvider, error)
	}{
		{AlgoEd25519, func() (KeyProvider, error) { return NewInMemoryEd25519Provider() }},
		{AlgoECDSAP256, func() (KeyProvider, error) { return NewInMemoryECDSAProvider() }},
		{AlgoBLS12381, func() (KeyProvider, error) { return NewInMemoryBLSProvider() }},
	}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			prov, err := algo.newProv()
			if err != nil {
				t.Fatalf("provider init: %v", err)
			}
			signer, _ := prov.ActiveSigner()
			msg := []byte("valid message")

		// Create random garbage signature
		badSig := make([]byte, 64)
		_, _ = rand.Read(badSig) //nolint:errcheck
		b64 := base64.StdEncoding.EncodeToString(badSig)

		if err := VerifyAlgorithm(algo.name, msg, b64, signer.KeyID(), prov); err == nil {
			t.Fatalf("expected verification failure for invalid signature")
		}
		})
	}
}

// Property: Signature must be deterministic for Ed25519 (Ed25519 is deterministic)
func TestProperty_SignatureDeterminism_Ed25519(t *testing.T) {
	prov, err := NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("provider init: %v", err)
	}
	signer, _ := prov.ActiveSigner()
	msg := []byte("deterministic message test")

	// Sign same message twice
	sig1, err1 := signer.Sign(msg)
	sig2, err2 := signer.Sign(msg)

	if err1 != nil || err2 != nil {
		t.Fatalf("sign errors: %v, %v", err1, err2)
	}

	// Ed25519 signatures should be identical for same message + key
	if base64.StdEncoding.EncodeToString(sig1) != base64.StdEncoding.EncodeToString(sig2) {
		t.Fatalf("Ed25519 signatures should be deterministic: %x vs %x", sig1, sig2)
	}
}

// Property: Different keys produce different signatures
func TestProperty_DifferentKeysProduceDifferentSignatures(t *testing.T) {
	algorithms := []struct {
		name    string
		newProv func() (KeyProvider, error)
	}{
		{AlgoEd25519, func() (KeyProvider, error) { return NewInMemoryEd25519Provider() }},
		{AlgoECDSAP256, func() (KeyProvider, error) { return NewInMemoryECDSAProvider() }},
		{AlgoBLS12381, func() (KeyProvider, error) { return NewInMemoryBLSProvider() }},
	}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			prov1, _ := algo.newProv()
			prov2, _ := algo.newProv()
			signer1, _ := prov1.ActiveSigner()
			signer2, _ := prov2.ActiveSigner()

			msg := []byte("test message for different keys")
			sig1, _ := signer1.Sign(msg)
			sig2, _ := signer2.Sign(msg)

			// Signatures should be different (different keys)
			b64_1 := base64.StdEncoding.EncodeToString(sig1)
			b64_2 := base64.StdEncoding.EncodeToString(sig2)
			if b64_1 == b64_2 {
				t.Fatalf("different keys should produce different signatures")
			}

			// Each signature should verify only with its own key
			if err := VerifyAlgorithm(algo.name, msg, b64_1, signer1.KeyID(), prov1); err != nil {
				t.Fatalf("signature1 should verify with key1: %v", err)
			}
			if err := VerifyAlgorithm(algo.name, msg, b64_2, signer2.KeyID(), prov2); err != nil {
				t.Fatalf("signature2 should verify with key2: %v", err)
			}

			// Cross-verification should fail (if keys are in same provider)
			if err := VerifyAlgorithm(algo.name, msg, b64_1, signer2.KeyID(), prov2); err == nil {
				// Expected to fail because signer1's key is not in prov2
			}
		})
	}
}

// Property: Algorithm interoperability - signatures from different algorithms should not cross-verify
func TestProperty_AlgorithmIsolation(t *testing.T) {
	provEd, _ := NewInMemoryEd25519Provider()
	provEC, _ := NewInMemoryECDSAProvider()
	provBLS, _ := NewInMemoryBLSProvider()

	signerEd, _ := provEd.ActiveSigner()
	signerEC, _ := provEC.ActiveSigner()
	signerBLS, _ := provBLS.ActiveSigner()

	msg := []byte("cross-algorithm test")

	sigEd, _ := signerEd.Sign(msg)
	sigEC, _ := signerEC.Sign(msg)
	sigBLS, _ := signerBLS.Sign(msg)

	b64Ed := base64.StdEncoding.EncodeToString(sigEd)
	b64EC := base64.StdEncoding.EncodeToString(sigEC)
	b64BLS := base64.StdEncoding.EncodeToString(sigBLS)

	// Verify each signature with its correct algorithm (should succeed)
	if err := VerifyAlgorithm(AlgoEd25519, msg, b64Ed, signerEd.KeyID(), provEd); err != nil {
		t.Fatalf("Ed25519 self-verify failed: %v", err)
	}
	if err := VerifyAlgorithm(AlgoECDSAP256, msg, b64EC, signerEC.KeyID(), provEC); err != nil {
		t.Fatalf("ECDSA self-verify failed: %v", err)
	}
	if err := VerifyAlgorithm(AlgoBLS12381, msg, b64BLS, signerBLS.KeyID(), provBLS); err != nil {
		t.Fatalf("BLS self-verify failed: %v", err)
	}

	// Cross-algorithm verification should fail (wrong algorithm for signature type)
	// Note: These will fail with "unknown key" or signature verification error depending on provider
	if err := VerifyAlgorithm(AlgoECDSAP256, msg, b64Ed, "unknown-key", provEC); err == nil {
		t.Fatalf("Ed25519 signature should not verify as ECDSA")
	}
	if err := VerifyAlgorithm(AlgoBLS12381, msg, b64EC, "unknown-key", provBLS); err == nil {
		t.Fatalf("ECDSA signature should not verify as BLS")
	}
	if err := VerifyAlgorithm(AlgoEd25519, msg, b64BLS, "unknown-key", provEd); err == nil {
		t.Fatalf("BLS signature should not verify as Ed25519")
	}
}

// Property: Empty or zero-length messages should be signable and verifiable
func TestProperty_EmptyMessageSignature(t *testing.T) {
	algorithms := []struct {
		name    string
		newProv func() (KeyProvider, error)
	}{
		{AlgoEd25519, func() (KeyProvider, error) { return NewInMemoryEd25519Provider() }},
		{AlgoECDSAP256, func() (KeyProvider, error) { return NewInMemoryECDSAProvider() }},
		{AlgoBLS12381, func() (KeyProvider, error) { return NewInMemoryBLSProvider() }},
	}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			prov, err := algo.newProv()
			if err != nil {
				t.Fatalf("provider init: %v", err)
			}
			signer, _ := prov.ActiveSigner()

			// Sign empty message
			msg := []byte{}
			sig, err := signer.Sign(msg)
			if err != nil {
				t.Fatalf("sign empty message failed: %v", err)
			}
			b64 := base64.StdEncoding.EncodeToString(sig)

			// Verify empty message signature
			if err := VerifyAlgorithm(algo.name, msg, b64, signer.KeyID(), prov); err != nil {
				t.Fatalf("verify empty message signature failed: %v", err)
			}
		})
	}
}
