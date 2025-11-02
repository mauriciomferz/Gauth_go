package replay

import (
	"context"
	"errors"
	"os"
	"time"

	redis "github.com/go-redis/redis/v8"
)

// RedisReplayBackend implements ExternalReplayBackend using SETNX + expiration TTL.
// Keys are stored with optional prefix. Existence implies replay.
type RedisReplayBackend struct {
    client *redis.Client
    ttl    time.Duration
    prefix string
}

// NewRedisReplayBackend constructs the adapter. Environment overrides:
//   GAUTH_REPLAY_BACKEND_TTL    (Go duration string, e.g. 30m, 2h)
//   GAUTH_REPLAY_BACKEND_PREFIX (string prefix for keys)
func NewRedisReplayBackend(addr string, ttl time.Duration) (*RedisReplayBackend, error) {
    if addr == "" { return nil, errors.New("redis addr empty") }
    if envTTL := os.Getenv("GAUTH_REPLAY_BACKEND_TTL"); envTTL != "" {
        if d, err := time.ParseDuration(envTTL); err == nil { ttl = d }
    }
    prefix := os.Getenv("GAUTH_REPLAY_BACKEND_PREFIX")
    if ttl <= 0 { ttl = time.Hour }
    opts := &redis.Options{Addr: addr}
    client := redis.NewClient(opts)
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    if err := client.Ping(ctx).Err(); err != nil { return nil, err }
    return &RedisReplayBackend{client: client, ttl: ttl, prefix: prefix}, nil
}

func (r *RedisReplayBackend) key(k string) string {
    if r.prefix != "" { return r.prefix + k }
    return k
}

// Seen returns true if key already recorded.
func (r *RedisReplayBackend) Seen(key string) (bool, error) {
    if key == "" { return false, nil }
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()
    res := r.client.Exists(ctx, r.key(key))
    if res.Err() != nil { return false, res.Err() }
    return res.Val() > 0, nil
}

// Record stores key with TTL (idempotent).
func (r *RedisReplayBackend) Record(key string) error {
    if key == "" { return nil }
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()
    res := r.client.SetNX(ctx, r.key(key), "1", r.ttl)
    return res.Err()
}

// Size performs SCAN with pattern prefix* (approximate, may be expensive for large sets; acceptable for benchmark/testing).
func (r *RedisReplayBackend) Size() (int, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    cursor := uint64(0)
    count := 0
    pattern := r.prefix + "*"
    for {
        keys, next, err := r.client.Scan(ctx, cursor, pattern, 1000).Result()
        if err != nil { return count, err }
        count += len(keys)
        cursor = next
        if cursor == 0 { break }
    }
    return count, nil
}

func (r *RedisReplayBackend) Close() error { return r.client.Close() }
