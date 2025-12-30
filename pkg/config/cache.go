package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/cache"
)

// LoadCacheConfig loads cache configuration from environment variables
func LoadCacheConfig() *cache.Config {
	config := cache.DefaultConfig()

	// Cache type (redis or memory)
	if cacheType := os.Getenv("CACHE_TYPE"); cacheType != "" {
		config.Type = cacheType
	}

	// Redis configuration
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		config.RedisURL = redisURL
	}

	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.RedisPassword = redisPassword
	}

	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			config.RedisDB = db
		}
	}

	if poolSize := os.Getenv("REDIS_POOL_SIZE"); poolSize != "" {
		if size, err := strconv.Atoi(poolSize); err == nil {
			config.PoolSize = size
		}
	}

	if maxRetries := os.Getenv("REDIS_MAX_RETRIES"); maxRetries != "" {
		if retries, err := strconv.Atoi(maxRetries); err == nil {
			config.MaxRetries = retries
		}
	}

	// TTL configuration
	if verificationTTL := os.Getenv("CACHE_VERIFICATION_TTL"); verificationTTL != "" {
		if ttl, err := time.ParseDuration(verificationTTL); err == nil {
			config.VerificationTTL = ttl
		}
	}

	if poaTTL := os.Getenv("CACHE_POA_TTL"); poaTTL != "" {
		if ttl, err := time.ParseDuration(poaTTL); err == nil {
			config.PoATTL = ttl
		}
	}

	if statsTTL := os.Getenv("CACHE_STATS_TTL"); statsTTL != "" {
		if ttl, err := time.ParseDuration(statsTTL); err == nil {
			config.StatsTTL = ttl
		}
	}

	// Memory cache configuration
	if maxSize := os.Getenv("CACHE_MAX_SIZE"); maxSize != "" {
		if size, err := strconv.Atoi(maxSize); err == nil {
			config.MaxSize = size
		}
	}

	return config
}

// ValidateCacheConfig validates the cache configuration
func ValidateCacheConfig(config *cache.Config) error {
	if config.Type != "redis" && config.Type != "memory" {
		return fmt.Errorf("invalid cache type: %s (must be 'redis' or 'memory')", config.Type)
	}

	if config.Type == "redis" && config.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required when CACHE_TYPE is 'redis'")
	}

	if config.PoolSize <= 0 {
		return fmt.Errorf("invalid pool size: %d (must be > 0)", config.PoolSize)
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("invalid max retries: %d (must be >= 0)", config.MaxRetries)
	}

	if config.VerificationTTL <= 0 {
		return fmt.Errorf("invalid verification TTL: %s (must be > 0)", config.VerificationTTL)
	}

	if config.PoATTL <= 0 {
		return fmt.Errorf("invalid PoA TTL: %s (must be > 0)", config.PoATTL)
	}

	if config.StatsTTL <= 0 {
		return fmt.Errorf("invalid stats TTL: %s (must be > 0)", config.StatsTTL)
	}

	if config.Type == "memory" && config.MaxSize <= 0 {
		return fmt.Errorf("invalid max size: %d (must be > 0)", config.MaxSize)
	}

	return nil
}
