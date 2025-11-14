package oidc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisRateLimiter implements distributed rate limiting using Redis.
// Suitable for multi-instance deployments where rate limits need to be shared.
type RedisRateLimiter struct {
	client    *redis.Client
	limit     int           // Requests per window
	window    time.Duration // Time window
	keyPrefix string
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter.
func NewRedisRateLimiter(client *redis.Client, limit int, window time.Duration, keyPrefix string) *RedisRateLimiter {
	if keyPrefix == "" {
		keyPrefix = "ratelimit"
	}
	return &RedisRateLimiter{
		client:    client,
		limit:     limit,
		window:    window,
		keyPrefix: keyPrefix,
	}
}

// Allow checks if a request should be allowed using Redis for distributed counting.
func (l *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks if N requests should be allowed using a Lua script for atomicity.
func (l *RedisRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s", l.keyPrefix, key)

	// Use Lua script to atomically check and increment counter
	// This implements a sliding window counter
	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local increment = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
		
		-- Count current requests in the window
		local current = redis.call('ZCARD', key)
		
		-- Check if we're within the limit
		if current + increment <= limit then
			-- Add new entries
			for i = 1, increment do
				redis.call('ZADD', key, now, now .. ':' .. i)
			end
			-- Set expiration
			redis.call('EXPIRE', key, window)
			return 1
		else
			return 0
		end
	`)

	now := time.Now().UnixMilli()
	windowMs := l.window.Milliseconds()

	result, err := script.Run(ctx, l.client, []string{redisKey}, l.limit, windowMs, n, now).Int()
	if err != nil {
		return false, fmt.Errorf("redis rate limit script failed: %w", err)
	}

	return result == 1, nil
}

// Reset resets the rate limit for a specific key.
func (l *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s:%s", l.keyPrefix, key)
	return l.client.Del(ctx, redisKey).Err()
}

// GetLimit returns the current limit configuration.
func (l *RedisRateLimiter) GetLimit() (int, time.Duration) {
	return l.limit, l.window
}

// Close closes the Redis client.
func (l *RedisRateLimiter) Close() error {
	return l.client.Close()
}

// RedisRateLimitService provides distributed rate limiting using Redis.
type RedisRateLimitService struct {
	config      *RateLimitConfig
	redisClient *redis.Client
	limiters    map[string]RateLimiter
	mu          sync.RWMutex
}

// NewRedisRateLimitService creates a new Redis-backed rate limiting service.
func NewRedisRateLimitService(config *RateLimitConfig, redisAddr, redisPassword string, redisDB int) (*RedisRateLimitService, error) {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           redisDB,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	service := &RedisRateLimitService{
		config:      config,
		redisClient: client,
		limiters:    make(map[string]RateLimiter),
	}

	// Create Redis-backed limiters for each endpoint
	if config.TokenEndpoint.Enabled {
		service.limiters["token"] = NewRedisRateLimiter(
			client,
			config.TokenEndpoint.RequestsPerIP,
			config.TokenEndpoint.Window,
			"ratelimit:token",
		)
	}
	if config.RefreshEndpoint.Enabled {
		service.limiters["refresh"] = NewRedisRateLimiter(
			client,
			config.RefreshEndpoint.RequestsPerIP,
			config.RefreshEndpoint.Window,
			"ratelimit:refresh",
		)
	}
	if config.IntrospectionEndpoint.Enabled {
		service.limiters["introspection"] = NewRedisRateLimiter(
			client,
			config.IntrospectionEndpoint.RequestsPerIP,
			config.IntrospectionEndpoint.Window,
			"ratelimit:introspection",
		)
	}
	if config.RevocationEndpoint.Enabled {
		service.limiters["revocation"] = NewRedisRateLimiter(
			client,
			config.RevocationEndpoint.RequestsPerIP,
			config.RevocationEndpoint.Window,
			"ratelimit:revocation",
		)
	}
	if config.DeviceAuthEndpoint.Enabled {
		service.limiters["device_auth"] = NewRedisRateLimiter(
			client,
			config.DeviceAuthEndpoint.RequestsPerIP,
			config.DeviceAuthEndpoint.Window,
			"ratelimit:device_auth",
		)
	}
	if config.PAREndpoint.Enabled {
		service.limiters["par"] = NewRedisRateLimiter(
			client,
			config.PAREndpoint.RequestsPerIP,
			config.PAREndpoint.Window,
			"ratelimit:par",
		)
	}
	if config.JWKSEndpoint.Enabled {
		service.limiters["jwks"] = NewRedisRateLimiter(
			client,
			config.JWKSEndpoint.RequestsPerIP,
			config.JWKSEndpoint.Window,
			"ratelimit:jwks",
		)
	}
	if config.UserinfoEndpoint.Enabled {
		service.limiters["userinfo"] = NewRedisRateLimiter(
			client,
			config.UserinfoEndpoint.RequestsPerIP,
			config.UserinfoEndpoint.Window,
			"ratelimit:userinfo",
		)
	}

	// Global limiters
	if config.GlobalIPLimit != nil && config.GlobalIPLimit.Enabled {
		service.limiters["global_ip"] = NewRedisRateLimiter(
			client,
			config.GlobalIPLimit.RequestsPerIP,
			config.GlobalIPLimit.Window,
			"ratelimit:global:ip",
		)
	}
	if config.GlobalClientLimit != nil && config.GlobalClientLimit.Enabled {
		service.limiters["global_client"] = NewRedisRateLimiter(
			client,
			config.GlobalClientLimit.RequestsPerIP,
			config.GlobalClientLimit.Window,
			"ratelimit:global:client",
		)
	}

	return service, nil
}

// CheckLimit checks if a request should be allowed for a specific endpoint.
func (s *RedisRateLimitService) CheckLimit(ctx context.Context, endpoint, ip, clientID string) error {
	if !s.config.Enabled {
		return nil
	}

	// Check global IP limit first
	if s.config.GlobalIPLimit != nil && s.config.GlobalIPLimit.Enabled {
		limiter := s.limiters["global_ip"]
		allowed, err := limiter.Allow(ctx, fmt.Sprintf("ip:%s", ip))
		if err != nil {
			return fmt.Errorf("rate limit check failed: %w", err)
		}
		if !allowed {
			return s.rateLimitError("global IP limit exceeded", s.config.GlobalIPLimit.Window)
		}
	}

	// Check global client limit
	if clientID != "" && s.config.GlobalClientLimit != nil && s.config.GlobalClientLimit.Enabled {
		limiter := s.limiters["global_client"]
		allowed, err := limiter.Allow(ctx, fmt.Sprintf("client:%s", clientID))
		if err != nil {
			return fmt.Errorf("rate limit check failed: %w", err)
		}
		if !allowed {
			return s.rateLimitError("client limit exceeded", s.config.GlobalClientLimit.Window)
		}
	}

	// Check endpoint-specific limit
	limiter, exists := s.limiters[endpoint]
	if !exists {
		return nil // No limit configured for this endpoint
	}

	key := fmt.Sprintf("%s:ip:%s", endpoint, ip)
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	if !allowed {
		requests, window := limiter.GetLimit()
		return &OIDCError{
			ErrorCode:        "rate_limit_exceeded",
			ErrorDescription: fmt.Sprintf("Rate limit exceeded: %d requests per %s", requests, window),
		}
	}

	return nil
}

// rateLimitError creates a rate limit error with retry-after information.
func (s *RedisRateLimitService) rateLimitError(message string, window time.Duration) error {
	return &OIDCError{
		ErrorCode:        "rate_limit_exceeded",
		ErrorDescription: fmt.Sprintf("%s. Retry after %s", message, window),
	}
}

// Reset resets rate limits for a specific key.
func (s *RedisRateLimitService) Reset(ctx context.Context, endpoint, key string) error {
	limiter, exists := s.limiters[endpoint]
	if !exists {
		return fmt.Errorf("no limiter found for endpoint: %s", endpoint)
	}
	return limiter.Reset(ctx, key)
}

// Close closes all rate limiters and Redis connection.
func (s *RedisRateLimitService) Close() error {
	for _, limiter := range s.limiters {
		if err := limiter.Close(); err != nil {
			return err
		}
	}
	return s.redisClient.Close()
}
