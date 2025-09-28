package rate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// Lua script for sliding window rate limiting
	slidingWindowScript = `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])

		-- Clean old requests
		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

		-- Count requests in current window
		local count = redis.call('ZCARD', key)

		if count >= limit then
			return 0
		end

		-- Add new request
		redis.call('ZADD', key, now, now)
		redis.call('EXPIRE', key, math.ceil(window/1000000000))

		return limit - count
	`

	// Lua script for getting remaining requests
	remainingRequestsScript = `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])

		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		local count = redis.call('ZCARD', key)

		return limit - count
	`
)

// RedisLimiter implements distributed rate limiting using Redis
type RedisLimiter struct {
	client    *redis.Client
	config    Config
	keyPrefix string
}

// NewRedisLimiter creates a new Redis-based rate limiter
func NewRedisLimiter(cfg Config) (*RedisLimiter, error) {
	if cfg.DistributedConfig == nil {
		return nil, errors.New("redis configuration is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.DistributedConfig.Addresses[0],
		Password: cfg.DistributedConfig.Password,
		DB:       cfg.DistributedConfig.DB,
	})

	return &RedisLimiter{
		client:    client,
		config:    cfg,
		keyPrefix: cfg.DistributedConfig.KeyPrefix,
	}, nil
}

// Allow implements the Limiter interface
func (rl *RedisLimiter) Allow(ctx context.Context, id string) error {
	key := rl.getKey(id)
	now := time.Now().UnixNano()

	// Lua script for sliding window rate limiting

	result, err := rl.client.Eval(ctx, slidingWindowScript, []string{key},
		now,
		rl.config.Window.Nanoseconds(),
		rl.config.Rate).Result()

	if err != nil {
		return fmt.Errorf("failed to evaluate rate limit: %w", err)
	}

	remaining := result.(int64)
	if remaining <= 0 {
		return ErrRateLimitExceeded
	}

	return nil
}

// GetRemainingRequests implements the Limiter interface
func (rl *RedisLimiter) GetRemainingRequests(id string) int64 {
	key := rl.getKey(id)
	now := time.Now().UnixNano()

	// Clean old requests and count remaining
	result, err := rl.client.Eval(context.Background(), remainingRequestsScript, []string{key},
		now,
		rl.config.Window.Nanoseconds(),
		rl.config.Rate).Result()

	if err != nil {
		return 0
	}

	return result.(int64)
}

// Reset implements the Limiter interface
func (rl *RedisLimiter) Reset(id string) {
	key := rl.getKey(id)
	rl.client.Del(context.Background(), key)
}

func (rl *RedisLimiter) getKey(id string) string {
	return fmt.Sprintf("%s:%s", rl.keyPrefix, id)
}

// Close closes the Redis connection
func (rl *RedisLimiter) Close() error {
	return rl.client.Close()
}

// RedisClusterLimiter implements distributed rate limiting using Redis Cluster
type RedisClusterLimiter struct {
	cluster   *redis.ClusterClient
	config    Config
	keyPrefix string
}

// NewRedisClusterLimiter creates a new Redis Cluster-based rate limiter
func NewRedisClusterLimiter(cfg Config) (*RedisClusterLimiter, error) {
	if cfg.DistributedConfig == nil {
		return nil, errors.New("redis configuration is required")
	}

	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cfg.DistributedConfig.Addresses,
		Password: cfg.DistributedConfig.Password,
	})

	return &RedisClusterLimiter{
		cluster:   cluster,
		config:    cfg,
		keyPrefix: cfg.DistributedConfig.KeyPrefix,
	}, nil
}

// Allow implements the Limiter interface
func (rcl *RedisClusterLimiter) Allow(ctx context.Context, id string) error {
	key := rcl.getKey(id)
	now := time.Now().UnixNano()

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])

		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		local count = redis.call('ZCARD', key)

		if count >= limit then
			return 0
		end

		redis.call('ZADD', key, now, now .. '-' .. math.random())
		redis.call('EXPIRE', key, math.ceil(window/1000000000))

		return limit - count
	`

	result, err := rcl.cluster.Eval(ctx, script, []string{key},
		now,
		rcl.config.Window.Nanoseconds(),
		rcl.config.Rate).Result()

	if err != nil {
		return fmt.Errorf("failed to evaluate rate limit: %w", err)
	}

	remaining := result.(int64)
	if remaining <= 0 {
		return ErrRateLimitExceeded
	}

	return nil
}

// GetRemainingRequests implements the Limiter interface
func (rcl *RedisClusterLimiter) GetRemainingRequests(id string) int64 {
	key := rcl.getKey(id)
	now := time.Now().UnixNano()

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])

		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		local count = redis.call('ZCARD', key)

		return limit - count
	`

	result, err := rcl.cluster.Eval(context.Background(), script, []string{key},
		now,
		rcl.config.Window.Nanoseconds(),
		rcl.config.Rate).Result()

	if err != nil {
		return 0
	}

	return result.(int64)
}

// Reset implements the Limiter interface
func (rcl *RedisClusterLimiter) Reset(id string) {
	key := rcl.getKey(id)
	rcl.cluster.Del(context.Background(), key)
}

func (rcl *RedisClusterLimiter) getKey(id string) string {
	return fmt.Sprintf("%s:%s", rcl.keyPrefix, id)
}

// Close closes the Redis Cluster connection
func (rcl *RedisClusterLimiter) Close() error {
	return rcl.cluster.Close()
}
