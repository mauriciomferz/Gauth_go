package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestShamirSplitReconstructEd25519Seed(t *testing.T) {
    // Generate Ed25519 key
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatalf("gen ed25519: %v", err) }
    seed := priv.Seed()
    n, threshold := 5, 3
    shares, err := SplitSecret(seed, n, threshold, rand.Reader)
    if err != nil { t.Fatalf("split: %v", err) }
    // Reconstruct using first threshold shares
    recovered, err := Reconstruct(shares[:threshold], threshold)
    if err != nil { t.Fatalf("reconstruct: %v", err) }
    if len(recovered) != len(seed) { t.Fatalf("length mismatch") }
    for i := range seed { if seed[i] != recovered[i] { t.Fatalf("seed mismatch at %d", i) } }
    // Threshold signing
    msg := []byte("threshold-message")
    sig, err := ThresholdSignEd25519(shares[:threshold], threshold, pub, msg)
    if err != nil { t.Fatalf("threshold sign: %v", err) }
    if !ed25519.Verify(pub, msg, sig) { t.Fatalf("signature verify failed") }
}

func TestShamirCorruptedShare(t *testing.T) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatalf("gen: %v", err) }
    seed := priv.Seed()
    n, threshold := 4, 3
    shares, err := SplitSecret(seed, n, threshold, rand.Reader)
    if err != nil { t.Fatalf("split: %v", err) }
    // Corrupt one share used in reconstruction
    shares[1].Value[0] ^= 0xFF
    _, err = ThresholdSignEd25519(shares[:threshold], threshold, pub, []byte("msg"))
    if err == nil { t.Fatalf("expected error due to corrupted share") }
}

func TestShamirInsufficientShares(t *testing.T) {
    _, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatalf("gen: %v", err) }
    seed := priv.Seed()
    n, threshold := 5, 4
    shares, err := SplitSecret(seed, n, threshold, rand.Reader)
    if err != nil { t.Fatalf("split: %v", err) }
    if _, err := Reconstruct(shares[:threshold-1], threshold); err == nil { t.Fatalf("expected insufficient shares error") }
}

func TestShamirDuplicateIndex(t *testing.T) {
    _, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatalf("gen: %v", err) }
    seed := priv.Seed()
    shares, err := SplitSecret(seed, 3, 2, rand.Reader)
    if err != nil { t.Fatalf("split: %v", err) }
    // Duplicate index
    dup := shares[1]
    dup.Index = shares[0].Index
    _, err = Reconstruct([]Share{shares[0], dup}, 2)
    if err == nil { t.Fatalf("expected duplicate index error") }
}
