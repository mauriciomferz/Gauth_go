package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	config *Config
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *Config) (*RedisCache, error) {
	// Parse Redis options
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with config values
	if config.RedisPassword != "" {
		opt.Password = config.RedisPassword
	}
	opt.DB = config.RedisDB
	opt.PoolSize = config.PoolSize
	opt.MaxRetries = config.MaxRetries

	// Create client
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		config: config,
	}, nil
}

// Get retrieves a value from Redis
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Key not found
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in Redis with TTL
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a value from Redis
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// DeletePattern removes all keys matching the pattern
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	// Use SCAN to find matching keys
	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error
		batch, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Delete keys in batches
	if len(keys) > 0 {
		pipe := r.client.Pipeline()
		for _, key := range keys {
			pipe.Del(ctx, key)
		}
		_, err := pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// Exists checks if a key exists in Redis
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return count > 0, nil
}

// GetStats returns Redis statistics
func (r *RedisCache) GetStats(ctx context.Context) (*Stats, error) {
	// Get keyspace stats
	keyspace, err := r.client.DBSize(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DB size: %w", err)
	}

	// Get memory info
	info, err := r.client.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	stats := &Stats{
		Keys:        keyspace,
		Connections: r.config.PoolSize,
	}

	// Parse memory usage from INFO output
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				memory, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				stats.Memory = memory
			}
		}
		if strings.HasPrefix(line, "keyspace_hits:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				hits, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				stats.Hits = hits
			}
		}
		if strings.HasPrefix(line, "keyspace_misses:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				misses, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				stats.Misses = misses
			}
		}
	}

	// Calculate hit rate
	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
	}

	return stats, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Ping checks if Redis is reachable
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Additional Redis-specific methods

// SetNX sets a key only if it doesn't exist (for locking)
func (r *RedisCache) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to SetNX key %s: %w", key, err)
	}
	return ok, nil
}

// Increment increments a counter
func (r *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}
	return val, nil
}

// GetTTL returns the remaining TTL for a key
func (r *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}
	return ttl, nil
}

// MGet retrieves multiple values at once
func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to mget keys: %w", err)
	}

	result := make([][]byte, len(vals))
	for i, val := range vals {
		if val != nil {
			if str, ok := val.(string); ok {
				result[i] = []byte(str)
			}
		}
	}

	return result, nil
}

// MSet sets multiple key-value pairs at once
func (r *RedisCache) MSet(ctx context.Context, pairs map[string][]byte, ttl time.Duration) error {
	pipe := r.client.Pipeline()

	for key, value := range pairs {
		pipe.Set(ctx, key, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mset keys: %w", err)
	}

	return nil
}
