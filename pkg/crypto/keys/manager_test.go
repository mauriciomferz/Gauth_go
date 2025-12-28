package keys

import (
	"context"
	"crypto"
	"crypto/rsa"
	"path/filepath"
	"testing"
)

func TestLocalKeyManager_GenerateAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key.pem")

	// 1. Initialize - should generate new key
	km, err := NewLocalKeyManager(keyPath)
	if err != nil {
		t.Fatalf("First init failed: %v", err)
	}

	ctx := context.Background()

	// Verify key present
	kid, err := km.GetKeyID(ctx)
	if err != nil {
		t.Fatalf("GetKeyID failed: %v", err)
	}
	if kid == "" {
		t.Error("GetKeyID returned empty string")
	}

	pub, err := km.GetPublicKey(ctx)
	if err != nil {
		t.Fatalf("GetPublicKey failed: %v", err)
	}
	if _, ok := pub.(*rsa.PublicKey); !ok {
		t.Error("Public key is not RSA")
	}

	signer, err := km.CryptoSigner(ctx)
	if err != nil {
		t.Fatalf("CryptoSigner failed: %v", err)
	}
	if signer == nil {
		t.Error("Signer is nil")
	}

	// 2. Initialize again - should load existing key
	km2, err := NewLocalKeyManager(keyPath)
	if err != nil {
		t.Fatalf("Second init failed: %v", err)
	}

	kid2, err := km2.GetKeyID(ctx)
	if err != nil {
		t.Fatalf("Second GetKeyID failed: %v", err)
	}

	if kid != kid2 {
		t.Errorf("Key ID changed after reload. Got %s, want %s", kid2, kid)
	}
}

func TestLocalKeyManager_Sign(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "sign_test.pem")
	km, err := NewLocalKeyManager(keyPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ctx := context.Background()
	data := []byte("hello world")

	// Test Sign method (hashed)
	sig, err := km.Sign(ctx, data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify signature
	pub, _ := km.GetPublicKey(ctx)
	rsaPub := pub.(*rsa.PublicKey)
	hashed := crypto.SHA256.New()
	hashed.Write(data)
	digest := hashed.Sum(nil)

	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, digest, sig); err != nil {
		t.Errorf("Signature verification failed: %v", err)
	}
}

func TestPublicJWK(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "jwk_test.pem")
	km, err := NewLocalKeyManager(keyPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	jwk, err := PublicJWK(km)
	if err != nil {
		t.Fatalf("PublicJWK failed: %v", err)
	}

	if jwk["kty"] != "RSA" {
		t.Errorf("kty = %v, want RSA", jwk["kty"])
	}
	if jwk["alg"] != "RS256" {
		t.Errorf("alg = %v, want RS256", jwk["alg"])
	}
	if jwk["kid"] == "" {
		t.Error("kid is empty")
	}
	if jwk["n"] == "" || jwk["e"] == "" {
		t.Error("n or e missing")
	}
}
