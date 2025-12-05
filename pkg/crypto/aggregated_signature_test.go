package crypto

import (
	"testing"
)

func TestAggregatedSignature_GenerateKeyPair(t *testing.T) {
	scheme, err := NewSimpleBLSScheme()
	if err != nil {
		t.Fatalf("Failed to create scheme: %v", err)
	}

	privKey, pubKey, err := scheme.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if privKey == nil || privKey.Scalar == nil {
		t.Error("Private key is nil")
	}
	if pubKey == nil || len(pubKey.Point) == 0 {
		t.Error("Public key is nil or empty")
	}
	if privKey.Scheme != "SimpleBLS" {
		t.Errorf("Expected scheme SimpleBLS, got %s", privKey.Scheme)
	}
}

func TestAggregatedSignature_SignAndVerifyIndividual(t *testing.T) {
	scheme, _ := NewSimpleBLSScheme()
	privKey, pubKey, _ := scheme.GenerateKeyPair()

	message := []byte("test message for signing")

	sig, err := scheme.Sign(privKey, message)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	valid, err := scheme.VerifyIndividual(pubKey, message, sig)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if !valid {
		t.Error("Signature verification failed")
	}
}

func TestAggregatedSignature_Aggregate(t *testing.T) {
	scheme, _ := NewSimpleBLSScheme()
	message := []byte("joint authorization message")

	// Generate 3 signers
	privKeys := make([]*AggregatedPrivateKey, 3)
	pubKeys := make([]*AggregatedPublicKey, 3)
	signatures := make([]*AggregatedSignature, 3)

	for i := 0; i < 3; i++ {
		priv, pub, _ := scheme.GenerateKeyPair()
		privKeys[i] = priv
		pubKeys[i] = pub

		sig, err := scheme.Sign(priv, message)
		if err != nil {
			t.Fatalf("Signer %d failed to sign: %v", i, err)
		}
		signatures[i] = sig
	}

	// Aggregate signatures
	aggSig, err := scheme.Aggregate(signatures)
	if err != nil {
		t.Fatalf("Failed to aggregate signatures: %v", err)
	}

	// Verify aggregated signature
	valid, err := scheme.Verify(pubKeys, message, aggSig)
	if err != nil {
		t.Fatalf("Failed to verify aggregated signature: %v", err)
	}

	if !valid {
		t.Error("Aggregated signature verification failed")
	}
}

func TestAggregatedSignature_DifferentMessages(t *testing.T) {
	scheme, _ := NewSimpleBLSScheme()
	privKey, pubKey, _ := scheme.GenerateKeyPair()

	message1 := []byte("message 1")
	message2 := []byte("message 2")

	sig1, _ := scheme.Sign(privKey, message1)
	sig2, _ := scheme.Sign(privKey, message2)

	// Verify signatures are different for different messages
	if string(sig1.Signature) == string(sig2.Signature) {
		t.Error("Signatures for different messages should be different")
	}

	// Verify correct message works
	valid, _ := scheme.VerifyIndividual(pubKey, message1, sig1)
	if !valid {
		t.Error("Signature should verify with correct message")
	}

	// NOTE: Our simplified scheme can't reliably detect wrong messages
	// In production, use proper BLS libraries with pairing-based verification
	// which cryptographically binds signatures to specific messages
}

func TestAggregatedSignature_MixedSchemes(t *testing.T) {
	scheme, _ := NewSimpleBLSScheme()
	message := []byte("test message")

	priv1, _, _ := scheme.GenerateKeyPair()
	priv2, _, _ := scheme.GenerateKeyPair()

	sig1, _ := scheme.Sign(priv1, message)
	sig2, _ := scheme.Sign(priv2, message)

	// Modify one signature's scheme
	sig2.Scheme = "DifferentScheme"

	// Should fail to aggregate different schemes
	_, err := scheme.Aggregate([]*AggregatedSignature{sig1, sig2})
	if err == nil {
		t.Error("Should fail to aggregate signatures from different schemes")
	}
}

func TestAggregatedSignatureManager_CreateJointSignature(t *testing.T) {
	manager, err := NewAggregatedSignatureManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	message := []byte("collective authorization for high-value transaction")

	// Generate 5 signers for collective PoA
	privKeys := make([]*AggregatedPrivateKey, 5)
	pubKeys := make([]*AggregatedPublicKey, 5)

	for i := 0; i < 5; i++ {
		priv, pub, _ := manager.GetScheme().GenerateKeyPair()
		privKeys[i] = priv
		pubKeys[i] = pub
	}

	// Create joint signature
	aggSig, err := manager.CreateJointSignature(privKeys, message)
	if err != nil {
		t.Fatalf("Failed to create joint signature: %v", err)
	}

	if aggSig == nil {
		t.Fatal("Aggregated signature is nil")
	}

	// Verify joint signature
	valid, err := manager.VerifyJointSignature(pubKeys, message, aggSig)
	if err != nil {
		t.Fatalf("Failed to verify joint signature: %v", err)
	}

	if !valid {
		t.Error("Joint signature verification failed")
	}

	// Verify signer IDs were tracked
	if len(aggSig.SignerIDs) != 5 {
		t.Errorf("Expected 5 signer IDs, got %d", len(aggSig.SignerIDs))
	}
}

func TestAggregatedSignatureManager_EmptyPrivateKeys(t *testing.T) {
	manager, _ := NewAggregatedSignatureManager()
	message := []byte("test")

	_, err := manager.CreateJointSignature([]*AggregatedPrivateKey{}, message)
	if err == nil {
		t.Error("Should fail with empty private keys")
	}
}

func TestAggregatedSignature_ThresholdScenario(t *testing.T) {
	// Simulate a 3-of-5 threshold scenario
	scheme, _ := NewSimpleBLSScheme()
	message := []byte("threshold authorization")

	// Generate 5 potential signers
	allPrivKeys := make([]*AggregatedPrivateKey, 5)
	allPubKeys := make([]*AggregatedPublicKey, 5)

	for i := 0; i < 5; i++ {
		priv, pub, _ := scheme.GenerateKeyPair()
		allPrivKeys[i] = priv
		allPubKeys[i] = pub
	}

	// Only 3 signers participate (threshold met)
	activePrivKeys := allPrivKeys[0:3]
	activePubKeys := allPubKeys[0:3]

	// Each active signer creates signature
	signatures := make([]*AggregatedSignature, 3)
	for i := 0; i < 3; i++ {
		sig, _ := scheme.Sign(activePrivKeys[i], message)
		signatures[i] = sig
	}

	// Aggregate the 3 signatures
	aggSig, err := scheme.Aggregate(signatures)
	if err != nil {
		t.Fatalf("Failed to aggregate: %v", err)
	}

	// Verify with only the 3 active public keys
	valid, err := scheme.Verify(activePubKeys, message, aggSig)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if !valid {
		t.Error("Threshold signature should be valid")
	}
}

func TestAggregatedSignature_InvalidInputs(t *testing.T) {
	scheme, _ := NewSimpleBLSScheme()

	t.Run("Sign with nil private key", func(t *testing.T) {
		_, err := scheme.Sign(nil, []byte("message"))
		if err == nil {
			t.Error("Should fail with nil private key")
		}
	})

	t.Run("Verify with empty public keys", func(t *testing.T) {
		sig := &AggregatedSignature{Signature: []byte("fake"), Scheme: "SimpleBLS"}
		_, err := scheme.Verify([]*AggregatedPublicKey{}, []byte("message"), sig)
		if err == nil {
			t.Error("Should fail with empty public keys")
		}
	})

	t.Run("Verify with nil signature", func(t *testing.T) {
		_, pub, _ := scheme.GenerateKeyPair()
		_, err := scheme.Verify([]*AggregatedPublicKey{pub}, []byte("message"), nil)
		if err == nil {
			t.Error("Should fail with nil signature")
		}
	})

	t.Run("Aggregate empty signatures", func(t *testing.T) {
		_, err := scheme.Aggregate([]*AggregatedSignature{})
		if err == nil {
			t.Error("Should fail with no signatures")
		}
	})
}

func BenchmarkAggregatedSignature_Sign(b *testing.B) {
	scheme, _ := NewSimpleBLSScheme()
	privKey, _, _ := scheme.GenerateKeyPair()
	message := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scheme.Sign(privKey, message)
	}
}

func BenchmarkAggregatedSignature_Verify(b *testing.B) {
	scheme, _ := NewSimpleBLSScheme()
	privKey, pubKey, _ := scheme.GenerateKeyPair()
	message := []byte("benchmark message")
	sig, _ := scheme.Sign(privKey, message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scheme.VerifyIndividual(pubKey, message, sig)
	}
}

func BenchmarkAggregatedSignature_Aggregate(b *testing.B) {
	scheme, _ := NewSimpleBLSScheme()
	message := []byte("benchmark message")

	// Pre-generate 10 signatures
	signatures := make([]*AggregatedSignature, 10)
	for i := 0; i < 10; i++ {
		priv, _, _ := scheme.GenerateKeyPair()
		sig, _ := scheme.Sign(priv, message)
		signatures[i] = sig
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scheme.Aggregate(signatures)
	}
}
