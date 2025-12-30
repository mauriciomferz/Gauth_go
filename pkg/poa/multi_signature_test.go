package poa

import (
	"encoding/base64"
	"errors"
	"testing"

	internalCrypto "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// MockKeyProvider implements internalCrypto.KeyProvider for testing
type MockKeyProvider struct {
}

func (m *MockKeyProvider) ActiveSigner() (internalCrypto.Signer, error) {
	return nil, nil // Not needed for verification tests
}

func (m *MockKeyProvider) PublicKey(keyID string) ([]byte, string, error) {
	return nil, "", nil // Not needed for verification tests
}

func (m *MockKeyProvider) VerifyWith(payload, sig []byte, kid string) error {
	// Simple mock behavior:
	// We check if "kid:base64(payload):base64(sig)" key exists in our valid map.
	// In real tests, we might simplify this, e.g. just check if sig == hash(payload)
	// But to test VerifyMultiSig, we just need control over pass/fail.

	// Better mock strategy:
	// The test will setup signatures such that valid ones have sig="valid_sig_for_<kid>"
	// and we check that. Payload is less relevant for the multi-sig counting logic
	// (we trust the crypto provider does its job, we test the aggregator).

	// NOTE: VerifyMultiSig calls VerifyWith(msg, sigBytes, kid).
	// msg is buildPoASigningPayload(p).

	sigStr := string(sig)
	if sigStr == "valid_sig_for_"+kid {
		return nil
	}
	return errors.New("invalid signature")
}

// We need to ensure we implement whatever KeyProvider requires.
// Let's assume VerifyWith and ActiveSigner are the main ones from previous usage.

func TestMultiSignatureVerification_RFC115_C8(t *testing.T) {
	// Mock provider that accepts "valid_sig_for_<kid>"
	kp := &MockKeyProvider{}

	t.Run("Threshold_1_SingleSignature_Valid", func(t *testing.T) {
		p := &ProofOfAuthorization{
			ID:        "poa-1",
			Threshold: 1,
			Signatures: []string{
				base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key1")),
			},
			SignerKids: []string{"key1"},
		}

		validCount, satisfied, threshold := VerifyMultiSig(p, kp)
		if validCount != 1 {
			t.Errorf("Expected 1 valid signature, got %d", validCount)
		}
		if !satisfied {
			t.Errorf("Expected threshold execution satisfied")
		}
		if threshold != 1 {
			t.Errorf("Expected threshold 1, got %d", threshold)
		}
	})

	t.Run("Threshold_2_InsufficientSignatures", func(t *testing.T) {
		p := &ProofOfAuthorization{
			ID:        "poa-2",
			Threshold: 2,
			Signatures: []string{
				base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key1")),
			},
			SignerKids: []string{"key1"},
		}

		validCount, satisfied, _ := VerifyMultiSig(p, kp)
		if validCount != 1 {
			t.Errorf("Expected 1 valid signature, got %d", validCount)
		}
		if satisfied {
			t.Errorf("Expected threshold NOT satisfied (1 < 2)")
		}
	})

	t.Run("Threshold_2_Met", func(t *testing.T) {
		p := &ProofOfAuthorization{
			ID:        "poa-3",
			Threshold: 2,
			Signatures: []string{
				base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key1")),
				base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key2")),
			},
			SignerKids: []string{"key1", "key2"},
		}

		validCount, satisfied, _ := VerifyMultiSig(p, kp)
		if validCount != 2 {
			t.Errorf("Expected 2 valid signatures, got %d", validCount)
		}
		if !satisfied {
			t.Errorf("Expected threshold satisfied (2 >= 2)")
		}
	})

	t.Run("Threshold_2_MixedValidity", func(t *testing.T) {
		p := &ProofOfAuthorization{
			ID:        "poa-4",
			Threshold: 2,
			Signatures: []string{
				base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key1")),
				base64.RawStdEncoding.EncodeToString([]byte("invalid_sig")), // Invalid
				base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key3")),
			},
			SignerKids: []string{"key1", "key2", "key3"},
		}

		validCount, satisfied, _ := VerifyMultiSig(p, kp)
		if validCount != 2 {
			t.Errorf("Expected 2 valid signatures, got %d", validCount)
		}
		if !satisfied {
			t.Errorf("Expected threshold satisfied (2 valid >= 2)")
		}
	})

	t.Run("ZeroThreshold_EdgeCase", func(t *testing.T) {
		// VerifyMultiSig returns 0, false, threshold if logic fails checks
		p := &ProofOfAuthorization{
			ID:         "poa-5",
			Threshold:  0,
			Signatures: []string{base64.RawStdEncoding.EncodeToString([]byte("valid_sig_for_key1"))},
			SignerKids: []string{"key1"},
		}

		_, satisfied, _ := VerifyMultiSig(p, kp)
		if satisfied {
			t.Errorf("Zero threshold should return false (invalid input guard)")
		}
	})
}
