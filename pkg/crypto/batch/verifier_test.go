package batch

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyBatchEd25519(t *testing.T) {
	// Generate some keys and signatures
	var items []Item
	for i := 0; i < 50; i++ {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		msg := []byte("test message")
		sig := ed25519.Sign(priv, msg)

		items = append(items, Item{
			PublicKey: pub,
			Message:   msg,
			Signature: sig,
		})
	}

	// 1. Success Case
	err := VerifyBatchEd25519(context.Background(), items)
	assert.NoError(t, err)

	// 2. Failure Case (One bad signature)
	badItems := make([]Item, len(items))
	copy(badItems, items)
	badItems[25].Signature[0] ^= 0xFF // Corrupt signature

	err = VerifyBatchEd25519(context.Background(), badItems)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")

	// 3. Invalid Key Size
	badKeyItems := make([]Item, len(items))
	copy(badKeyItems, items)
	badKeyItems[10].PublicKey = []byte("short")

	err = VerifyBatchEd25519(context.Background(), badKeyItems)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid public key size")
}
