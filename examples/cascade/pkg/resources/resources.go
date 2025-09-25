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

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name        string      `json:"name"`
	Type        ServiceType `json:"type"`
	URL         string      `json:"url"`
	Timeout     time.Duration `json:"timeout"`
	Retries     int         `json:"retries"`
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	Threshold    int           `json:"threshold"`
	Timeout      time.Duration `json:"timeout"`
	MaxRequests  int           `json:"max_requests"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second"`
	BurstSize         int `json:"burst_size"`
}

// Resource represents a system resource
type Resource struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Status      string       `json:"status"`
	Capacity    int64        `json:"capacity"`
	Used        int64        `json:"used"`
	Available   int64        `json:"available"`
	LastUpdated time.Time    `json:"last_updated"`
}

// ResourcesPackageInit initializes the resources package
func ResourcesPackageInit() string {
	return "resources package initialized"
}
