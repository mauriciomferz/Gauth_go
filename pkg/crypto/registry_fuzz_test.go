package crypto

import (
	"encoding/base64"
	"math/rand"
	"testing"
)

func FuzzRegistry_Dispatch(f *testing.F) {
	prov, _ := NewInMemoryEd25519Provider()
	signer, _ := prov.ActiveSigner()
	f.Add([]byte("fuzz-registry"))
	f.Fuzz(func(t *testing.T, msg []byte) {
		sig, _ := signer.Sign(msg)
		b64 := base64.StdEncoding.EncodeToString(sig)
		if err := VerifyAlgorithm(AlgoEd25519, msg, b64, signer.KeyID(), prov); err != nil {
			t.Fatalf("verify via registry failed: %v", err)
		}
		// Tamper with signature
		if len(sig) > 0 {
			sig[0] ^= byte(rand.Intn(255))
			b64 = base64.StdEncoding.EncodeToString(sig)
			if err := VerifyAlgorithm(AlgoEd25519, msg, b64, signer.KeyID(), prov); err == nil {
				t.Fatalf("expected error for tampered signature")
			}
		}
	})
}
