package resources

import "time"

// RequestDetails encapsulates the details of a service request
type RequestDetails struct {
	RequestID     string
	SourceService string
	TargetService string
	Timestamp     time.Time
	Timeout       time.Duration
	Priority      int
	Metadata      map[string]string // Limited to string values for clarity
}

// ServiceConfig represents service configuration options
type ServiceConfig struct {
	Name              string
	Type              ServiceType
	Version           string
	Dependencies      []ServiceType
	CircuitBreaker    CircuitBreakerConfig
	RateLimit         RateLimitConfig
	MaxConcurrent     int
	MaxConcurrency    int // Alias for MaxConcurrent
	RequestsPerSecond int
	TimeoutSeconds    int
	Timeout           time.Duration
	RetryAttempts     int
	RetryBackoff      time.Duration
}

// ResourceUsage tracks resource utilization
type ResourceUsage struct {
	CPUPercent    float64
	MemoryPercent float64
	Connections   int
	ThreadCount   int
	UpdatedAt     time.Time
}

// HealthStatus represents the current health status of a service
type HealthStatus struct {
	Status       StatusType
	Message      string
	LastChecked  time.Time
	Dependencies map[string]bool
}

// StatusType represents different health status values
type StatusType int

const (
	StatusHealthy StatusType = iota
	StatusDegraded
	StatusUnhealthy
	StatusUnknown
)

func (st StatusType) String() string {
	switch st {
	case StatusHealthy:
		return "Healthy"
	case StatusDegraded:
		return "Degraded"
	case StatusUnhealthy:
		return "Unhealthy"
	default:
		return "Unknown"
	}
}

// ServiceType represents different types of services in the mesh
type ServiceType int

const (
	// Core Services
	AuthService ServiceType = iota
	UserService
	OrderService
	InventoryService
	PaymentService
	NotificationService
	LogisticsService
)

// String returns the string representation of a ServiceType
func (s ServiceType) String() string {
	switch s {
	case AuthService:
		return "AuthService"
	case UserService:
		return "UserService"
	case OrderService:
		return "OrderService"
	case InventoryService:
		return "InventoryService"
	case PaymentService:
		return "PaymentService"
	case NotificationService:
		return "NotificationService"
	case LogisticsService:
		return "LogisticsService"
	default:
		return "UnknownService"
	}
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	ErrorThreshold int
	ResetTimeout   time.Duration
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
}
