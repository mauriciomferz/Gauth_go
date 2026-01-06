package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements Limiter using Redis (Fixed Window)
type RedisLimiter struct {
	client *redis.Client
	config Config
}

func NewRedisLimiter(client *redis.Client, config Config) *RedisLimiter {
	return &RedisLimiter{client: client, config: config}
}

// Ensure interface compliance
var _ Limiter = (*RedisLimiter)(nil)

func (r *RedisLimiter) Allow(key string) bool {
	return r.AllowN(key, 1)
}

func (r *RedisLimiter) AllowN(key string, n int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Fixed Window: Key = "ratelimit:<key>:<current_period>"
	// Period bucket = Now() / Period
	now := time.Now().UnixNano()
	periodIdx := now / r.config.Period.Nanoseconds()
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, periodIdx)

	// INCRBY
	count, err := r.client.IncrBy(ctx, redisKey, int64(n)).Result()
	if err != nil {
		// Fail open on redis error? Or closed?
		// For availability, usually fail open, but for strictly security fail closed.
		// Let's safe-fail closed to avoid abuse if redis is down, or just return false.
		fmt.Printf("Redis Limit Error: %v\n", err)
		return false
	}

	// Set Expiry if new
	if count == int64(n) {
		r.client.Expire(ctx, redisKey, r.config.Period)
	}

	return count <= int64(r.config.Rate)
}

func (r *RedisLimiter) AllowWithLimit(key string, limit int, period time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Fixed Window: Key = "ratelimit:<key>:<period_marker>"
	// Period bucket = Now() / Period
	now := time.Now().UnixNano()
	periodIdx := now / period.Nanoseconds()
	// Include period duration in key to differentiate distinct rules (e.g. minute vs hour)
	// format: ratelimit:<key>:<duration_ns>:<period_idx>
	redisKey := fmt.Sprintf("ratelimit:%s:%d:%d", key, period.Nanoseconds(), periodIdx)

	// INCRBY
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		fmt.Printf("Redis Limit Error: %v\n", err)
		return false // Fail closed
	}

	// Set Expiry if new
	if count == 1 {
		r.client.Expire(ctx, redisKey, period)
	}

	return count <= int64(limit)
}

func (r *RedisLimiter) Wait(ctx context.Context, key string) error {
	return r.WaitN(ctx, key, 1)
}

func (r *RedisLimiter) WaitN(ctx context.Context, key string, n int) error {
	for {
		if r.AllowN(key, n) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond): // Poll
			continue
		}
	}
}

func (r *RedisLimiter) Reset(key string) {
	// Hard to reset all windows for a key without scanning.
	// Resetting current window is easy.
	ctx := context.Background()
	now := time.Now().UnixNano()
	periodIdx := now / r.config.Period.Nanoseconds()
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, periodIdx)
	r.client.Del(ctx, redisKey)
}

func (r *RedisLimiter) Close() error {
	return nil // Client managed externally usually
}
