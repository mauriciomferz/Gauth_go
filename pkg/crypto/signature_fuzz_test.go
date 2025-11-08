package crypto

import (
	"math/rand"
	"testing"
)

func FuzzSignerVerifier_Ed25519(f *testing.F) {
	prov, _ := NewInMemoryEd25519Provider()
	signer, _ := prov.ActiveSigner()
	f.Add([]byte("fuzz-message"))
	f.Fuzz(func(t *testing.T, msg []byte) {
		sig, err := signer.Sign(msg)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if err := prov.VerifyWith(msg, sig, signer.KeyID()); err != nil {
			t.Fatalf("verify failed: %v", err)
		}
		// Tamper with signature
		if len(sig) > 0 {
			//nolint:gosec // G404: weak random acceptable for fuzz test mutation
			sig[0] ^= byte(rand.Intn(255))
			if err := prov.VerifyWith(msg, sig, signer.KeyID()); err == nil {
				t.Fatalf("expected verification failure with tampered signature")
			}
		}
	})
}
