package keys

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalKeyManager(t *testing.T) {
	// 1. Setup Simulated KMS
	kid := "test-key-1"
	simKMS, err := NewSimulatedKMS(kid)
	require.NoError(t, err)

	// 2. Create Manager
	km := NewExternalKeyManager(simKMS)

	ctx := context.Background()

	// 3. Test GetKeyID
	gotKid, err := km.GetKeyID(ctx)
	require.NoError(t, err)
	assert.Equal(t, kid, gotKid)

	// 4. Test GetPublicKey
	pub, err := km.GetPublicKey(ctx)
	require.NoError(t, err)
	rsaPub, ok := pub.(*rsa.PublicKey)
	require.True(t, ok, "Expected RSA public key")

	// 5. Test Sign
	data := []byte("hello world")
	sig, err := km.Sign(ctx, data)
	require.NoError(t, err)
	assert.NotEmpty(t, sig)

	// Verify signature using standard crypto/rsa
	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed[:], sig)
	assert.NoError(t, err, "Signature verification failed")

	// 6. Test CryptoSigner adaptation
	signer, err := km.CryptoSigner(ctx)
	require.NoError(t, err)
	require.NotNil(t, signer)

	assert.Equal(t, pub, signer.Public())

	// Sign using signer (expects digest)
	digest := hashed[:]
	sig2, err := signer.Sign(nil, digest, crypto.SHA256)
	require.NoError(t, err)
	assert.NotEmpty(t, sig2)

	err = rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, digest, sig2)
	assert.NoError(t, err, "Signer signature verification failed")
}

func TestPublicJWK_External(t *testing.T) {
	simKMS, err := NewSimulatedKMS("jwk-test-key")
	require.NoError(t, err)
	km := NewExternalKeyManager(simKMS)

	jwk, err := PublicJWK(km)
	require.NoError(t, err)
	assert.Equal(t, "RSA", jwk["kty"])
	assert.Equal(t, "RS256", jwk["alg"])
	assert.Equal(t, "jwk-test-key", jwk["kid"])
	assert.Equal(t, "sig", jwk["use"])

	// Check n and e are present
	assert.NotEmpty(t, jwk["n"])
	assert.NotEmpty(t, jwk["e"])

	// Verify we can decode n and e
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk["n"].(string))
	require.NoError(t, err)
	assert.Len(t, nBytes, 256) // 2048 bit key = 256 bytes

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk["e"].(string))
	require.NoError(t, err)
	assert.NotEmpty(t, eBytes)
}
