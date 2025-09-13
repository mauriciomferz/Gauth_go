// Package tokenstore provides token storage functionality for GAuth
package tokenstore

import (
	"sync"
	"time"
)

// TokenData represents the data associated with a token
type TokenData struct {
	Valid      bool
	ValidUntil time.Time
	ClientID   string
	OwnerID    string
	Scope      []string
}

// Store defines the interface for token storage implementations
type Store interface {
	// Store stores a token with its associated data
	Store(token string, data TokenData) error

	// Get retrieves token data for a given token
	Get(token string) (TokenData, bool)

	// Delete removes a token from the store
	Delete(token string) error

	// Cleanup removes expired tokens
	Cleanup() error
}

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
		if now.After(data.ValidUntil) {
			delete(m.store, token)
		}
	}
	return nil
}
