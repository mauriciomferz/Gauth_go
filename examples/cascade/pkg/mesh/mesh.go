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
	ID   string      `json:"id"`
	Name string      `json:"name"`
	Type ServiceType `json:"type"`
	URL  string      `json:"url"`
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
func (sm *ServiceMesh) SetServiceLoad(serviceID string, load float64) error {
	_, err := sm.GetService(serviceID)
	if err != nil {
		return err
	}
	
	// Simulate setting service load
	fmt.Printf("Setting load %.2f for service %s\n", load, serviceID)
	return nil
}

// NewMicroservice creates a new microservice
func NewMicroservice(name string, serviceType ServiceType) *Service {
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
