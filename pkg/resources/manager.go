package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/metrics"
)

// Manager handles resource lifecycle and monitoring
type Manager struct {
	services    sync.Map // map[ServiceType]*ServiceState
	metrics   *metrics.Collector
	configStore ConfigStore
	mu          sync.RWMutex
}

// ServiceState tracks the current state of a service
type ServiceState struct {
	Config     ServiceConfig
	Metrics    ServiceMetrics
	LastUpdate time.Time
	LastCheck  time.Time
	Dependents []ServiceType
	Status     ServiceStatus
	UpdateChan chan struct{}
}

// ConfigStore persists service configurations
type ConfigStore interface {
	// Load loads service configuration
	Load(ctx context.Context, serviceType ServiceType) (*ServiceConfig, error)

	// Save persists service configuration
	Save(ctx context.Context, config ServiceConfig) error

	// List returns all service configurations
	List(ctx context.Context) ([]ServiceConfig, error)

	// Watch observes configuration changes
	Watch(ctx context.Context) (<-chan ServiceConfig, error)
}

// NewManager creates a new resource manager
func NewManager(store ConfigStore, metrics *metrics.Collector) *Manager {
	m := &Manager{
		configStore: store,
		metrics:     metrics,
	}

	return m
}

// RegisterService registers a new service
func (m *Manager) RegisterService(ctx context.Context, config ServiceConfig) error {
	if config.Type == "" {
		return errors.New("service type is required")
	}

	if _, exists := m.services.LoadOrStore(config.Type, &ServiceState{
		Config:     config,
		Status:     StatusHealthy,
		UpdateChan: make(chan struct{}, 1),
	}); exists {
		return fmt.Errorf("service %s already registered", config.Type)
	}

	// Save configuration
	if err := m.configStore.Save(ctx, config); err != nil {
		m.services.Delete(config.Type)
		return fmt.Errorf("failed to save service config: %w", err)
	}

	// Register dependencies
	if err := m.registerDependencies(config); err != nil {
		m.services.Delete(config.Type)
		return fmt.Errorf("failed to register dependencies: %w", err)
	}

	return nil
}

// UpdateService updates service configuration
func (m *Manager) UpdateService(ctx context.Context, config ServiceConfig) error {
	stateVal, ok := m.services.Load(config.Type)
	if !ok {
		return fmt.Errorf("service %s not found", config.Type)
	}

	state := stateVal.(*ServiceState)
	m.mu.Lock()
	oldConfig := state.Config
	state.Config = config
	state.LastUpdate = time.Now()
	m.mu.Unlock()

	// Update configuration store
	if err := m.configStore.Save(ctx, config); err != nil {
		// Rollback
		state.Config = oldConfig
		return fmt.Errorf("failed to save service config: %w", err)
	}

	// Notify dependents
	select {
	case state.UpdateChan <- struct{}{}:
	default:
	}

	return nil
}

// GetService retrieves service information
func (m *Manager) GetService(serviceType ServiceType) (*ServiceState, error) {
	if stateVal, ok := m.services.Load(serviceType); ok {
		return stateVal.(*ServiceState), nil
	}
	return nil, fmt.Errorf("service %s not found", serviceType)
}

// ListServices returns all registered services
func (m *Manager) ListServices() []ServiceState {
	var services []ServiceState
	m.services.Range(func(_, value interface{}) bool {
		services = append(services, *value.(*ServiceState))
		return true
	})
	return services
}

// UpdateMetrics updates service metrics
func (m *Manager) UpdateMetrics(serviceType ServiceType, metrics ServiceMetrics) error {
	stateVal, ok := m.services.Load(serviceType)
	if !ok {
		return fmt.Errorf("service %s not found", serviceType)
	}

	state := stateVal.(*ServiceState)
	m.mu.Lock()
	state.Metrics = metrics
	m.mu.Unlock()

	// Record metrics if metrics collector is available
	if m.metrics != nil {
		m.recordMetrics(serviceType, metrics)
	}

	return nil
}

// UpdateStatus updates service status
func (m *Manager) UpdateStatus(serviceType ServiceType, status ServiceStatus) error {
	stateVal, ok := m.services.Load(serviceType)
	if !ok {
		return fmt.Errorf("service %s not found", serviceType)
	}

	state := stateVal.(*ServiceState)
	m.mu.Lock()
	oldStatus := state.Status
	state.Status = status
	state.LastCheck = time.Now()
	m.mu.Unlock()

	// Check dependent services
	if oldStatus != status {
		m.checkDependentServices(serviceType)
	}

	return nil
}

// WatchService watches for service updates
func (m *Manager) WatchService(serviceType ServiceType) (<-chan struct{}, error) {
	stateVal, ok := m.services.Load(serviceType)
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceType)
	}

	return stateVal.(*ServiceState).UpdateChan, nil
}

// CheckHealth checks service health including dependencies
func (m *Manager) CheckHealth(serviceType ServiceType) error {
	state, err := m.GetService(serviceType)
	if err != nil {
		return err
	}

	if state.Status != StatusHealthy {
		return fmt.Errorf("service %s is %s", serviceType, state.Status)
	}

	// Check dependencies
	for _, dep := range state.Config.Dependencies {
		if err := m.CheckHealth(dep); err != nil {
			return fmt.Errorf("dependency %s is unhealthy: %w", dep, err)
		}
	}

	return nil
}

func (m *Manager) registerDependencies(config ServiceConfig) error {
	for _, depType := range config.Dependencies {
		if depVal, ok := m.services.Load(depType); ok {
			dep := depVal.(*ServiceState)
			dep.Dependents = append(dep.Dependents, config.Type)
		} else {
			// Try to load dependency configuration
			depConfig, err := m.configStore.Load(context.Background(), depType)
			if err != nil {
				return fmt.Errorf("dependency %s not found", depType)
			}

			if err := m.RegisterService(context.Background(), *depConfig); err != nil {
				return fmt.Errorf("failed to register dependency %s: %w", depType, err)
			}
		}
	}
	return nil
}

func (m *Manager) checkDependentServices(serviceType ServiceType) {
	if stateVal, ok := m.services.Load(serviceType); ok {
		state := stateVal.(*ServiceState)
		for _, depType := range state.Dependents {
			if depVal, ok := m.services.Load(depType); ok {
				dep := depVal.(*ServiceState)
				select {
				case dep.UpdateChan <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (m *Manager) recordMetrics(serviceType ServiceType, metrics ServiceMetrics) {
	// Record request metrics
	m.metrics.RecordValue(fmt.Sprintf("service.%s.requests.total", serviceType), float64(metrics.TotalRequests))
	m.metrics.RecordValue(fmt.Sprintf("service.%s.requests.success", serviceType), float64(metrics.SuccessfulCalls))
	m.metrics.RecordValue(fmt.Sprintf("service.%s.requests.failed", serviceType), float64(metrics.FailedCalls))
	m.metrics.RecordValue(fmt.Sprintf("service.%s.latency.average", serviceType), metrics.AverageLatency.Seconds())
	m.metrics.RecordValue(fmt.Sprintf("service.%s.latency.p95", serviceType), metrics.P95Latency.Seconds())
	m.metrics.RecordValue(fmt.Sprintf("service.%s.latency.p99", serviceType), metrics.P99Latency.Seconds())

	// Record circuit breaker metrics
	m.metrics.RecordValue(fmt.Sprintf("service.%s.circuit.error_rate", serviceType), metrics.ErrorRate)

	// Record rate limiting metrics
	m.metrics.RecordValue(fmt.Sprintf("service.%s.rate.current", serviceType), metrics.CurrentRate)
	m.metrics.RecordValue(fmt.Sprintf("service.%s.rate.rejected", serviceType), float64(metrics.RejectedRequests))

	// Record resource metrics
	m.metrics.RecordValue(fmt.Sprintf("service.%s.resources.active", serviceType), float64(metrics.ActiveRequests))
	m.metrics.RecordValue(fmt.Sprintf("service.%s.resources.queued", serviceType), float64(metrics.QueuedRequests))
	m.metrics.RecordValue(fmt.Sprintf("service.%s.resources.usage", serviceType), metrics.ResourceUsage)
}
