package crypto

import "testing"

func TestBatchVerifyEd25519(t *testing.T) {
	pubs := []interface{}{}
	msgs := [][]byte{}
	sigs := [][]byte{}
	if BatchVerifyEd25519(pubs, msgs, sigs) {
		t.Fatalf("BatchVerifyEd25519 should return false for empty input")
	}
}

func TestBatchVerifyECDSA(t *testing.T) {
	pubs := []interface{}{}
	msgs := [][]byte{}
	sigs := [][]byte{}
	if BatchVerifyECDSA(pubs, msgs, sigs) {
		t.Fatalf("BatchVerifyECDSA should return false for empty input")
	}
}

func TestBatchVerifyBLS(t *testing.T) {
	pubs := []interface{}{}
	msgs := [][]byte{}
	sigs := [][]byte{}
	if BatchVerifyBLS(pubs, msgs, sigs) {
		t.Fatalf("BatchVerifyBLS should return false for empty input")
	}
}
