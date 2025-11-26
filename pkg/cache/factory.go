package cache

import (
	"fmt"
	"log"
)

// NewCache creates a new cache instance based on the configuration
func NewCache(config *Config) (Cache, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case "redis":
		cache, err := NewRedisCache(config)
		if err != nil {
			log.Printf("Failed to create Redis cache: %v. Falling back to memory cache.", err)
			return NewMemoryCache(config), nil
		}
		log.Println("Redis cache initialized successfully")
		return cache, nil

	case "memory":
		log.Println("Memory cache initialized")
		return NewMemoryCache(config), nil

	default:
		return nil, fmt.Errorf("unsupported cache type: %s", config.Type)
	}
}

// NewCacheWithFallback creates a Redis cache with automatic fallback to memory cache
func NewCacheWithFallback(config *Config) Cache {
	if config == nil {
		config = DefaultConfig()
	}

	// Try Redis first
	if config.Type == "redis" || config.Type == "" {
		cache, err := NewRedisCache(config)
		if err == nil {
			log.Println("Redis cache initialized successfully")
			return cache
		}
		log.Printf("Failed to initialize Redis cache: %v. Using memory cache as fallback.", err)
	}

	// Fallback to memory cache
	log.Println("Memory cache initialized (fallback)")
	return NewMemoryCache(config)
}
