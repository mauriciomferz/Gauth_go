package rate

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// DistributedLimiter implements distributed rate limiting using Redis
type DistributedLimiter struct {
	client    *redis.Client
	keyPrefix string
	window    time.Duration
	rate      int64
	burstSize int64
}

// NewDistributedLimiter creates a new distributed rate limiter
func NewDistributedLimiter(cfg Config) (*DistributedLimiter, error) {
	if cfg.DistributedConfig == nil {
		return nil, ErrInvalidConfig
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.DistributedConfig.Addresses[0], // TODO: Support cluster
		Password: cfg.DistributedConfig.Password,
		DB:       cfg.DistributedConfig.DB,
	})

	return &DistributedLimiter{
		client:    client,
		keyPrefix: cfg.DistributedConfig.KeyPrefix,
		window:    cfg.Window,
		rate:      cfg.Rate,
		burstSize: cfg.BurstSize,
	}, nil
}

// Allow implements the Limiter interface
func (dl *DistributedLimiter) Allow(ctx context.Context, id string) error {
	key := dl.keyPrefix + id

	// Lua script for atomic rate limiting
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local rate = tonumber(ARGV[3])

		-- Clean old data
		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

		-- Count requests in window
		local count = redis.call('ZCARD', key)
		
		if count >= rate then
			return 0
		end

		-- Add new request
		redis.call('ZADD', key, now, now)
		redis.call('EXPIRE', key, window)
		
		return 1
	`

	result, err := dl.client.Eval(ctx, script, []string{key},
		time.Now().Unix(),
		int64(dl.window.Seconds()),
		dl.rate).Result()

	if err != nil {
		return err
	}

	if result.(int64) == 0 {
		return ErrRateLimitExceeded
	}

	return nil
}

// GetRemainingRequests implements the Limiter interface
func (dl *DistributedLimiter) GetRemainingRequests(id string) int64 {
	key := dl.keyPrefix + id

	count, err := dl.client.ZCard(context.Background(), key).Result()
	if err != nil {
		return 0
	}

	return dl.rate - count
}

// Reset implements the Limiter interface
func (dl *DistributedLimiter) Reset(id string) {
	key := dl.keyPrefix + id
	dl.client.Del(context.Background(), key)
}

// Close releases resources used by the limiter
func (dl *DistributedLimiter) Close() error {
	return dl.client.Close()
}
