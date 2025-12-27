package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestEd25519Provider_SignAndVerify tests Ed25519 signature operations.
func TestEd25519Provider_SignAndVerify(t *testing.T) {
	provider := &Ed25519Provider{}
	message := []byte("test message for Ed25519 signature")

	// Generate key pair
	privKey, pubKey, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Verify key types
	if _, ok := privKey.(ed25519.PrivateKey); !ok {
		t.Errorf("Expected ed25519.PrivateKey, got %T", privKey)
	}
	if _, ok := pubKey.(ed25519.PublicKey); !ok {
		t.Errorf("Expected ed25519.PublicKey, got %T", pubKey)
	}

	// Sign message
	signature, err := provider.Sign(privKey, message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify signature
	err = provider.Verify(pubKey, message, signature)
	if err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	// Verify with wrong message should fail
	err = provider.Verify(pubKey, []byte("different message"), signature)
	if err == nil {
		t.Error("Expected verification to fail with wrong message")
	}

	// Verify with corrupted signature should fail
	corruptedSig := make([]byte, len(signature))
	copy(corruptedSig, signature)
	corruptedSig[0] ^= 0xFF
	err = provider.Verify(pubKey, message, corruptedSig)
	if err == nil {
		t.Error("Expected verification to fail with corrupted signature")
	}
}

// TestEd25519Provider_KeyMarshaling tests PEM encoding/decoding for Ed25519 keys.
func TestEd25519Provider_KeyMarshaling(t *testing.T) {
	provider := &Ed25519Provider{}

	// Generate key pair
	privKey, pubKey, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Marshal private key
	privPEM, err := provider.MarshalPrivateKey(privKey)
	if err != nil {
		t.Fatalf("MarshalPrivateKey failed: %v", err)
	}
	if !strings.Contains(string(privPEM), "BEGIN PRIVATE KEY") {
		t.Error("Private key PEM missing header")
	}

	// Unmarshal private key
	recoveredPriv, err := provider.UnmarshalPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("UnmarshalPrivateKey failed: %v", err)
	}

	// Marshal public key
	pubPEM, err := provider.MarshalPublicKey(pubKey)
	if err != nil {
		t.Fatalf("MarshalPublicKey failed: %v", err)
	}
	if !strings.Contains(string(pubPEM), "BEGIN PUBLIC KEY") {
		t.Error("Public key PEM missing header")
	}

	// Unmarshal public key
	recoveredPub, err := provider.UnmarshalPublicKey(pubPEM)
	if err != nil {
		t.Fatalf("UnmarshalPublicKey failed: %v", err)
	}

	// Test sign/verify with recovered keys
	message := []byte("test message")
	signature, err := provider.Sign(recoveredPriv, message)
	if err != nil {
		t.Fatalf("Sign with recovered key failed: %v", err)
	}
	err = provider.Verify(recoveredPub, message, signature)
	if err != nil {
		t.Errorf("Verify with recovered key failed: %v", err)
	}
}

// TestEd25519Provider_Metadata tests metadata methods.
func TestEd25519Provider_Metadata(t *testing.T) {
	provider := &Ed25519Provider{}

	if provider.KeyType() != "ed25519.PrivateKey" {
		t.Errorf("Expected KeyType 'ed25519.PrivateKey', got '%s'", provider.KeyType())
	}

	if provider.AlgorithmID() != AlgorithmEd25519 {
		t.Errorf("Expected AlgorithmID '%s', got '%s'", AlgorithmEd25519, provider.AlgorithmID())
	}
}

// TestRSAPSSProvider_SignAndVerify tests RSA-PSS signature operations.
func TestRSAPSSProvider_SignAndVerify(t *testing.T) {
	provider := NewRSAPSSProvider(2048)
	message := []byte("test message for RSA-PSS signature")

	// Generate key pair
	privKey, pubKey, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Verify key types
	if _, ok := privKey.(*rsa.PrivateKey); !ok {
		t.Errorf("Expected *rsa.PrivateKey, got %T", privKey)
	}
	if _, ok := pubKey.(*rsa.PublicKey); !ok {
		t.Errorf("Expected *rsa.PublicKey, got %T", pubKey)
	}

	// Sign message
	signature, err := provider.Sign(privKey, message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify signature
	err = provider.Verify(pubKey, message, signature)
	if err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	// Verify with wrong message should fail
	err = provider.Verify(pubKey, []byte("different message"), signature)
	if err == nil {
		t.Error("Expected verification to fail with wrong message")
	}

	// Verify with corrupted signature should fail
	corruptedSig := make([]byte, len(signature))
	copy(corruptedSig, signature)
	corruptedSig[0] ^= 0xFF
	err = provider.Verify(pubKey, message, corruptedSig)
	if err == nil {
		t.Error("Expected verification to fail with corrupted signature")
	}
}

// TestRSAPSSProvider_KeySizes tests RSA-PSS with different key sizes.
func TestRSAPSSProvider_KeySizes(t *testing.T) {
	keySizes := []int{2048, 3072, 4096}
	message := []byte("test message")

	for _, size := range keySizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			provider := NewRSAPSSProvider(size)
			privKey, pubKey, err := provider.GenerateKey()
			if err != nil {
				t.Fatalf("GenerateKey(%d) failed: %v", size, err)
			}

			rsaPriv := privKey.(*rsa.PrivateKey)
			if rsaPriv.N.BitLen() != size {
				t.Errorf("Expected key size %d, got %d", size, rsaPriv.N.BitLen())
			}

			signature, err := provider.Sign(privKey, message)
			if err != nil {
				t.Fatalf("Sign failed: %v", err)
			}

			err = provider.Verify(pubKey, message, signature)
			if err != nil {
				t.Errorf("Verify failed: %v", err)
			}
		})
	}
}

// TestRSAPSSProvider_MinimumKeySize tests that key sizes below 2048 are enforced to 2048.
func TestRSAPSSProvider_MinimumKeySize(t *testing.T) {
	provider := NewRSAPSSProvider(1024) // Below minimum
	if provider.keySize != 2048 {
		t.Errorf("Expected minimum key size 2048, got %d", provider.keySize)
	}
}

// TestRSAPSSProvider_KeyMarshaling tests PEM encoding/decoding for RSA keys.
func TestRSAPSSProvider_KeyMarshaling(t *testing.T) {
	provider := NewRSAPSSProvider(2048)

	// Generate key pair
	privKey, pubKey, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Marshal private key
	privPEM, err := provider.MarshalPrivateKey(privKey)
	if err != nil {
		t.Fatalf("MarshalPrivateKey failed: %v", err)
	}

	// Unmarshal private key
	recoveredPriv, err := provider.UnmarshalPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("UnmarshalPrivateKey failed: %v", err)
	}

	// Marshal public key
	pubPEM, err := provider.MarshalPublicKey(pubKey)
	if err != nil {
		t.Fatalf("MarshalPublicKey failed: %v", err)
	}

	// Unmarshal public key
	recoveredPub, err := provider.UnmarshalPublicKey(pubPEM)
	if err != nil {
		t.Fatalf("UnmarshalPublicKey failed: %v", err)
	}

	// Test sign/verify with recovered keys
	message := []byte("test message")
	signature, err := provider.Sign(recoveredPriv, message)
	if err != nil {
		t.Fatalf("Sign with recovered key failed: %v", err)
	}
	err = provider.Verify(recoveredPub, message, signature)
	if err != nil {
		t.Errorf("Verify with recovered key failed: %v", err)
	}
}

// TestRSAPSSProvider_Metadata tests metadata methods.
func TestRSAPSSProvider_Metadata(t *testing.T) {
	provider := NewRSAPSSProvider(2048)

	if provider.KeyType() != "*rsa.PrivateKey" {
		t.Errorf("Expected KeyType '*rsa.PrivateKey', got '%s'", provider.KeyType())
	}

	if provider.AlgorithmID() != AlgorithmRSAPSS {
		t.Errorf("Expected AlgorithmID '%s', got '%s'", AlgorithmRSAPSS, provider.AlgorithmID())
	}
}

// TestECDSAP256Provider_SignAndVerify tests ECDSA P-256 signature operations.
func TestECDSAP256Provider_SignAndVerify(t *testing.T) {
	provider := &ECDSAP256Provider{}
	message := []byte("test message for ECDSA P-256 signature")

	// Generate key pair
	privKey, pubKey, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Verify key types
	ecdsaPriv, ok := privKey.(*ecdsa.PrivateKey)
	if !ok {
		t.Errorf("Expected *ecdsa.PrivateKey, got %T", privKey)
	}
	if _, ok := pubKey.(*ecdsa.PublicKey); !ok {
		t.Errorf("Expected *ecdsa.PublicKey, got %T", pubKey)
	}

	// Verify P-256 curve
	if ecdsaPriv.Curve.Params().Name != "P-256" {
		t.Errorf("Expected P-256 curve, got %s", ecdsaPriv.Curve.Params().Name)
	}

	// Sign message
	signature, err := provider.Sign(privKey, message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify signature
	err = provider.Verify(pubKey, message, signature)
	if err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	// Verify with wrong message should fail
	err = provider.Verify(pubKey, []byte("different message"), signature)
	if err == nil {
		t.Error("Expected verification to fail with wrong message")
	}

	// Verify with corrupted signature should fail
	corruptedSig := make([]byte, len(signature))
	copy(corruptedSig, signature)
	corruptedSig[0] ^= 0xFF
	err = provider.Verify(pubKey, message, corruptedSig)
	if err == nil {
		t.Error("Expected verification to fail with corrupted signature")
	}
}

// TestECDSAP256Provider_KeyMarshaling tests PEM encoding/decoding for ECDSA keys.
func TestECDSAP256Provider_KeyMarshaling(t *testing.T) {
	provider := &ECDSAP256Provider{}

	// Generate key pair
	privKey, pubKey, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Marshal private key
	privPEM, err := provider.MarshalPrivateKey(privKey)
	if err != nil {
		t.Fatalf("MarshalPrivateKey failed: %v", err)
	}

	// Unmarshal private key
	recoveredPriv, err := provider.UnmarshalPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("UnmarshalPrivateKey failed: %v", err)
	}

	// Marshal public key
	pubPEM, err := provider.MarshalPublicKey(pubKey)
	if err != nil {
		t.Fatalf("MarshalPublicKey failed: %v", err)
	}

	// Unmarshal public key
	recoveredPub, err := provider.UnmarshalPublicKey(pubPEM)
	if err != nil {
		t.Fatalf("UnmarshalPublicKey failed: %v", err)
	}

	// Test sign/verify with recovered keys
	message := []byte("test message")
	signature, err := provider.Sign(recoveredPriv, message)
	if err != nil {
		t.Fatalf("Sign with recovered key failed: %v", err)
	}
	err = provider.Verify(recoveredPub, message, signature)
	if err != nil {
		t.Errorf("Verify with recovered key failed: %v", err)
	}
}

// TestECDSAP256Provider_Metadata tests metadata methods.
func TestECDSAP256Provider_Metadata(t *testing.T) {
	provider := &ECDSAP256Provider{}

	if provider.KeyType() != "*ecdsa.PrivateKey" {
		t.Errorf("Expected KeyType '*ecdsa.PrivateKey', got '%s'", provider.KeyType())
	}

	if provider.AlgorithmID() != AlgorithmECDSAP256 {
		t.Errorf("Expected AlgorithmID '%s', got '%s'", AlgorithmECDSAP256, provider.AlgorithmID())
	}
}

// TestCrossAlgorithmVerification tests that signatures from one algorithm cannot be verified by another.
func TestCrossAlgorithmVerification(t *testing.T) {
	message := []byte("cross-algorithm test message")

	// Generate keys for all algorithms
	ed25519Provider := &Ed25519Provider{}
	ed25519Priv, ed25519Pub, _ := ed25519Provider.GenerateKey()

	rsaProvider := NewRSAPSSProvider(2048)
	rsaPriv, rsaPub, _ := rsaProvider.GenerateKey()

	ecdsaProvider := &ECDSAP256Provider{}
	ecdsaPriv, ecdsaPub, _ := ecdsaProvider.GenerateKey()

	// Generate signatures
	ed25519Sig, _ := ed25519Provider.Sign(ed25519Priv, message)
	rsaSig, _ := rsaProvider.Sign(rsaPriv, message)
	ecdsaSig, _ := ecdsaProvider.Sign(ecdsaPriv, message)

	// Test cross-verification (should all fail)
	tests := []struct {
		name      string
		provider  SignatureAlgorithm
		pubKey    interface{}
		signature []byte
	}{
		{"Ed25519 sig with RSA key", rsaProvider, rsaPub, ed25519Sig},
		{"Ed25519 sig with ECDSA key", ecdsaProvider, ecdsaPub, ed25519Sig},
		{"RSA sig with Ed25519 key", ed25519Provider, ed25519Pub, rsaSig},
		{"RSA sig with ECDSA key", ecdsaProvider, ecdsaPub, rsaSig},
		{"ECDSA sig with Ed25519 key", ed25519Provider, ed25519Pub, ecdsaSig},
		{"ECDSA sig with RSA key", rsaProvider, rsaPub, ecdsaSig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.Verify(tt.pubKey, message, tt.signature)
			if err == nil {
				t.Error("Expected cross-algorithm verification to fail")
			}
		})
	}
}

// TestAlgorithmRegistry_BasicOperations tests registry CRUD operations.
func TestAlgorithmRegistry_BasicOperations(t *testing.T) {
	registry := NewAlgorithmRegistry()

	// Check default algorithms are registered
	algorithms := registry.ListAlgorithms()
	if len(algorithms) != 4 {
		t.Errorf("Expected 4 default algorithms, got %d", len(algorithms))
	}

	// Get each default algorithm
	for _, algID := range []string{AlgorithmEd25519, AlgorithmRSAPSS, AlgorithmECDSAP256, AlgorithmBLS} {
		provider, err := registry.Get(algID)
		if err != nil {
			t.Errorf("Failed to get algorithm '%s': %v", algID, err)
		}
		if provider.AlgorithmID() != algID {
			t.Errorf("Expected algorithm ID '%s', got '%s'", algID, provider.AlgorithmID())
		}
	}

	// Try to get non-existent algorithm
	_, err := registry.Get("UNKNOWN_ALGO")
	if err == nil {
		t.Error("Expected error when getting unknown algorithm")
	}
}

// TestAlgorithmRegistry_Registration tests custom algorithm registration.
func TestAlgorithmRegistry_Registration(t *testing.T) {
	registry := NewAlgorithmRegistry()

	// Register custom RSA-PSS provider with 4096-bit keys
	customRSA := NewRSAPSSProvider(4096)
	registry.Register("PS256-4096", customRSA)

	// Retrieve and verify
	provider, err := registry.Get("PS256-4096")
	if err != nil {
		t.Fatalf("Failed to get custom algorithm: %v", err)
	}

	if provider.KeyType() != "*rsa.PrivateKey" {
		t.Errorf("Expected RSA provider, got KeyType '%s'", provider.KeyType())
	}

	// Generate key and verify size
	privKey, _, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	rsaPriv := privKey.(*rsa.PrivateKey)
	if rsaPriv.N.BitLen() != 4096 {
		t.Errorf("Expected 4096-bit key, got %d bits", rsaPriv.N.BitLen())
	}
}

// TestAlgorithmRegistry_ConcurrentAccess tests thread-safe operations.
func TestAlgorithmRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewAlgorithmRegistry()
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := registry.Get(AlgorithmEd25519)
				if err != nil {
					t.Errorf("Concurrent Get failed: %v", err)
				}
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				customID := string(rune('A' + id))
				registry.Register(customID, &Ed25519Provider{})
			}
		}(i)
	}

	wg.Wait()
}

// TestAlgorithmRegistry_DefaultRegistry tests the global default registry.
func TestAlgorithmRegistry_DefaultRegistry(t *testing.T) {
	// Verify default algorithms are available
	for _, algID := range []string{AlgorithmEd25519, AlgorithmRSAPSS, AlgorithmECDSAP256, AlgorithmBLS} {
		provider, err := DefaultRegistry.Get(algID)
		if err != nil {
			t.Errorf("Failed to get algorithm '%s' from DefaultRegistry: %v", algID, err)
		}
		if provider.AlgorithmID() != algID {
			t.Errorf("Expected algorithm ID '%s', got '%s'", algID, provider.AlgorithmID())
		}
	}
}

// TestInvalidKeyTypes tests error handling for invalid key types.
func TestInvalidKeyTypes(t *testing.T) {
	message := []byte("test message")

	tests := []struct {
		name     string
		provider SignatureAlgorithm
		wrongKey interface{}
	}{
		{"Ed25519 with RSA key", &Ed25519Provider{}, &rsa.PrivateKey{}},
		{"RSA with Ed25519 key", NewRSAPSSProvider(2048), ed25519.PrivateKey{}},
		{"ECDSA with Ed25519 key", &ECDSAP256Provider{}, ed25519.PrivateKey{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.provider.Sign(tt.wrongKey, message)
			if err == nil {
				t.Error("Expected error when signing with wrong key type")
			}
			if !strings.Contains(err.Error(), "invalid key type") {
				t.Errorf("Expected 'invalid key type' error, got: %v", err)
			}
		})
	}
}

// TestInvalidPEMData tests error handling for malformed PEM data.
func TestInvalidPEMData(t *testing.T) {
	providers := []SignatureAlgorithm{
		&Ed25519Provider{},
		NewRSAPSSProvider(2048),
		&ECDSAP256Provider{},
	}

	invalidPEM := []byte("not a valid PEM block")

	for _, provider := range providers {
		t.Run(provider.AlgorithmID(), func(t *testing.T) {
			_, err := provider.UnmarshalPrivateKey(invalidPEM)
			if err == nil {
				t.Error("Expected error when unmarshaling invalid PEM")
			}

			_, err = provider.UnmarshalPublicKey(invalidPEM)
			if err == nil {
				t.Error("Expected error when unmarshaling invalid PEM")
			}
		})
	}
}

// TestSignatureConsistency tests that multiple signatures of the same message verify correctly.
func TestSignatureConsistency(t *testing.T) {
	providers := []SignatureAlgorithm{
		&Ed25519Provider{},
		NewRSAPSSProvider(2048),
		&ECDSAP256Provider{},
	}

	message := []byte("consistency test message")

	for _, provider := range providers {
		t.Run(provider.AlgorithmID(), func(t *testing.T) {
			privKey, pubKey, err := provider.GenerateKey()
			if err != nil {
				t.Fatalf("GenerateKey failed: %v", err)
			}

			// Generate multiple signatures
			for i := 0; i < 10; i++ {
				signature, err := provider.Sign(privKey, message)
				if err != nil {
					t.Fatalf("Sign iteration %d failed: %v", i, err)
				}

				err = provider.Verify(pubKey, message, signature)
				if err != nil {
					t.Errorf("Verify iteration %d failed: %v", i, err)
				}
			}
		})
	}
}

// TestVerifyBatch tests batch verification logic for all providers.
func TestVerifyBatch(t *testing.T) {
	providers := []SignatureAlgorithm{
		&Ed25519Provider{},
		NewRSAPSSProvider(2048),
		&ECDSAP256Provider{},
	}

	for _, provider := range providers {
		t.Run(provider.AlgorithmID(), func(t *testing.T) {
			count := 5
			pubKeys := make([]interface{}, count)
			messages := make([][]byte, count)
			signatures := make([][]byte, count)

			for i := 0; i < count; i++ {
				priv, pub, err := provider.GenerateKey()
				if err != nil {
					t.Fatalf("GenerateKey failed: %v", err)
				}
				pubKeys[i] = pub
				messages[i] = []byte(fmt.Sprintf("message-%d", i))
				sig, err := provider.Sign(priv, messages[i])
				if err != nil {
					t.Fatalf("Sign failed: %v", err)
				}
				signatures[i] = sig
			}

			// 1. Valid Batch
			if err := provider.VerifyBatch(pubKeys, messages, signatures); err != nil {
				t.Errorf("Valid batch failed: %v", err)
			}

			// 2. Corrupted Signature
			corruptedSigs := make([][]byte, count)
			copy(corruptedSigs, signatures)
			// Copy the signature slice to avoid mutating original
			corruptedSig := make([]byte, len(signatures[0]))
			copy(corruptedSig, signatures[0])
			corruptedSig[0] ^= 0xFF
			corruptedSigs[0] = corruptedSig

			if err := provider.VerifyBatch(pubKeys, messages, corruptedSigs); err == nil {
				t.Error("Expected failure with corrupted signature")
			}

			// 3. Length Mismatch
			if err := provider.VerifyBatch(pubKeys[:count-1], messages, signatures); err == nil {
				t.Error("Expected failure with mismatching public keys count")
			}
		})
	}
}
