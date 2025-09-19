package tokenstore

import (
	"sync"
	"time"
)

// MemoryStore implements Store using in-memory storage
type MemoryStore struct {
	mu    sync.RWMutex
	store map[string]TokenData
}

// NewMemoryStore creates a new in-memory token store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store: make(map[string]TokenData),
	}
}

func (m *MemoryStore) Store(token string, data TokenData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[token] = data
	return nil
}

func (m *MemoryStore) Get(token string) (TokenData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, exists := m.store[token]
	return data, exists
}

func (m *MemoryStore) Delete(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, token)
	return nil
}

func (m *MemoryStore) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for token, data := range m.store {
		if !data.Valid || now.After(data.ValidUntil) {
			delete(m.store, token)
		}
	}
	return nil
}
