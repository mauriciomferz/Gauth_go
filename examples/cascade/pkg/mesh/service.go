package mesh

// Package mesh provides the core service mesh implementation.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/events"
)

// Service is an alias for Microservice for backward compatibility
type Service = Microservice

// Microservice represents an individual service in the mesh, equipped with resilience patterns and health monitoring.
type Microservice struct {
	Type         ServiceType
	Name         string
	Dependencies []ServiceType
	Health       *HealthMetrics
	loadFactor   float64
	mu           sync.RWMutex
	eventBus     *events.SimpleEventBus
}

// NewMicroservice creates a new Microservice instance.
func NewMicroservice(sType ServiceType, name string, deps []ServiceType) *Microservice {
	return &Microservice{
		Type:         sType,
		Name:         name,
		Dependencies: deps,
		Health:       &HealthMetrics{},
		loadFactor:   1.0,
		eventBus:     events.NewSimpleEventBus(),
	}
}

// SetLoadFactor sets the load factor for the microservice.
func (m *Microservice) SetLoadFactor(factor float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadFactor = factor
}

// GetLoadFactor returns the current load factor for the microservice.
func (m *Microservice) GetLoadFactor() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadFactor
}

// ProcessRequest simulates processing a request through this microservice
func (m *Microservice) ProcessRequest(ctx context.Context, mesh *ServiceMesh) error {
	m.mu.RLock()
	loadFactor := m.loadFactor
	m.mu.RUnlock()

	// Check dependencies first
	for _, depType := range m.Dependencies {
		if dep, exists := mesh.GetService(depType); exists {
			if err := dep.ProcessRequest(ctx, mesh); err != nil {
				return fmt.Errorf("%s dependency failed: %w", dep.Name, err)
			}
		}
	}

	// Simulate processing with current load factor
	latency := time.Duration(float64(100*time.Millisecond) * (1 + loadFactor))
	select {
	case <-time.After(latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Higher chance of failure under high load
	errorRate := 0.1 * (1 + loadFactor)
	
	// Use simple pseudo-random for demo purposes
	if time.Now().UnixNano()%100 < int64(errorRate*100) {
		m.recordFailure()
		return fmt.Errorf("%s: service error under load %.2f", m.Name, loadFactor)
	}

	m.recordSuccess()
	return nil
}

// recordSuccess increments the success count
func (m *Microservice) recordSuccess() {
	m.Health.mu.Lock()
	defer m.Health.mu.Unlock()
	m.Health.successes++
}

// recordFailure increments the failure count
func (m *Microservice) recordFailure() {
	m.Health.mu.Lock()
	defer m.Health.mu.Unlock()
	m.Health.failures++
	m.Health.lastFailure = time.Now()
}

// OnEvent subscribes an event handler to this microservice
func (m *Microservice) OnEvent(handler events.EventHandler) {
	m.eventBus.Subscribe(handler)
}

// MetricsSnapshot represents current service metrics
type MetricsSnapshot struct {
	TotalRequests   int
	SuccessfulCalls int
	FailedCalls     int
	AverageLatency  time.Duration
}

// GetMetrics returns current service metrics
func (m *Microservice) GetMetrics() MetricsSnapshot {
	m.Health.mu.RLock()
	defer m.Health.mu.RUnlock()
	
	total := m.Health.successes + m.Health.failures
	avgLatency := time.Duration(0)
	if len(m.Health.responseTimes) > 0 {
		var sum time.Duration
		for _, t := range m.Health.responseTimes {
			sum += t
		}
		avgLatency = sum / time.Duration(len(m.Health.responseTimes))
	}
	
	return MetricsSnapshot{
		TotalRequests:   total,
		SuccessfulCalls: m.Health.successes,
		FailedCalls:     m.Health.failures,
		AverageLatency:  avgLatency,
	}
}

// Process executes a function within the service context
func (m *Microservice) Process(ctx context.Context, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	// Record response time
	m.Health.mu.Lock()
	m.Health.responseTimes = append(m.Health.responseTimes, duration)
	// Keep only last 100 response times
	if len(m.Health.responseTimes) > 100 {
		m.Health.responseTimes = m.Health.responseTimes[1:]
	}
	m.Health.mu.Unlock()
	
	if err != nil {
		m.recordFailure()
		// Publish failure event
		m.eventBus.PublishEvent(events.Event{
			Type:      events.RequestFailed,
			ServiceID: m.Name,
			Timestamp: time.Now(),
			Duration:  duration,
			Error:     err,
		})
		return err
	}
	
	m.recordSuccess()
	// Publish success event
	m.eventBus.PublishEvent(events.Event{
		Type:      events.RequestCompleted,
		ServiceID: m.Name,
		Timestamp: time.Now(),
		Duration:  duration,
	})
	return nil
}
