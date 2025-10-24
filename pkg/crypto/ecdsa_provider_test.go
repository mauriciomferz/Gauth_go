package crypto

import (
    "encoding/base64"
    "testing"
)

func TestECDSAProvider_SignAndVerify(t *testing.T) {
    prov, err := NewInMemoryECDSAProvider()
    if err != nil { t.Fatalf("provider init: %v", err) }
    signer, err := prov.ActiveSigner()
    if err != nil { t.Fatalf("active signer: %v", err) }
    msg := []byte("canonical-poa-digest-or-json")
    sig, err := signer.Sign(msg)
    if err != nil { t.Fatalf("sign: %v", err) }
    b64 := base64.StdEncoding.EncodeToString(sig)
    if err := VerifyAlgorithm(AlgoECDSAP256, msg, b64, signer.KeyID(), prov); err != nil {
        t.Fatalf("verify ecdsa-p256 failed: %v", err)
    }
}

func TestECDSARegistry_Registered(t *testing.T) {
    if GetAlgorithm(AlgoECDSAP256) == nil { t.Fatalf("expected ecdsa-p256 registered") }
}
