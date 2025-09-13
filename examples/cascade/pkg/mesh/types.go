package mesh

import (
	"sync"
	"time"
)

// ServiceType represents different types of microservices in the mesh
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

// HealthMetrics tracks service health and performance metrics
type HealthMetrics struct {
	failures      int
	successes     int
	responseTimes []time.Duration
	lastFailure   time.Time
	mu            sync.RWMutex
}

// SuccessRate returns the percentage of successful requests
func (h *HealthMetrics) SuccessRate() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := h.failures + h.successes
	if total == 0 {
		return 100.0
	}
	return float64(h.successes) / float64(total) * 100.0
}

// LastFailureTime returns the time of the last recorded failure
func (h *HealthMetrics) LastFailureTime() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastFailure
}

// Snapshot returns a point-in-time snapshot of the metrics
type HealthSnapshot struct {
	TotalRequests   int
	SuccessRate     float64
	LastFailureTime time.Time
	AverageLatency  time.Duration
}

// GetSnapshot returns the current state of health metrics
func (h *HealthMetrics) GetSnapshot() HealthSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var avgLatency time.Duration
	if len(h.responseTimes) > 0 {
		var total time.Duration
		for _, t := range h.responseTimes {
			total += t
		}
		avgLatency = total / time.Duration(len(h.responseTimes))
	}

	return HealthSnapshot{
		TotalRequests:   h.failures + h.successes,
		SuccessRate:     float64(h.successes) / float64(h.failures+h.successes) * 100.0,
		LastFailureTime: h.lastFailure,
		AverageLatency:  avgLatency,
	}
}

// DependencyGraph represents service dependencies in the mesh
type DependencyGraph struct {
	// dependencies maps a service to its dependent services
	dependencies map[ServiceType][]ServiceType
}

// String returns a human-readable name for the service type
func (st ServiceType) String() string {
	switch st {
	case AuthService:
		return "Auth"
	case UserService:
		return "User"
	case OrderService:
		return "Order"
	case InventoryService:
		return "Inventory"
	case PaymentService:
		return "Payment"
	case NotificationService:
		return "Notification"
	case LogisticsService:
		return "Logistics"
	default:
		return "Unknown"
	}
}
