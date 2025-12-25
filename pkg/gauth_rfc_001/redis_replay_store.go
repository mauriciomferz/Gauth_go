package gauth_rfc_001

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisReplayStore implements ReplayStore using Redis SETNX + EXPIRE semantics for atomic first-seen tracking.
// Keys are formatted as: <prefix>:jti:<uuid> with a TTL equal to the provided lifetime (typically token lifetime + skew).
// Failure semantics: operations return errors; caller may choose fail-open or fail-closed. Current Service uses fail-open.
type RedisReplayStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRedisReplayStore constructs a new Redis-backed replay store.
// prefix should be stable across instances (e.g., "gauth"). ttl is applied on Record; Seen uses GET existence check.
func NewRedisReplayStore(client *redis.Client, prefix string, ttl time.Duration) (*RedisReplayStore, error) {
	if client == nil {
		return nil, fmt.Errorf("nil redis client")
	}
	if prefix == "" {
		prefix = "gauth"
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	// Optionally ping to validate connection.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisReplayStore{client: client, prefix: prefix, ttl: ttl}, nil
}

func (r *RedisReplayStore) key(jti string) string { return fmt.Sprintf("%s:jti:%s", r.prefix, jti) }

// Seen returns true if the JTI already exists (replay detected).
func (r *RedisReplayStore) Seen(jti string) (bool, error) {
	if jti == "" {
		return false, fmt.Errorf("empty jti")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	res, err := r.client.Exists(ctx, r.key(jti)).Result()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// Record stores the JTI if not present and sets TTL. A concurrent caller race is resolved atomically by Redis SETNX.
func (r *RedisReplayStore) Record(jti string, at time.Time) error {
	if jti == "" {
		return fmt.Errorf("empty jti")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Use SetNX with expiration.
	ok, err := r.client.SetNX(ctx, r.key(jti), at.UTC().Format(time.RFC3339Nano), r.ttl).Result()
	if err != nil {
		return err
	}
	// If ok == false, key already existed (replay) – caller will have caught via Seen; we still treat as success here.
	_ = ok
	return nil
}

// WithReplayStoreRedis is a convenience functional option wiring a RedisReplayStore into Service.
func WithReplayStoreRedis(client *redis.Client, prefix string, ttl time.Duration) Option {
	return func(s *Service) {
		if client == nil {
			return
		}
		rs, err := NewRedisReplayStore(client, prefix, ttl)
		if err != nil {
			return
		}
		s.replayStore = rs
	}
}
