package resources

import (
	"time"
)

// ServiceType represents different types of services
type ServiceType string

const (
	PaymentService   ServiceType = "payment"
	OrderService     ServiceType = "order"
	InventoryService ServiceType = "inventory"
)

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	ErrorThreshold int           `json:"error_threshold"`
	ResetTimeout   time.Duration `json:"reset_timeout"`
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second"`
	BurstSize         int `json:"burst_size"`
}

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name           string               `json:"name"`
	Type           ServiceType          `json:"type"`
	Version        string               `json:"version"`
	Dependencies   []ServiceType        `json:"dependencies"`
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`
	RateLimit      RateLimitConfig      `json:"rate_limit"`
	MaxConcurrency int                  `json:"max_concurrency"`
	Timeout        time.Duration        `json:"timeout"`
}

// Resources package for service configuration
func ResourcesPackageInit() string {
	return "resources package initialized"
}
