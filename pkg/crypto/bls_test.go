package crypto

import (
	"testing"

	blslib "github.com/herumi/bls-eth-go-binary/bls"
)

func TestBLSKeyGeneration(t *testing.T) {
	key, err := GenerateBLSKey()
	if err != nil {
		t.Fatalf("BLS key generation failed: %v", err)
	}
	// Check for zero value SecretKey/PublicKey
	if key == nil || len(key.Private.Serialize()) == 0 || len(key.Public.Serialize()) == 0 {
		t.Fatalf("BLS key fields missing")
	}
}

func TestBLSSignAndVerify(t *testing.T) {
	key, _ := GenerateBLSKey()
	msg := []byte("test message")
	sig, err := BLSSign(key, msg)
	if err != nil {
		t.Fatalf("BLS sign failed: %v", err)
	}
	if !BLSVerify(key, msg, sig) {
		t.Fatalf("BLS verify failed")
	}
}

func TestBLSAggregate(t *testing.T) {
	msg := []byte("aggregate message")
	var sigs [][]byte
	var pubs []blslib.PublicKey
	for i := 0; i < 3; i++ {
		k, err := GenerateBLSKey()
		if err != nil {
			t.Fatalf("keygen: %v", err)
		}
		s, err := BLSSign(k, msg)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		sigs = append(sigs, s)
		pubs = append(pubs, k.Public)
	}
	agg, err := BLSAggregate(sigs)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if !BLSAggregateVerify(pubs, msg, agg) {
		t.Fatalf("aggregate verify failed")
	}
}
