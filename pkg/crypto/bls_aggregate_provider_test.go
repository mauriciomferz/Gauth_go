package crypto

import (
	"encoding/base64"
	"testing"
)

// TestAggregatedBLS_SignAndVerify ensures aggregated verification succeeds for multiple participants.
func TestAggregatedBLS_SignAndVerify(t *testing.T) {
	prov := newMultiBLSKeyProvider()
	keyIDs, privs, err := prov.GenerateBLSKeys(3)
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}
	msg := []byte("canonical-shared-message")
	aggSig, err := AggregateSign(msg, privs)
	if err != nil {
		t.Fatalf("aggregate sign: %v", err)
	}
	aggB64 := base64.StdEncoding.EncodeToString(aggSig)
	// Verify using aggregated algorithm via registry
	if err := VerifyAggregatedAlgorithm(AlgoBLS12381Agg, [][]byte{msg}, []string{aggB64}, keyIDs, prov); err != nil {
		t.Fatalf("aggregated verify failed: %v", err)
	}
}

// TestAggregatedBLS_Tamper flips a byte in the aggregated signature and expects failure.
func TestAggregatedBLS_Tamper(t *testing.T) {
	prov := newMultiBLSKeyProvider()
	keyIDs, privs, _ := prov.GenerateBLSKeys(2)
	msg := []byte("same-message")
	aggSig, _ := AggregateSign(msg, privs)
	aggSig[0] ^= 0xFF // tamper
	aggB64 := base64.StdEncoding.EncodeToString(aggSig)
	if err := VerifyAggregatedAlgorithm(AlgoBLS12381Agg, [][]byte{msg}, []string{aggB64}, keyIDs, prov); err == nil {
		t.Fatalf("expected failure on tampered aggregated signature")
	}
}

// TestAggregatedBLS_StructuralErrors validates structural contract enforcement.
func TestAggregatedBLS_StructuralErrors(t *testing.T) {
	prov := newMultiBLSKeyProvider()
	keyIDs, privs, _ := prov.GenerateBLSKeys(1)
	msg := []byte("m")
	aggSig, _ := AggregateSign(msg, privs)
	b64 := base64.StdEncoding.EncodeToString(aggSig)
	// messages count != 1
	if err := VerifyAggregatedAlgorithm(AlgoBLS12381Agg, [][]byte{msg, msg}, []string{b64}, keyIDs, prov); err == nil {
		t.Fatalf("expected error for >1 messages")
	}
	// signatures count != 1
	if err := VerifyAggregatedAlgorithm(AlgoBLS12381Agg, [][]byte{msg}, []string{b64, b64}, keyIDs, prov); err == nil {
		t.Fatalf("expected error for >1 signatures")
	}
	// empty keyIDs
	if err := VerifyAggregatedAlgorithm(AlgoBLS12381Agg, [][]byte{msg}, []string{b64}, []string{}, prov); err == nil {
		t.Fatalf("expected error for 0 keyIDs")
	}
}

// Ensure library initialization path works for repeated runs.
func TestAggregatedBLS_ReinitSafety(t *testing.T) {
	prov := newMultiBLSKeyProvider()
	for i := 0; i < 2; i++ { // multiple initializations
		_, privs, err := prov.GenerateBLSKeys(1)
		if err != nil {
			t.Fatalf("generate keys iter %d: %v", i, err)
		}
		msg := []byte("x")
		aggSig, err := AggregateSign(msg, privs)
		if err != nil {
			t.Fatalf("aggregate sign: %v", err)
		}
		if len(aggSig) == 0 {
			t.Fatalf("empty aggregated signature")
		}
	}
}

// BenchmarkAggregatedBLS provides rough sizing (optional manual run).
func BenchmarkAggregatedBLS(b *testing.B) {
	prov := newMultiBLSKeyProvider()
	keyCount := 5
	keyIDs, privs, err := prov.GenerateBLSKeys(keyCount)
	if err != nil {
		b.Fatalf("generate keys: %v", err)
	}
	msg := []byte("benchmark-message")
	for i := 0; i < b.N; i++ {
		aggSig, err := AggregateSign(msg, privs)
		if err != nil {
			b.Fatalf("aggregate sign: %v", err)
		}
		aggB64 := base64.StdEncoding.EncodeToString(aggSig)
		if err := VerifyAggregatedAlgorithm(AlgoBLS12381Agg, [][]byte{msg}, []string{aggB64}, keyIDs, prov); err != nil {
			b.Fatalf("verify failed: %v", err)
		}
	}
}
