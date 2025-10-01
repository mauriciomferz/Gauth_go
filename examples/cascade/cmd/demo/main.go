package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/events"
	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/mesh"
	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/resources"
)

// OrderService demonstrates a service with dependencies
type OrderService struct {
	mesh      *mesh.ServiceMesh
	service   *mesh.Service
	eventLogs []string
}

// NewOrderService creates a new order service with dependencies
func NewOrderService(m *mesh.ServiceMesh) *OrderService {
	svc := &OrderService{
		mesh:      m,
		eventLogs: make([]string, 0),
	}

	// Configure the service with type-safe config
	config := resources.ServiceConfig{
		Name:    "order-service",
		Type:    resources.OrderService,
		Version: "1.0.0",
		Dependencies: []resources.ServiceType{
			resources.PaymentService,
			resources.InventoryService,
		},
		CircuitBreaker: resources.CircuitBreakerConfig{
			ErrorThreshold: 5,
			ResetTimeout:   time.Minute,
		},
		RateLimit: resources.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         20,
		},
		MaxConcurrency: 50,
		Timeout:        30 * time.Second,
	}

	// Add service to mesh
	svc.service = m.AddService(config)

	// Subscribe to events
	svc.service.OnEvent(&EventHandler{svc})

	return svc
}

// EventHandler implements events.EventHandler interface
type EventHandler struct {
	svc *OrderService
}

func (h *EventHandler) Handle(e events.Event) {
	// Log event with available data
	log := fmt.Sprintf("[%s] %s: ", e.Timestamp.Format(time.RFC3339), e.ServiceID)

	switch e.Type {
	case events.CircuitOpened:
		log += "Circuit opened"
	case events.RateLimitExceeded:
		log += "Rate limit exceeded"
	case events.RequestCompleted:
		log += fmt.Sprintf("Request completed in %v", e.Duration)
	case events.RequestFailed:
		log += fmt.Sprintf("Request failed: %v", e.Error)
	default:
		log += fmt.Sprintf("Event: %v", e.Type)
	}

	h.svc.eventLogs = append(h.svc.eventLogs, log)
	fmt.Println(log)
}

func main() {
	// Create service mesh with monitoring
	serviceMesh := mesh.NewServiceMesh()

	// Create services
	orderSvc := NewOrderService(serviceMesh)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Simulate requests
	processOrder := func() error {
		// Simulated order processing
		time.Sleep(100 * time.Millisecond)
		if time.Now().Unix()%7 == 0 { // Occasional failure
			return fmt.Errorf("simulated order processing error")
		}
		return nil
	}

	// Process orders with resilience patterns
	for i := 0; ; i++ {
		select {
		case <-ctx.Done(): // Print final metrics
			metrics := orderSvc.service.GetMetrics()
			fmt.Printf("\nFinal Metrics:\n")
			if totalReq, ok := metrics["requests_processed"].(int); ok {
				fmt.Printf("Total Requests: %d\n", totalReq)
			}
			if loadFactor, ok := metrics["load_factor"].(float64); ok {
				fmt.Printf("Load Factor: %.2f\n", loadFactor)
			}
			if errors, ok := metrics["errors"].(int); ok {
				fmt.Printf("Errors: %d\n", errors)
			}
			return
		default:
			// Process order with resilience patterns
			if err := orderSvc.service.Process(ctx, processOrder); err != nil {
				log.Printf("Order %d failed: %v\n", i+1, err)
			} else {
				log.Printf("Order %d processed successfully\n", i+1)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
