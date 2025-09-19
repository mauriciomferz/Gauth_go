package tokenstore

import "sync"

type DistributedStore struct{
    mu sync.RWMutex
    store map[string]TokenData
}

type DistributedConfig struct {
    Addresses []string
    Password  string
    DB        int
}

func NewDistributedStore(cfg DistributedConfig) *DistributedStore {
    return &DistributedStore{
        store: make(map[string]TokenData),
    }
}

// Store implements the Store interface for distributed storage (in-memory)
func (s *DistributedStore) Store(token string, data TokenData) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.store[token] = data
    return nil
}

// Get implements the Store interface for distributed storage (in-memory)
func (s *DistributedStore) Get(token string) (TokenData, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    data, ok := s.store[token]
    return data, ok
}

// Delete implements the Store interface for distributed storage (in-memory)
func (s *DistributedStore) Delete(token string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.store, token)
    return nil
}

func (s *DistributedStore) Close() error {
    return nil
}
