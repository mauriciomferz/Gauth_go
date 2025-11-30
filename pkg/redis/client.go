package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds the Redis configuration
type Config struct {
	Host            string
	Port            int
	Password        string
	DB              int
	MaxRetries      int
	PoolSize        int
	MinIdleConns    int
	PoolTimeout     time.Duration
	IdleTimeout     time.Duration
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

// Client wraps the Redis client
type Client struct {
	client *redis.Client
	cfg    *Config
}

// NewClient creates a new Redis client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis configuration cannot be nil")
	}

	// Set defaults
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 50
	}
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = 10
	}
	if cfg.PoolTimeout == 0 {
		cfg.PoolTimeout = 4 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 5 * time.Minute
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = time.Hour
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 3 * time.Second
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.IdleTimeout,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Redis connection established: %s:%d (db=%d, pool_size=%d)",
		cfg.Host, cfg.Port, cfg.DB, cfg.PoolSize)

	return &Client{
		client: rdb,
		cfg:    cfg,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			return fmt.Errorf("failed to close Redis connection: %w", err)
		}
		log.Println("Redis connection closed")
	}
	return nil
}

// Ping checks if Redis is reachable
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// HealthCheck performs a comprehensive health check
func (c *Client) HealthCheck(ctx context.Context) error {
	// Check connection
	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	// Check pool stats
	stats := c.client.PoolStats()
	if stats.Hits == 0 && stats.Misses > 0 {
		log.Printf("Redis pool stats: hits=%d, misses=%d, timeouts=%d, total_conns=%d, idle_conns=%d",
			stats.Hits, stats.Misses, stats.Timeouts, stats.TotalConns, stats.IdleConns)
	}

	return nil
}

// Stats returns Redis pool statistics
type Stats struct {
	Hits       uint32
	Misses     uint32
	Timeouts   uint32
	TotalConns uint32
	IdleConns  uint32
	StaleConns uint32
}

// GetStats returns current connection pool statistics
func (c *Client) GetStats() Stats {
	stats := c.client.PoolStats()
	return Stats{
		Hits:       stats.Hits,
		Misses:     stats.Misses,
		Timeouts:   stats.Timeouts,
		TotalConns: stats.TotalConns,
		IdleConns:  stats.IdleConns,
		StaleConns: stats.StaleConns,
	}
}

// ============================================================================
// TOKEN BLACKLIST OPERATIONS
// ============================================================================

// AddToBlacklist adds a token to the blacklist with TTL
func (c *Client) AddToBlacklist(ctx context.Context, tokenID string, reason string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", tokenID)
	err := c.client.Set(ctx, key, reason, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

// IsBlacklisted checks if a token is blacklisted
func (c *Client) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", tokenID)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return exists > 0, nil
}

// GetBlacklistReason gets the reason a token was blacklisted
func (c *Client) GetBlacklistReason(ctx context.Context, tokenID string) (string, error) {
	key := fmt.Sprintf("blacklist:%s", tokenID)
	reason, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("token not blacklisted")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get blacklist reason: %w", err)
	}
	return reason, nil
}

// RemoveFromBlacklist removes a token from the blacklist
func (c *Client) RemoveFromBlacklist(ctx context.Context, tokenID string) error {
	key := fmt.Sprintf("blacklist:%s", tokenID)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}
	return nil
}

// GetBlacklistedTokens returns all blacklisted tokens (paginated)
func (c *Client) GetBlacklistedTokens(ctx context.Context, cursor uint64, count int64) ([]string, uint64, error) {
	pattern := "blacklist:*"
	keys, newCursor, err := c.client.Scan(ctx, cursor, pattern, count).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan blacklist: %w", err)
	}

	// Strip prefix
	tokens := make([]string, len(keys))
	for i, key := range keys {
		tokens[i] = key[10:] // Remove "blacklist:" prefix
	}

	return tokens, newCursor, nil
}

// ============================================================================
// RATE LIMITING OPERATIONS
// ============================================================================

// IncrementRateLimit increments the rate limit counter for a key
func (c *Client) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	rateLimitKey := fmt.Sprintf("ratelimit:%s", key)

	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	return incr.Val(), nil
}

// GetRateLimit gets the current rate limit counter for a key
func (c *Client) GetRateLimit(ctx context.Context, key string) (int64, error) {
	rateLimitKey := fmt.Sprintf("ratelimit:%s", key)
	count, err := c.client.Get(ctx, rateLimitKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit: %w", err)
	}
	return count, nil
}

// ResetRateLimit resets the rate limit counter for a key
func (c *Client) ResetRateLimit(ctx context.Context, key string) error {
	rateLimitKey := fmt.Sprintf("ratelimit:%s", key)
	err := c.client.Del(ctx, rateLimitKey).Err()
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}
	return nil
}

// ============================================================================
// CIRCUIT BREAKER STATE OPERATIONS
// ============================================================================

// SetCircuitBreakerState sets the circuit breaker state
func (c *Client) SetCircuitBreakerState(ctx context.Context, breakerName string, state string, ttl time.Duration) error {
	key := fmt.Sprintf("circuit:%s:state", breakerName)
	err := c.client.Set(ctx, key, state, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set circuit breaker state: %w", err)
	}
	return nil
}

// GetCircuitBreakerState gets the circuit breaker state
func (c *Client) GetCircuitBreakerState(ctx context.Context, breakerName string) (string, error) {
	key := fmt.Sprintf("circuit:%s:state", breakerName)
	state, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "closed", nil // Default state
	}
	if err != nil {
		return "", fmt.Errorf("failed to get circuit breaker state: %w", err)
	}
	return state, nil
}

// IncrementCircuitBreakerFailures increments failure counter
func (c *Client) IncrementCircuitBreakerFailures(ctx context.Context, breakerName string) (int64, error) {
	key := fmt.Sprintf("circuit:%s:failures", breakerName)
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment circuit breaker failures: %w", err)
	}
	return count, nil
}

// ResetCircuitBreakerFailures resets failure counter
func (c *Client) ResetCircuitBreakerFailures(ctx context.Context, breakerName string) error {
	key := fmt.Sprintf("circuit:%s:failures", breakerName)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset circuit breaker failures: %w", err)
	}
	return nil
}

// ============================================================================
// ACTIVE SESSION OPERATIONS
// ============================================================================

// SetActiveSession stores an active session with TTL
func (c *Client) SetActiveSession(ctx context.Context, sessionID string, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	err := c.client.Set(ctx, key, userID, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set active session: %w", err)
	}
	return nil
}

// GetActiveSession retrieves an active session
func (c *Client) GetActiveSession(ctx context.Context, sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	userID, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("session not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get active session: %w", err)
	}
	return userID, nil
}

// DeleteActiveSession removes an active session
func (c *Client) DeleteActiveSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete active session: %w", err)
	}
	return nil
}

// ExtendSession extends the TTL of an active session
func (c *Client) ExtendSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	err := c.client.Expire(ctx, key, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}
	return nil
}

// ============================================================================
// CACHING OPERATIONS
// ============================================================================

// Set stores a value in cache with TTL
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value from cache
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found")
	}
	return val, err
}

// Delete removes a value from cache
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// Expire sets a TTL on a key
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// TTL gets the remaining TTL of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}
