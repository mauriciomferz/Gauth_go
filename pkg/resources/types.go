// Package resources provides strongly-typed resource definitions and configuration.
package resources

import (
	"time"
)

// ServiceType represents different types of services
type ServiceType string

const (
	// Core service types
	AuthService      ServiceType = "auth"
	UserService      ServiceType = "user"
	OrderService     ServiceType = "order"
	PaymentService   ServiceType = "payment"
	InventoryService ServiceType = "inventory"
)

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	StatusHealthy     ServiceStatus = "healthy"
	StatusDegraded    ServiceStatus = "degraded"
	StatusUnhealthy   ServiceStatus = "unhealthy"
	StatusMaintenance ServiceStatus = "maintenance"
)

// ServiceConfig provides strongly-typed configuration for services
type ServiceConfig struct {
	// Core settings
	Type         ServiceType   `json:"type"`
	Name         string        `json:"name"`
	Version      string        `json:"version"`
	Dependencies []ServiceType `json:"dependencies"`
	Status       ServiceStatus `json:"status"`

	// Resilience settings
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`
	RateLimit      RateLimitConfig      `json:"rate_limit"`
	Bulkhead       BulkheadConfig       `json:"bulkhead"`
	Retry          RetryConfig          `json:"retry"`

	// Resource limits
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        time.Duration `json:"timeout"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	ErrorThreshold int           `json:"error_threshold"`
	ResetTimeout   time.Duration `json:"reset_timeout"`
	HalfOpenCalls  int           `json:"half_open_calls"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	BurstSize         int     `json:"burst_size"`
	WindowSize        int     `json:"window_size"`
}

// BulkheadConfig defines resource isolation settings
type BulkheadConfig struct {
	MaxConcurrent int `json:"max_concurrent"`
	QueueSize     int `json:"queue_size"`
}

// RetryConfig defines retry behavior settings
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	BackoffBase time.Duration `json:"backoff_base"`
	MaxBackoff  time.Duration `json:"max_backoff"`
}

// ServiceMetrics provides strongly-typed service metrics
type ServiceMetrics struct {
	// Request metrics
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulCalls int64         `json:"successful_calls"`
	FailedCalls     int64         `json:"failed_calls"`
	AverageLatency  time.Duration `json:"average_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`

	// Circuit breaker metrics
	CircuitState    string    `json:"circuit_state"`
	ErrorRate       float64   `json:"error_rate"`
	LastFailureTime time.Time `json:"last_failure_time"`

	// Rate limiting metrics
	CurrentRate      float64 `json:"current_rate"`
	RejectedRequests int64   `json:"rejected_requests"`

	// Resource metrics
	ActiveRequests int     `json:"active_requests"`
	QueuedRequests int     `json:"queued_requests"`
	ResourceUsage  float64 `json:"resource_usage"`
}
