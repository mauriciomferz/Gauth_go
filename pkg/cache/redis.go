package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/metrics"
	"github.com/go-redis/redis/v8"
)

// RedisCache provides Redis-backed caching for identity validation results
type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(redisURL string, defaultTTL time.Duration) (*RedisCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:     client,
		defaultTTL: defaultTTL,
	}, nil
}

// ValidationResult represents cached validation result
type ValidationResult struct {
	Valid        bool                   `json:"valid"`
	Result       map[string]interface{} `json:"result"`
	CachedAt     time.Time              `json:"cached_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// GetValidation retrieves a cached validation result
func (c *RedisCache) GetValidation(ctx context.Context, country, docType, docNumber string) (*ValidationResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("get", "success", duration)
	}()

	key := c.validationKey(country, docType, docNumber)

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		metrics.RecordCacheOperation(country, docType, false)
		return nil, nil // Not found
	}
	if err != nil {
		metrics.RecordRedisOperation("get", "error", time.Since(start).Seconds())
		return nil, err
	}

	var result ValidationResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	metrics.RecordCacheOperation(country, docType, true)
	return &result, nil
}

// SetValidation caches a validation result
func (c *RedisCache) SetValidation(ctx context.Context, country, docType, docNumber string, result *ValidationResult, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("set", "success", duration)
	}()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	result.CachedAt = time.Now()
	result.ExpiresAt = time.Now().Add(ttl)

	key := c.validationKey(country, docType, docNumber)
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		metrics.RecordRedisOperation("set", "error", time.Since(start).Seconds())
		return err
	}

	return nil
}

// InvalidateValidation removes a cached validation
func (c *RedisCache) InvalidateValidation(ctx context.Context, country, docType, docNumber string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("del", "success", duration)
	}()

	key := c.validationKey(country, docType, docNumber)
	return c.client.Del(ctx, key).Err()
}

// GetMCPResponse retrieves a cached MCP response
func (c *RedisCache) GetMCPResponse(ctx context.Context, method string, params interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("get", "success", duration)
	}()

	key := c.mcpResponseKey(method, params)

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		metrics.RecordRedisOperation("get", "error", time.Since(start).Seconds())
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SetMCPResponse caches an MCP response
func (c *RedisCache) SetMCPResponse(ctx context.Context, method string, params interface{}, response interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("set", "success", duration)
	}()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	key := c.mcpResponseKey(method, params)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		metrics.RecordRedisOperation("set", "error", time.Since(start).Seconds())
		return err
	}

	return nil
}

// GetPoA retrieves a cached PoA credential
func (c *RedisCache) GetPoA(ctx context.Context, poaID string) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("get", "success", duration)
	}()

	key := fmt.Sprintf("poa:%s", poaID)

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		metrics.RecordRedisOperation("get", "error", time.Since(start).Seconds())
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SetPoA caches a PoA credential
func (c *RedisCache) SetPoA(ctx context.Context, poaID string, poa interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("set", "success", duration)
	}()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	key := fmt.Sprintf("poa:%s", poaID)
	data, err := json.Marshal(poa)
	if err != nil {
		return err
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		metrics.RecordRedisOperation("set", "error", time.Since(start).Seconds())
		return err
	}

	return nil
}

// InvalidatePoA removes a cached PoA
func (c *RedisCache) InvalidatePoA(ctx context.Context, poaID string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("del", "success", duration)
	}()

	key := fmt.Sprintf("poa:%s", poaID)
	return c.client.Del(ctx, key).Err()
}

// validationKey generates cache key for validation results
func (c *RedisCache) validationKey(country, docType, docNumber string) string {
	return fmt.Sprintf("validation:%s:%s:%s", country, docType, docNumber)
}

// mcpResponseKey generates cache key for MCP responses
func (c *RedisCache) mcpResponseKey(method string, params interface{}) string {
	paramsJSON, _ := json.Marshal(params)
	return fmt.Sprintf("mcp:%s:%s", method, string(paramsJSON))
}

// FlushAll clears all cached data
func (c *RedisCache) FlushAll(ctx context.Context) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordRedisOperation("flushall", "success", duration)
	}()

	return c.client.FlushAll(ctx).Err()
}

// Stats returns cache statistics
func (c *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	dbSize, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"db_size": dbSize,
		"info":    info,
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}
