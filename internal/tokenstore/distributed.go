package tokenstore

import (
	"sync"
	"time"
)

// DistributedConfig holds configuration for distributed token store
type DistributedConfig struct {
	Addresses []string
	Password  string
	DB        int
}

// DistributedStore implements a distributed token store
// For testing purposes, this is a simple implementation that could be replaced with Redis
type DistributedStore struct {
	config DistributedConfig
	mu     sync.RWMutex
	store  map[string]TokenData
}

// NewDistributedStore creates a new distributed token store
func NewDistributedStore(config DistributedConfig) *DistributedStore {
	return &DistributedStore{
		config: config,
		store:  make(map[string]TokenData),
	}
}

// Store stores a token with its associated data
func (d *DistributedStore) Store(token string, data TokenData) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store[token] = data
	return nil
}

// Get retrieves token data for a given token
func (d *DistributedStore) Get(token string) (TokenData, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	data, exists := d.store[token]
	return data, exists
}

// Delete removes a token from the store
func (d *DistributedStore) Delete(token string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.store, token)
	return nil
}

// Cleanup removes expired tokens
func (d *DistributedStore) Cleanup() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	now := time.Now()
	for token, data := range d.store {
		if now.After(data.ValidUntil) {
			delete(d.store, token)
		}
	}
	return nil
}

// Close closes the distributed store connection
func (d *DistributedStore) Close() error {
	// In a real implementation, this would close Redis connections
	// For now, just clear the store
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store = make(map[string]TokenData)
	return nil
}
