package mesh

import (
	"context"
	"fmt"
	"time"
)

// ServiceType represents different service types in the mesh
type ServiceType string

const (
	PaymentService   ServiceType = "payment"
	OrderService     ServiceType = "order"
	InventoryService ServiceType = "inventory"
)

// ServiceMesh represents a service mesh managing multiple services and their loads
type ServiceMesh struct {
	services map[string]*Service
}

// Service represents a service in the mesh, with a name and load factor
type Service struct {
	name string
	load float64
}

// NewServiceMesh creates a new service mesh instance
func NewServiceMesh() *ServiceMesh {
	return &ServiceMesh{
		services: make(map[string]*Service),
	}
}

// SetServiceLoad sets the load factor for a service
func (sm *ServiceMesh) SetServiceLoad(serviceName string, load float64) error {
	if service, exists := sm.services[serviceName]; exists {
		service.load = load
	} else {
		sm.services[serviceName] = &Service{name: serviceName, load: load}
	}
	return nil
}

// GetService gets a service by name, creating it if it doesn't exist
func (sm *ServiceMesh) GetService(serviceName string) (*Service, error) {
	if service, exists := sm.services[serviceName]; exists {
		return service, nil
	}
	// Create service if it doesn't exist
	service := &Service{name: serviceName, load: 0.0}
	sm.services[serviceName] = service
	return service, nil
}

// ProcessRequest simulates processing a request for this service, with delay based on load
func (s *Service) ProcessRequest(ctx context.Context, mesh *ServiceMesh) (interface{}, error) {
	// Simulate processing time based on load
	processingTime := time.Duration(s.load*100) * time.Millisecond
	time.Sleep(processingTime)

	return fmt.Sprintf("Request processed by %s", s.name), nil
}

// PrintHealthReport prints a health report for all services, showing load and status
func (sm *ServiceMesh) PrintHealthReport() {
	fmt.Println("=== Service Mesh Health Report ===")
	for name, service := range sm.services {
		status := "HEALTHY"
		if service.load > 0.8 {
			status = "OVERLOADED"
		} else if service.load > 0.5 {
			status = "HIGH_LOAD"
		}
		fmt.Printf("Service: %s, Load: %.2f, Status: %s\n", name, service.load, status)
	}
	fmt.Println("===================================")
}
