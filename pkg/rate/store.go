package rate

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Store defines the interface for rate limit storage backends
type Store interface {
	// GetCount gets the current request count for a key
	GetCount(ctx context.Context, key string) (int, error)

	// Increment increments the count for a key with expiration
	Increment(ctx context.Context, key string, expiry time.Duration) error

	// Reset resets the count for a key
	Reset(ctx context.Context, key string) error

	// ResetAll resets all counts
	ResetAll(ctx context.Context) error

	// Cleanup removes expired entries
	Cleanup(ctx context.Context) error
}

// RedisStore implements Store using Redis
type RedisStore struct {
	client RedisClient
	prefix string
}

// RedisClient defines required Redis operations
type RedisClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiry time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
	Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error)
	Expire(ctx context.Context, key string, expiry time.Duration) (bool, error)
}

// NewRedisStore creates a new Redis-backed store
func NewRedisStore(client RedisClient, prefix string) *RedisStore {
	return &RedisStore{
		client: client,
		prefix: prefix,
	}
}

func (s *RedisStore) key(key string) string {
	return s.prefix + key
}

// GetCount implements Store
func (s *RedisStore) GetCount(ctx context.Context, key string) (int, error) {
	val, err := s.client.Get(ctx, s.key(key))
	if err != nil {
		return 0, err
	}

	var count int
	_, err = fmt.Sscan(val, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Increment implements Store
func (s *RedisStore) Increment(ctx context.Context, key string, expiry time.Duration) error {
	k := s.key(key)

	_, err := s.client.Incr(ctx, k)
	if err != nil {
		return err
	}

	// Set expiry
	_, err = s.client.Expire(ctx, k, expiry)
	return err
}

// Reset implements Store
func (s *RedisStore) Reset(ctx context.Context, key string) error {
	_, err := s.client.Del(ctx, s.key(key))
	return err
}

// ResetAll implements Store
func (s *RedisStore) ResetAll(ctx context.Context) error {
	var cursor uint64
	var err error

	// Scan and delete all keys with prefix
	for {
		var keys []string
		keys, cursor, err = s.client.Scan(ctx, cursor, s.prefix+"*", 100)
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			_, err = s.client.Del(ctx, keys...)
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// Cleanup implements Store
func (s *RedisStore) Cleanup(_ context.Context) error {
	// Redis handles expiration automatically
	return nil
}

// MemoryStore implements Store using in-memory storage
type MemoryStore struct {
	mu      sync.RWMutex
	counts  map[string]int
	expires map[string]time.Time
}

// NewMemoryStore creates a new memory-backed store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		counts:  make(map[string]int),
		expires: make(map[string]time.Time),
	}
}

// GetCount implements Store
func (s *MemoryStore) GetCount(_ context.Context, key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if expires, ok := s.expires[key]; ok && time.Now().After(expires) {
		return 0, nil
	}

	return s.counts[key], nil
}

// Increment implements Store
func (s *MemoryStore) Increment(_ context.Context, key string, expiry time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counts[key]++
	s.expires[key] = time.Now().Add(expiry)
	return nil
}

// Reset implements Store
func (s *MemoryStore) Reset(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.counts, key)
	delete(s.expires, key)
	s.mu.Unlock()
	return nil
}

// ResetAll implements Store
func (s *MemoryStore) ResetAll(ctx context.Context) error {
	s.mu.Lock()
	s.counts = make(map[string]int)
	s.expires = make(map[string]time.Time)
	s.mu.Unlock()
	return nil
}

// Cleanup implements Store
func (s *MemoryStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, expires := range s.expires {
		if now.After(expires) {
			delete(s.counts, key)
			delete(s.expires, key)
		}
	}
	return nil
}
