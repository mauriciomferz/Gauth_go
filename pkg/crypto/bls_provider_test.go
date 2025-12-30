package crypto

import (
	"crypto/rand"
	"fmt"
	"testing"

	bls12381 "github.com/kilic/bls12-381"
)

// Helper to generate a key pair
// SK: Fr (scalar), PK: G2 point
// Sig: G1 point
func generateBLSKey(t *testing.T) (*bls12381.Fr, *bls12381.PointG2) {
	sk := bls12381.NewFr()
	_, err := sk.Rand(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate random secret: %v", err)
	}

	g2 := bls12381.NewG2()
	pk := g2.New()

	// PK = SK * G2_Generator (One)
	// There isn't a direct ScalarBaseMult usually?
	// Use MulScalar(dst, base, scalar)

	g2.MulScalar(pk, g2.One(), sk)
	return sk, pk
}

func signBLS(t *testing.T, sk *bls12381.Fr, msg []byte, domain []byte) *bls12381.PointG1 {
	g1 := bls12381.NewG1()

	// H(m)
	hm, err := g1.HashToCurve(msg, domain)
	if err != nil {
		t.Fatalf("hash to curve failed: %v", err)
	}

	// Sig = SK * H(m)
	sig := g1.New()
	g1.MulScalar(sig, hm, sk)
	return sig
}

func TestBLSProvider_AggregateAndVerify(t *testing.T) {
	domainStr := "AGENTAUTH_BLS_SIG_V1"
	provider := NewBLSProvider(domainStr)
	domain := []byte(domainStr)

	msg := []byte("test-message-for-aggregation")

	// Generate 3 signers
	count := 3
	var sigs [][]byte
	var pubKeys [][]byte

	g1 := bls12381.NewG1()
	g2 := bls12381.NewG2()

	for i := 0; i < count; i++ {
		sk, pk := generateBLSKey(t)
		sigPoint := signBLS(t, sk, msg, domain)

		sigBytes := g1.ToCompressed(sigPoint)
		pkBytes := g2.ToCompressed(pk)

		sigs = append(sigs, sigBytes)
		pubKeys = append(pubKeys, pkBytes)
	}

	// Aggregate
	aggSig, err := provider.Aggregate(sigs)
	if err != nil {
		t.Fatalf("Aggregate failed: %v", err)
	}

	// Verify
	if err := provider.VerifyAggregated(pubKeys, msg, aggSig); err != nil {
		t.Fatalf("VerifyAggregated failed: %v", err)
	}

	// Negative Test: Wrong Message
	if err := provider.VerifyAggregated(pubKeys, []byte("wrong-message"), aggSig); err == nil {
		t.Error("VerifyAggregated succeeded with wrong message")
	}

	if err := provider.VerifyAggregated(pubKeys[:2], msg, aggSig); err == nil {
		t.Error("VerifyAggregated succeeded with missing key subset")
	}
}

func TestBLSProvider_StandardMethods(t *testing.T) {
	provider := NewBLSProvider("")
	msg := []byte("test-standard-methods")

	// Generate
	priv, pub, err := provider.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Sign
	sig, err := provider.Sign(priv, msg)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify
	err = provider.Verify(pub, msg, sig)
	if err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	// Verify wrong message
	if err := provider.Verify(pub, []byte("wrong"), sig); err == nil {
		t.Error("Verify succeeded on wrong message")
	}
}

func TestBLSProvider_VerifyBatch(t *testing.T) {
	provider := NewBLSProvider("")
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
		messages[i] = []byte(fmt.Sprintf("msg-%d", i))
		sig, err := provider.Sign(priv, messages[i])
		if err != nil {
			t.Fatalf("Sign failed: %v", err)
		}
		signatures[i] = sig
	}

	// Batch Verify
	if err := provider.VerifyBatch(pubKeys, messages, signatures); err != nil {
		t.Errorf("VerifyBatch failed: %v", err)
	}
}
