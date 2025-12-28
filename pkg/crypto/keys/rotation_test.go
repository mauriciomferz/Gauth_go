package keys

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalKeyManager_DualKeyRotation(t *testing.T) {
	// 1. Setup Manager with Active Key
	km, err := NewLocalKeyManager("")
	require.NoError(t, err)

	// Save original active public key and ID
	activePub, err := km.GetPublicKey(context.Background())
	require.NoError(t, err)
	_, err = km.GetKeyID(context.Background())
	require.NoError(t, err)

	// 2. Simulate Rotation:
	// - Current Active becomes Previous
	// - New Key generated as Active

	// Manually inject keys into struct to simulate rotation state without file I/O complexity
	previousKey := km.privateKey
	prevKid := km.keyID

	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	modBytes := newKey.PublicKey.N.Bytes()
	newKid := base64.RawURLEncoding.EncodeToString(modBytes[:8])

	km.mu.Lock()
	km.previousKey = previousKey
	km.prevKeyID = prevKid
	km.privateKey = newKey
	km.keyID = newKid
	km.mu.Unlock()

	// 3. Verify Lookups

	// A. Lookup New Active Key
	pub, err := km.LookupPublicKey(context.Background(), newKid)
	require.NoError(t, err)
	assert.Equal(t, &newKey.PublicKey, pub, "Should find new active key")

	// B. Lookup Previous Key (Old Active)
	pubPrev, err := km.LookupPublicKey(context.Background(), prevKid)
	require.NoError(t, err)
	assert.Equal(t, activePub, pubPrev, "Should find previous key")

	// C. Lookup Unknown Key
	_, err = km.LookupPublicKey(context.Background(), "unknown-kid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// Mock KMS Client for testing Lookup in External Manager
type MockKMSInternal struct {
	activeKid   string
	prevKid     string
	activePub   crypto.PublicKey
	previousPub crypto.PublicKey
}

func (m *MockKMSInternal) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	return nil, nil
}
func (m *MockKMSInternal) PublicKey(ctx context.Context) (crypto.PublicKey, error) {
	return m.activePub, nil
}
func (m *MockKMSInternal) KeyID() string { return m.activeKid }
func (m *MockKMSInternal) LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	if kid == m.activeKid {
		return m.activePub, nil
	}
	if kid == m.prevKid {
		return m.previousPub, nil
	}
	return nil, assert.AnError
}

func TestExternalKeyManager_Lookup(t *testing.T) {
	activeKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	prevKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	mock := &MockKMSInternal{
		activeKid:   "active-1",
		prevKid:     "prev-1",
		activePub:   &activeKey.PublicKey,
		previousPub: &prevKey.PublicKey,
	}

	em := NewExternalKeyManager(mock)

	// Test Lookup
	p1, err := em.LookupPublicKey(context.Background(), "active-1")
	require.NoError(t, err)
	assert.Equal(t, &activeKey.PublicKey, p1)

	p2, err := em.LookupPublicKey(context.Background(), "prev-1")
	require.NoError(t, err)
	assert.Equal(t, &prevKey.PublicKey, p2)

	_, err = em.LookupPublicKey(context.Background(), "missing")
	require.Error(t, err)
}
