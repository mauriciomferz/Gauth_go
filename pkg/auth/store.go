package auth

import (
	"errors"
	"sync"
)

// MemoryKeyStore implements KeyProvider in memory
type MemoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]any // Map "clientID:keyID" -> PublicKey (RSA, ECDSA, Ed25519)
}

// NewMemoryKeyStore creates a new memory key store
func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{
		keys: make(map[string]any),
	}
}

// GetPublicKey retrieves a public key
func (m *MemoryKeyStore) GetPublicKey(clientID string, keyID string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.keys[clientID+":"+keyID]
	if !ok {
		return nil, errors.New("key not found")
	}
	return key, nil
}

// RegisterKey registers a public key for a client
func (m *MemoryKeyStore) RegisterKey(clientID string, keyID string, key any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys[clientID+":"+keyID] = key
}
