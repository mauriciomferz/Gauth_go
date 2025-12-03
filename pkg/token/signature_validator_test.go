package token

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHMACValidator(t *testing.T) {
	key := []byte("test-secret")
	validator := NewHMACValidator(key)

	claims := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(key)
	require.NoError(t, err)

	t.Run("ValidToken", func(t *testing.T) {
		err := validator.Validate(context.Background(), signedToken)
		assert.NoError(t, err)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		err := validator.Validate(context.Background(), signedToken+"invalid")
		assert.Error(t, err)
	})

	t.Run("WrongSigningKey", func(t *testing.T) {
		wrongKeyValidator := NewHMACValidator([]byte("wrong-key"))
		err := wrongKeyValidator.Validate(context.Background(), signedToken)
		assert.Error(t, err)
		validationErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Equal(t, "invalid_signature", validationErr.Code)
	})
}

func TestRSAVerifier(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	verifier := NewRSAVerifier(publicKey)

	claims := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	t.Run("ValidToken", func(t *testing.T) {
		err := verifier.Validate(context.Background(), signedToken)
		assert.NoError(t, err)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		err := verifier.Validate(context.Background(), signedToken+"invalid")
		assert.Error(t, err)
	})

	t.Run("WrongPublicKey", func(t *testing.T) {
		wrongPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		wrongPublicKey := &wrongPrivateKey.PublicKey
		wrongKeyVerifier := NewRSAVerifier(wrongPublicKey)
		err := wrongKeyVerifier.Validate(context.Background(), signedToken)
		assert.Error(t, err)
	})
}

func TestECDSAValidator(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	validator := NewECDSAValidator(publicKey)

	claims := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	t.Run("ValidToken", func(t *testing.T) {
		err := validator.Validate(context.Background(), signedToken)
		assert.NoError(t, err)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		err := validator.Validate(context.Background(), signedToken+"invalid")
		assert.Error(t, err)
	})

	t.Run("WrongPublicKey", func(t *testing.T) {
		wrongPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		wrongPublicKey := &wrongPrivateKey.PublicKey
		wrongKeyValidator := NewECDSAValidator(wrongPublicKey)
		err := wrongKeyValidator.Validate(context.Background(), signedToken)
		assert.Error(t, err)
	})
}

func TestValidatorRegistry(t *testing.T) {
	registry := NewValidatorRegistry()

	// Setup keys
	hmacKey := []byte("hmac-secret")
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecdsaKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Register validators
	registry.Register("HS256", NewHMACValidator(hmacKey))
	registry.Register("RS256", NewRSAVerifier(&rsaKey.PublicKey))
	registry.Register("ES256", NewECDSAValidator(&ecdsaKey.PublicKey))

	// Create tokens
	hs256Token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u1"}).SignedString(hmacKey)
	rs256Token, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "u2"}).SignedString(rsaKey)
	es256Token, _ := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"sub": "u3"}).SignedString(ecdsaKey)

	t.Run("ValidateHS256", func(t *testing.T) {
		err := registry.Validate(context.Background(), hs256Token)
		assert.NoError(t, err)
	})

	t.Run("ValidateRS256", func(t *testing.T) {
		err := registry.Validate(context.Background(), rs256Token)
		assert.NoError(t, err)
	})

	t.Run("ValidateES256", func(t *testing.T) {
		err := registry.Validate(context.Background(), es256Token)
		assert.NoError(t, err)
	})

	t.Run("UnsupportedAlgorithm", func(t *testing.T) {
		ps256Token, _ := jwt.NewWithClaims(jwt.SigningMethodPS256, jwt.MapClaims{"sub": "u4"}).SignedString(rsaKey)
		err := registry.Validate(context.Background(), ps256Token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported signing method")
	})

	t.Run("MalformedToken", func(t *testing.T) {
		err := registry.Validate(context.Background(), "malformed.token.string")
		assert.Error(t, err)
	})
}
