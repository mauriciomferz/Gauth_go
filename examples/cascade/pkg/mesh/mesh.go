package mesh

import (
	"context"
	"fmt"
	"sync"
)

// ServiceType represents different types of services
type ServiceType string

const (
	PaymentService   ServiceType = "payment"
	OrderService     ServiceType = "order"
	InventoryService ServiceType = "inventory"
)

// Service represents a service in the mesh
type Service struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Type       ServiceType `json:"type"`
	URL        string      `json:"url"`
	LoadFactor float64     `json:"load_factor"`
}

// SetLoadFactor sets the load factor for the service
func (s *Service) SetLoadFactor(factor float64) {
	s.LoadFactor = factor
}

// ProcessRequest processes a request for the service
func (s *Service) ProcessRequest(ctx context.Context, data interface{}) (interface{}, error) {
	// Mock implementation for demo purposes
	return fmt.Sprintf("Processed request for %s service", s.Type), nil
}

// OnEvent handles events for the service with proper EventHandler interface
func (s *Service) OnEvent(handler interface{}) {
	// Mock implementation for demo purposes - accepts any handler
	fmt.Printf("Service %s registered event handler\n", s.Name)
}

// GetMetrics returns service metrics (mock for demo)
func (s *Service) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"requests_processed": 42,
		"errors":             0,
		"load_factor":        s.LoadFactor,
	}
}

// Process processes a request (alias for ProcessRequest)
func (s *Service) Process(ctx context.Context, data interface{}) error {
	_, err := s.ProcessRequest(ctx, data)
	return err
}

// ServiceMesh manages the service mesh
type ServiceMesh struct {
	services map[string]*Service
	mu       sync.RWMutex
}

// NewServiceMesh creates a new service mesh
func NewServiceMesh() *ServiceMesh {
	return &ServiceMesh{
		services: make(map[string]*Service),
	}
}

// RegisterService registers a service in the mesh
func (sm *ServiceMesh) RegisterService(service *Service) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.services[service.ID] = service
	return nil
}

// GetService retrieves a service by ID
func (sm *ServiceMesh) GetService(id string) (*Service, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	service, exists := sm.services[id]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", id)
	}
	return service, nil
}

// CallService makes a call to a service
func (sm *ServiceMesh) CallService(ctx context.Context, serviceID string, data interface{}) (interface{}, error) {
	service, err := sm.GetService(serviceID)
	if err != nil {
		return nil, err
	}

	// Simulate service call
	return fmt.Sprintf("Called %s service at %s", service.Name, service.URL), nil
}

// PrintHealthReport prints a health report of all services
func (sm *ServiceMesh) PrintHealthReport() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	fmt.Println("Service Mesh Health Report:")
	for id, service := range sm.services {
		fmt.Printf("  Service %s (%s): %s - %s\n", id, service.Type, service.Name, service.URL)
	}
}

// SetServiceLoad sets the load for a service (for demo purposes)
func (sm *ServiceMesh) SetServiceLoad(serviceType string, load float64) error {
	// Find service by type
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, service := range sm.services {
		if string(service.Type) == serviceType {
			service.LoadFactor = load
			fmt.Printf("Setting load %.2f for service type %s\n", load, serviceType)
			return nil
		}
	}

	return fmt.Errorf("service type not found: %s", serviceType)
}

// AddService adds a service to the mesh and returns it (used by demo)
func (sm *ServiceMesh) AddService(config interface{}) *Service {
	// Create a service based on config
	service := &Service{
		ID:         "demo-service",
		Name:       "Demo Service",
		Type:       OrderService,
		URL:        "http://localhost:8080/demo",
		LoadFactor: 0.0,
	}

	// Handle potential error from RegisterService
	if err := sm.RegisterService(service); err != nil {
		// Log error but continue - service creation shouldn't fail due to registration issues
		fmt.Printf("Warning: failed to register service %s: %v\n", service.Name, err)
	}
	return service
}

// NewMicroservice creates a new microservice
func NewMicroservice(serviceType ServiceType, name string, config interface{}) *Service {
	return &Service{
		ID:   fmt.Sprintf("%s-%s", name, serviceType),
		Name: name,
		Type: serviceType,
		URL:  fmt.Sprintf("http://localhost:8080/%s", name),
	}
}

// MeshPackageInit initializes the mesh package
func MeshPackageInit() string {
	return "mesh package initialized"
}
