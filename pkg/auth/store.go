package auth

import (
	"crypto/rsa"
	"errors"
	"sync"
)

// MemoryKeyStore implements KeyProvider in memory
type MemoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey // Map "clientID:keyID" -> PublicKey
}

// NewMemoryKeyStore creates a new memory key store
func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{
		keys: make(map[string]*rsa.PublicKey),
	}
}

// GetPublicKey retrieves a public key
func (m *MemoryKeyStore) GetPublicKey(clientID string, keyID string) (*rsa.PublicKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.keys[clientID+":"+keyID]
	if !ok {
		return nil, errors.New("key not found")
	}
	return key, nil
}

// RegisterKey registers a public key for a client
func (m *MemoryKeyStore) RegisterKey(clientID string, keyID string, key *rsa.PublicKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys[clientID+":"+keyID] = key
}
