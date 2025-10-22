// Package crypto tests cover the in-memory Ed25519 provider registry behaviors (rotation,
// lookup, error cases). They ensure cryptographic manager integration remains stable.
package crypto

import (
	"encoding/base64"
	"testing"
)

func TestRegistry_Ed25519Registered(t *testing.T) {
	if GetAlgorithm(AlgoEd25519) == nil {
		t.Fatalf("expected ed25519 registered")
	}
}

func TestRegistry_Dispatch(t *testing.T) {
	prov, _ := NewInMemoryEd25519Provider()
	signer, _ := prov.ActiveSigner()
	msg := []byte("canonical-bytes")
	sig, _ := signer.Sign(msg)
	b64 := base64.StdEncoding.EncodeToString(sig)
	if err := VerifyAlgorithm(AlgoEd25519, msg, b64, signer.KeyID(), prov); err != nil {
		t.Fatalf("verify via registry failed: %v", err)
	}
}

func TestRegistry_Unknown(t *testing.T) {
	prov, _ := NewInMemoryEd25519Provider()
	signer, _ := prov.ActiveSigner()
	msg := []byte("x")
	sig, _ := signer.Sign(msg)
	b64 := base64.StdEncoding.EncodeToString(sig)
	if err := VerifyAlgorithm("unknown", msg, b64, signer.KeyID(), prov); err == nil {
		t.Fatalf("expected error for unknown algorithm")
	}
}
