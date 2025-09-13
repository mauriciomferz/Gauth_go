package mesh

import (
	"fmt"
	"sync"
)

// ServiceMesh coordinates all services and their interactions
type ServiceMesh struct {
	services map[ServiceType]*Microservice
	graph    *DependencyGraph
	mu       sync.RWMutex
}

// NewServiceMesh creates a new service mesh with predefined services
func NewServiceMesh() *ServiceMesh {
	mesh := &ServiceMesh{
		services: make(map[ServiceType]*Microservice),
		graph:    &DependencyGraph{dependencies: make(map[ServiceType][]ServiceType)},
	}

	// Define service dependencies
	mesh.graph.dependencies[OrderService] = []ServiceType{AuthService, UserService, InventoryService, PaymentService}
	mesh.graph.dependencies[PaymentService] = []ServiceType{AuthService, UserService}
	mesh.graph.dependencies[LogisticsService] = []ServiceType{OrderService, InventoryService}
	mesh.graph.dependencies[NotificationService] = []ServiceType{UserService}

	// Create services with dependencies
	mesh.addService(AuthService, "Auth", nil)
	mesh.addService(UserService, "User", nil)
	mesh.addService(OrderService, "Order", mesh.graph.dependencies[OrderService])
	mesh.addService(InventoryService, "Inventory", nil)
	mesh.addService(PaymentService, "Payment", mesh.graph.dependencies[PaymentService])
	mesh.addService(NotificationService, "Notification", mesh.graph.dependencies[NotificationService])
	mesh.addService(LogisticsService, "Logistics", mesh.graph.dependencies[LogisticsService])

	return mesh
}

// addService creates and adds a new service to the mesh
func (m *ServiceMesh) addService(sType ServiceType, name string, deps []ServiceType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[sType] = NewMicroservice(sType, name, deps)
}

// SetServiceLoad updates the load factor for a specific service
func (m *ServiceMesh) SetServiceLoad(sType ServiceType, factor float64) error {
	m.mu.RLock()
	svc, exists := m.services[sType]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service %v not found", sType)
	}

	svc.SetLoadFactor(factor)
	return nil
}

// GetServiceHealth returns the health metrics for a service
func (m *ServiceMesh) GetServiceHealth(sType ServiceType) (*HealthMetrics, error) {
	m.mu.RLock()
	svc, exists := m.services[sType]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service %v not found", sType)
	}

	return svc.Health, nil
}

// PrintHealthReport generates a comprehensive health report for all services
func (m *ServiceMesh) PrintHealthReport() {
	fmt.Println("\nService Health Report:")
	fmt.Println("-----------------------------")

	m.mu.RLock()
	defer m.mu.RUnlock()

	for sType, svc := range m.services {
		svc.Health.mu.RLock()
		total := svc.Health.Successes + svc.Health.Failures
		successRate := 100.0
		if total > 0 {
			successRate = float64(svc.Health.Successes) / float64(total) * 100
		}
		svc.Health.mu.RUnlock()

		fmt.Printf("%s:\n", sType)
		fmt.Printf("  - Success Rate: %.2f%%\n", successRate)
		fmt.Printf("  - Load Factor: %.2f\n", svc.GetLoadFactor())
	}
}

// GetService returns a service by type
func (m *ServiceMesh) GetService(sType ServiceType) (*Microservice, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	svc, exists := m.services[sType]
	return svc, exists
}
