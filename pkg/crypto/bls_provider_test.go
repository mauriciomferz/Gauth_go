package crypto

import (
	"encoding/base64"
	"testing"
)

func TestBLSProvider_SignAndVerify(t *testing.T) {
    prov, err := NewInMemoryBLSProvider()
    if err != nil { t.Fatalf("init bls provider: %v", err) }
    signer, err := prov.ActiveSigner()
    if err != nil { t.Fatalf("active signer: %v", err) }
    msg := []byte("canonical-poa-json")
    sig, err := signer.Sign(msg)
    if err != nil { t.Fatalf("sign: %v", err) }
    b64 := base64.StdEncoding.EncodeToString(sig)
    if err := VerifyAlgorithm(AlgoBLS12381, msg, b64, signer.KeyID(), prov); err != nil {
        t.Fatalf("verify bls12-381 failed: %v", err)
    }
}

func TestBLSProvider_Tamper(t *testing.T) {
    prov, _ := NewInMemoryBLSProvider()
    signer, _ := prov.ActiveSigner()
    msg := []byte("A")
    sig, _ := signer.Sign(msg)
    b64 := base64.StdEncoding.EncodeToString(sig)
    // Tamper message
    if err := VerifyAlgorithm(AlgoBLS12381, []byte("B"), b64, signer.KeyID(), prov); err == nil {
        t.Fatalf("expected verification failure on tampered message")
    }
}
