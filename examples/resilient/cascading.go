package resilient

import (
	"context"
	"fmt"
	"crypto/rand"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
)

// ServiceType represents different types of microservices
type ServiceType int

const (
	AuthService ServiceType = iota
	UserService
	OrderService
	InventoryService
	PaymentService
	NotificationService
	LogisticsService
)

// DependencyGraph represents service dependencies
type DependencyGraph struct {
	dependencies map[ServiceType][]ServiceType
}

// Microservice represents a service in the system
type Microservice struct {
	Type         ServiceType
	Name         string
	Dependencies []ServiceType
	Health       *HealthMetrics
	Breaker      *circuit.CircuitBreaker
	Limiter      ratelimit.Algorithm
	Retry        *resilience.Retry
	Bulkhead     *resilience.Bulkhead
	LoadFactor   float64 // 0-1, affects service performance
	mu           sync.RWMutex
}

// HealthMetrics tracks service health
type HealthMetrics struct {
	Failures        int
	Successes       int
	ResponseTimes   []time.Duration
	LastFailureTime time.Time
	mu              sync.RWMutex
}

// ServiceMesh coordinates all services
type ServiceMesh struct {
	services map[ServiceType]*Microservice
	graph    *DependencyGraph
}

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

	// Create services with different configurations
	mesh.addService(AuthService, "Auth", 50*time.Millisecond, 0.05)
	mesh.addService(UserService, "User", 100*time.Millisecond, 0.1)
	mesh.addService(OrderService, "Order", 200*time.Millisecond, 0.15)
	mesh.addService(InventoryService, "Inventory", 150*time.Millisecond, 0.1)
	mesh.addService(PaymentService, "Payment", 300*time.Millisecond, 0.2)
	mesh.addService(NotificationService, "Notification", 80*time.Millisecond, 0.05)
	mesh.addService(LogisticsService, "Logistics", 250*time.Millisecond, 0.15)

	return mesh
}

func (mesh *ServiceMesh) addService(sType ServiceType, name string, baseLatency time.Duration, baseErrorRate float64) {
	svc := &Microservice{
		Type:         sType,
		Name:         name,
		Dependencies: mesh.graph.dependencies[sType],
		Health:       &HealthMetrics{},
		LoadFactor:   0.0,
	}

	// Configure resilience patterns
	svc.Breaker = circuit.NewCircuitBreaker(circuit.Options{
		Name:             name,
		FailureThreshold: 5,
		ResetTimeout:     10 * time.Second,
		HalfOpenLimit:    2,
		OnStateChange: func(name string, from, to circuit.State) {
			fmt.Printf("[%s] Circuit state changed: %s -> %s\n", name, from, to)
		},
	})

	svc.Limiter = ratelimit.WrapTokenBucket(&ratelimit.Config{
		RequestsPerSecond: 100,
		WindowSize:        1,
		BurstSize:         20,
	})

	svc.Retry = resilience.NewRetry(resilience.RetryStrategy{
		MaxAttempts:     3,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     500 * time.Millisecond,
		Multiplier:      2.0,
	})

	svc.Bulkhead = resilience.NewBulkhead(10)

	mesh.services[sType] = svc
}

func (s *Microservice) processRequest(ctx context.Context, mesh *ServiceMesh) error {
	s.mu.RLock()
	loadFactor := s.LoadFactor
	s.mu.RUnlock()

	// Check dependencies first
	for _, depType := range s.Dependencies {
		depSvc := mesh.services[depType]
		if err := depSvc.call(ctx, mesh); err != nil {
			return fmt.Errorf("%s dependency failed: %w", depSvc.Name, err)
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
	
	// Use crypto/rand for secure random generation
	var randomBytes [8]byte
	rand.Read(randomBytes[:])
	randomFloat := float64(randomBytes[0]) / 255.0 // Convert to 0-1 range
	
	if randomFloat < errorRate {
		s.recordFailure()
		return fmt.Errorf("%s: service error under load %.2f", s.Name, loadFactor)
	}

	s.recordSuccess()
	return nil
}

func (s *Microservice) call(ctx context.Context, mesh *ServiceMesh) error {
	// Apply bulkhead pattern
	return s.Bulkhead.Execute(ctx, func() error {
		// Check rate limit
		if err := s.Limiter.Allow(ctx, s.Name); err != nil {
			return fmt.Errorf("rate limit exceeded for %s: %w", s.Name, err)
		}

		// Use retry with circuit breaker
		return s.Retry.Execute(ctx, func() error {
			return s.Breaker.Execute(func() error {
				return s.processRequest(ctx, mesh)
			})
		})
	})
}

func (s *Microservice) recordSuccess() {
	s.Health.mu.Lock()
	defer s.Health.mu.Unlock()
	s.Health.Successes++
}

func (s *Microservice) recordFailure() {
	s.Health.mu.Lock()
	defer s.Health.mu.Unlock()
	s.Health.Failures++
	s.Health.LastFailureTime = time.Now()
}

func (s *Microservice) increaseLoad(factor float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LoadFactor = factor
}

func simulateCascadingFailures() {
	mesh := NewServiceMesh()
	ctx := context.Background()

	fmt.Println("\nStarting Cascading Failures Simulation...")
	fmt.Println("----------------------------------------")
	fmt.Println("Initial Configuration:")
	fmt.Println("- 7 interconnected services")
	fmt.Println("- Complex dependency graph")
	fmt.Println("- Circuit breakers, rate limits, and bulkheads")
	fmt.Println("----------------------------------------")

	// Channel for controlling load increase
	loadUpdates := make(chan struct{})

	// Start background load monitoring
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for _, svc := range mesh.services {
					svc.Health.mu.RLock()
					failureRate := float64(svc.Health.Failures) / float64(svc.Health.Successes+svc.Health.Failures)
					svc.Health.mu.RUnlock()

					fmt.Printf("[%s] Health: %.2f%% success rate\n",
						svc.Name, (1-failureRate)*100)
				}
			case <-loadUpdates:
				return
			}
		}
	}()

	// Simulate traffic with increasing load
	var wg sync.WaitGroup
	clients := 50
	phases := 4

	for phase := 1; phase <= phases; phase++ {
		fmt.Printf("\nPhase %d: Load Factor %.1f\n", phase, float64(phase)*0.25)

		// Increase load on critical services
		mesh.services[PaymentService].increaseLoad(float64(phase) * 0.25)
		mesh.services[OrderService].increaseLoad(float64(phase) * 0.2)
		mesh.services[InventoryService].increaseLoad(float64(phase) * 0.15)

		for client := 1; client <= clients; client++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()

				// Simulate complex transaction flow
				request := fmt.Sprintf("client%d-phase%d", clientID, phase)
				start := time.Now()

				// Start with order service which triggers dependency chain
				err := mesh.services[OrderService].call(ctx, mesh)
				duration := time.Since(start)

				if err != nil {
					fmt.Printf("[%s] Failed after %v: %v\n",
						request, duration.Round(time.Millisecond), err)
				} else {
					fmt.Printf("[%s] Completed in %v\n",
						request, duration.Round(time.Millisecond))
				}
			}(client)

			time.Sleep(100 * time.Millisecond)
		}

		wg.Wait()
		time.Sleep(2 * time.Second) // Pause between phases
	}

	close(loadUpdates)
	fmt.Println("\nCascading Failures simulation completed!")

	// Print final statistics
	fmt.Println("\nFinal Service Health Report:")
	fmt.Println("-----------------------------")
	for _, svc := range mesh.services {
		svc.Health.mu.RLock()
		total := svc.Health.Successes + svc.Health.Failures
		successRate := float64(svc.Health.Successes) / float64(total) * 100
		svc.Health.mu.RUnlock()

		fmt.Printf("%s:\n", svc.Name)
		fmt.Printf("  - Success Rate: %.2f%%\n", successRate)
		fmt.Printf("  - Load Factor: %.2f\n", svc.LoadFactor)
	}
}

func main() {
	simulateCascadingFailures()
}
