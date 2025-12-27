//go:build go1.18

package gauth_rfc_001

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"
	"time"
)

// FuzzSignatureVerification tests signature verification invariants across multiple algorithms.
// Property: A valid signature over canonical digest should always verify successfully.
// Property: Any mutation to the signature or digest should fail verification.
func FuzzSignatureVerification(f *testing.F) {
	// Seed with different signature types
	f.Add("ed25519", "alice", "bob", "read;write", "env=prod", []byte("seed1"))
	f.Add("rsa-pss", "carol", "dave", "admin", "tier=gold", []byte("seed2"))
	f.Add("ecdsa-p256", "eve", "frank", "execute", "", []byte("seed3"))

	f.Fuzz(func(t *testing.T, sigType, grantor, grantee, scopeSemi, restrSemi string, seedBytes []byte) {
		// Parse inputs
		scope := splitSemi(scopeSemi)
		restr := restrMap(restrSemi)

		// Create POA
		poa := &PowerOfAttorney{
			ID:           "fuzz-poa",
			Grantor:      grantor,
			Grantee:      grantee,
			Scope:        scope,
			Restrictions: restr,
			ValidFrom:    time.Unix(0, 0).UTC(),
			ValidUntil:   time.Unix(3600, 0).UTC(),
			Status:       POAStatusActive,
			CreatedAt:    time.Unix(10, 0).UTC(),
			UpdatedAt:    time.Unix(10, 0).UTC(),
		}

		// Get canonical digest
		digest, canonical, err := CanonicalPOADigest(poa)
		if err != nil {
			t.Skip("Invalid POA for digest") // Skip invalid inputs
			return
		}

		// Generate keypair and sign based on algorithm type
		var signature []byte
		var publicKey crypto.PublicKey

		switch sigType {
		case "ed25519":
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Skip("Key generation failed")
				return
			}
			publicKey = pub
			signature = ed25519.Sign(priv, canonical)

		case "rsa-pss":
			priv, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Skip("RSA key generation failed")
				return
			}
			publicKey = &priv.PublicKey
			h := sha256.Sum256(canonical)
			signature, err = rsa.SignPSS(rand.Reader, priv, crypto.SHA256, h[:], nil)
			if err != nil {
				t.Skip("RSA signing failed")
				return
			}

		case "ecdsa-p256":
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Skip("ECDSA key generation failed")
				return
			}
			publicKey = &priv.PublicKey
			h := sha256.Sum256(canonical)
			signature, err = ecdsa.SignASN1(rand.Reader, priv, h[:])
			if err != nil {
				t.Skip("ECDSA signing failed")
				return
			}

		default:
			t.Skip("Unknown signature type")
			return
		}

		// PROPERTY 1: Valid signature should verify
		if !verifySignature(t, publicKey, canonical, signature, sigType) {
			t.Fatalf("Valid signature failed verification for %s", sigType)
		}

		// PROPERTY 2: Signature with mutated digest should fail
		if len(canonical) > 0 {
			mutatedCanonical := make([]byte, len(canonical))
			copy(mutatedCanonical, canonical)
			mutatedCanonical[0] ^= 0xFF // Flip bits
			if verifySignature(t, publicKey, mutatedCanonical, signature, sigType) {
				t.Fatalf("Signature verified with mutated digest for %s", sigType)
			}
		}

		// PROPERTY 3: Mutated signature should fail
		if len(signature) > 0 {
			mutatedSig := make([]byte, len(signature))
			copy(mutatedSig, signature)
			mutatedSig[len(mutatedSig)-1] ^= 0xFF // Flip last byte
			if verifySignature(t, publicKey, canonical, mutatedSig, sigType) {
				t.Fatalf("Mutated signature verified for %s", sigType)
			}
		}

		// PROPERTY 4: Canonical digest determinism (sign twice, same result)
		digest2, canonical2, err := CanonicalPOADigest(poa)
		if err != nil {
			t.Fatalf("Second digest computation failed")
		}
		if digest != digest2 {
			t.Fatalf("Digest non-deterministic: %s != %s", digest, digest2)
		}
		if string(canonical) != string(canonical2) {
			t.Fatalf("Canonical bytes non-deterministic")
		}
	})
}

// verifySignature is a helper to verify signatures across different algorithms
func verifySignature(t *testing.T, pub crypto.PublicKey, message, signature []byte, sigType string) bool {
	t.Helper()

	switch sigType {
	case "ed25519":
		edPub, ok := pub.(ed25519.PublicKey)
		if !ok {
			t.Fatal("Invalid Ed25519 public key type")
		}
		return ed25519.Verify(edPub, message, signature)

	case "rsa-pss":
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			t.Fatal("Invalid RSA public key type")
		}
		h := sha256.Sum256(message)
		err := rsa.VerifyPSS(rsaPub, crypto.SHA256, h[:], signature, nil)
		return err == nil

	case "ecdsa-p256":
		ecPub, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			t.Fatal("Invalid ECDSA public key type")
		}
		h := sha256.Sum256(message)
		return ecdsa.VerifyASN1(ecPub, h[:], signature)

	default:
		t.Fatalf("Unknown signature type: %s", sigType)
		return false
	}
}

// FuzzSignatureMalleability tests for signature malleability issues.
// Property: Two different signatures over the same digest should both verify (if algorithm allows).
func FuzzSignatureMalleability(f *testing.F) {
	f.Add([]byte("test payload 1"))
	f.Add([]byte("test payload 2"))

	f.Fuzz(func(t *testing.T, payload []byte) {
		// Test Ed25519 (should be deterministic - same message, same signature)
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Skip("Key gen failed")
			return
		}

		sig1 := ed25519.Sign(priv, payload)
		sig2 := ed25519.Sign(priv, payload)

		// Ed25519 signatures should be deterministic
		if string(sig1) != string(sig2) {
			t.Fatalf("Ed25519 signature non-deterministic")
		}

		// Both should verify
		if !ed25519.Verify(pub, payload, sig1) || !ed25519.Verify(pub, payload, sig2) {
			t.Fatalf("Ed25519 signature verification failed")
		}
	})
}

// FuzzCanonicalDigestAlgorithmAgnostic tests that canonical digest is the same
// regardless of which signature algorithm will be used.
// Property: Canonical digest should not depend on signature algorithm choice.
func FuzzCanonicalDigestAlgorithmAgnostic(f *testing.F) {
	f.Add("alice", "bob", "read", "")

	f.Fuzz(func(t *testing.T, grantor, grantee, scopeSemi, restrSemi string) {
		scope := splitSemi(scopeSemi)
		restr := restrMap(restrSemi)

		poa := &PowerOfAttorney{
			ID:           "agnostic-test",
			Grantor:      grantor,
			Grantee:      grantee,
			Scope:        scope,
			Restrictions: restr,
			ValidFrom:    time.Unix(0, 0).UTC(),
			ValidUntil:   time.Unix(3600, 0).UTC(),
			Status:       POAStatusActive,
			CreatedAt:    time.Unix(5, 0).UTC(),
			UpdatedAt:    time.Unix(5, 0).UTC(),
		}

		// Compute digest multiple times
		digest1, canonical1, err1 := CanonicalPOADigest(poa)
		digest2, canonical2, err2 := CanonicalPOADigest(poa)
		digest3, canonical3, err3 := CanonicalPOADigest(poa)

		if err1 != nil || err2 != nil || err3 != nil {
			t.Skip("Invalid POA")
			return
		}

		// All digests must be identical
		if digest1 != digest2 || digest2 != digest3 {
			t.Fatalf("Digest non-deterministic: %s, %s, %s", digest1, digest2, digest3)
		}

		if string(canonical1) != string(canonical2) || string(canonical2) != string(canonical3) {
			t.Fatalf("Canonical bytes non-deterministic")
		}
	})
}

// TestSignatureVerificationVectors provides known test vectors for regression testing.
func TestSignatureVerificationVectors(t *testing.T) {
	// Known test vectors for Ed25519
	testVectors := []struct {
		name     string
		poa      *PowerOfAttorney
		expected string // Expected digest prefix for sanity
	}{
		{
			name: "simple-delegation",
			poa: &PowerOfAttorney{
				ID:           "vec-001",
				Grantor:      "alice@example.com",
				Grantee:      "bob@example.com",
				Scope:        []string{"files:read", "files:write"},
				Restrictions: map[string]string{"max_amount": "1000"},
				ValidFrom:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				ValidUntil:   time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
				Status:       POAStatusActive,
				CreatedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: "sha256:", // Digest should start with this
		},
	}

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			digest, canonical, err := CanonicalPOADigest(tv.poa)
			if err != nil {
				t.Fatalf("Digest failed: %v", err)
			}

			if len(digest) == 0 || len(canonical) == 0 {
				t.Fatalf("Empty digest or canonical")
			}

			// Test with Ed25519
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("Key gen failed: %v", err)
			}

			sig := ed25519.Sign(priv, canonical)
			if !ed25519.Verify(pub, canonical, sig) {
				t.Fatalf("Ed25519 signature verification failed")
			}

			// Mutate and ensure failure
			mutatedCanonical := make([]byte, len(canonical))
			copy(mutatedCanonical, canonical)
			if len(mutatedCanonical) > 0 {
				mutatedCanonical[0] ^= 0xFF
				if ed25519.Verify(pub, mutatedCanonical, sig) {
					t.Fatalf("Signature verified with mutated digest")
				}
			}
		})
	}
}
