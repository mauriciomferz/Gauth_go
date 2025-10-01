package mesh

// Package mesh provides the core service mesh implementation.

import (
	"context"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/events"
	"github.com/Gimel-Foundation/gauth/pkg/resources"
)

// ServiceMesh coordinates services and their interactions
type ServiceMesh struct {
	services map[resources.ServiceType]*Service
	config   Config
	eventBus *events.EventBus
	mu       sync.RWMutex
}

// Config represents mesh-wide configuration
type Config struct {
	Name           string
	MetricsEnabled bool
	TracingEnabled bool
	DefaultTimeout time.Duration
}

// Service represents a service in the mesh with resilience patterns
type Service struct {
	config   resources.ServiceConfig
	mesh     *ServiceMesh
	metrics  resources.ServiceMetrics
	eventBus *events.EventBus
	mu       sync.RWMutex
}

// New creates a new service mesh
func New(config Config) *ServiceMesh {
	return &ServiceMesh{
		services: make(map[resources.ServiceType]*Service),
		config:   config,
		eventBus: events.NewEventBus(),
	}
}

// AddService adds a new service to the mesh
func (m *ServiceMesh) AddService(config resources.ServiceConfig) *Service {
	m.mu.Lock()
	defer m.mu.Unlock()

	service := &Service{
		config:   config,
		mesh:     m,
		eventBus: m.eventBus,
	}

	m.services[config.Type] = service
	return service
}

// Process executes a request with configured resilience patterns
func (s *Service) Process(_ context.Context, action func() error) error {
	start := time.Now()

	// Execute action
	err := action()

	// Record metrics
	s.mu.Lock()
	s.metrics.TotalRequests++
	if err == nil {
		s.metrics.SuccessfulCalls++
	} else {
		s.metrics.FailedCalls++
		s.metrics.LastFailureTime = time.Now()
	}
	s.updateLatencyMetrics(time.Since(start))
	s.mu.Unlock()

	// Publish event
	meta := events.NewMetadata()
	meta.SetString("duration", time.Since(start).String())
	event := events.Event{
		Type:      events.EventTypeSystem,
		Timestamp: time.Now(),
		Resource:  s.config.Name,
		Metadata:  meta,
	}
	if err != nil {
		event.Error = err.Error()
	}
	s.eventBus.Publish(event)

	return err
}

// OnEvent subscribes an event handler
func (s *Service) OnEvent(handler events.EventHandler) {
	s.eventBus.Subscribe(handler)
}

// GetMetrics returns current service metrics
func (s *Service) GetMetrics() resources.ServiceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

func (s *Service) updateLatencyMetrics(duration time.Duration) {
	s.metrics.AverageLatency = time.Duration(
		(int64(s.metrics.AverageLatency)*(s.metrics.SuccessfulCalls-1) +
			int64(duration)) / s.metrics.SuccessfulCalls,
	)
}
