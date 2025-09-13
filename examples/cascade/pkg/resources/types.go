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
	MaxConcurrent     int
	RequestsPerSecond int
	TimeoutSeconds    int
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
