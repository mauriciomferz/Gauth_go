package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"
)

func TestSignerEd25519RoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	s := NewEd25519Signer(priv, pub)
	msg := []byte("hello-ed25519")
	sig, err := s.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !s.Verify(msg, sig) {
		t.Fatalf("verify failed")
	}
	// Negative: mutate one byte
	sig[0] ^= 0xFF
	if s.Verify(msg, sig) {
		t.Fatalf("verify should fail on mutated signature")
	}
}

func TestSignerECDSARoundTrip(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen ecdsa: %v", err)
	}
	s := NewP256Signer(priv, nil)
	msg := []byte("hello-ecdsa")
	sig, err := s.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !s.Verify(msg, sig) {
		t.Fatalf("verify failed")
	}
	// Decode, re-encode to ensure stability and low-S normalization
	r, vS, derr := decodeDERSignature(sig)
	if derr != nil {
		t.Fatalf("decode der: %v", derr)
	}
	sig2 := encodeDERSignature(r, vS)
	if !s.Verify(msg, sig2) {
		t.Fatalf("verify failed after re-encode")
	}
	// Negative: truncate
	bad := sig[:len(sig)/2]
	if s.Verify(msg, bad) {
		t.Fatalf("verify should fail on truncated sig")
	}
}

// TestSignerECDSAHighSRejection crafts a high-S variant of a valid signature
// and ensures verification rejects the malleable form.
func TestSignerECDSAHighSRejection(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen ecdsa: %v", err)
	}
	s := NewP256Signer(priv, nil)
	msg := []byte("ecdsa-high-s-rejection")
	sig, err := s.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !s.Verify(msg, sig) {
		t.Fatalf("verify failed for canonical low-S signature")
	}
	// Decode canonical (low-S) signature
	r, lowS, derr := decodeDERSignature(sig)
	if derr != nil {
		t.Fatalf("decode der: %v", derr)
	}
	n := priv.Params().N
	// Construct high-S counterpart: n - lowS (will be > n/2 unless lowS == n/2)
	highS := new(big.Int).Sub(n, lowS)
	// Sanity: ensure highS is actually high
	half := new(big.Int).Rsh(n, 1)
	if highS.Cmp(half) != 1 {
		t.Fatalf("constructed highS not greater than n/2")
	}
	highSig := encodeDERSignature(r, highS)
	if s.Verify(msg, highSig) {
		t.Fatalf("verify should reject high-S variant")
	}
}

func TestSignerBLSRoundTrip(t *testing.T) {
	key, err := GenerateBLSKey()
	if err != nil {
		t.Fatalf("gen bls: %v", err)
	}
	signer := NewBLSSigner(key)
	msg := []byte("hello-bls")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !signer.Verify(msg, sig) {
		t.Fatalf("verify failed")
	}
	// Negative: mutate first byte
	sig[0] ^= 0xAA
	if signer.Verify(msg, sig) {
		t.Fatalf("verify should fail after corruption")
	}
}
