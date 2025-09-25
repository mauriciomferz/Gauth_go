package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cascade/pkg/events"
	"cascade/pkg/mesh"
	"cascade/pkg/resources"
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
	// Log event with type-safe data
	log := fmt.Sprintf("[%s] %s: ", e.Timestamp.Format(time.RFC3339), e.Service)

	switch e.Type {
	case events.CircuitOpened:
		data := e.Data.(events.CircuitBreakerData)
		log += fmt.Sprintf("Circuit opened after %d failures", data.FailureCount)
	case events.RateLimitExceeded:
		data := e.Data.(events.RateLimitData)
		log += fmt.Sprintf("Rate limit exceeded: %.2f/s (limit: %.2f/s)", data.CurrentRate, data.Limit)
	case events.RequestCompleted:
		data := e.Data.(events.RequestData)
		log += fmt.Sprintf("Request completed in %v", data.Duration)
	case events.RequestFailed:
		data := e.Data.(events.RequestData)
		log += fmt.Sprintf("Request failed: %v", data.Error)
	}

	h.svc.eventLogs = append(h.svc.eventLogs, log)
	fmt.Println(log)
}

func main() {
	// Create service mesh with monitoring
	mesh := mesh.New(mesh.Config{
		Name:           "demo-mesh",
		MetricsEnabled: true,
		TracingEnabled: true,
		DefaultTimeout: 30 * time.Second,
	})

	// Create services
	orderSvc := NewOrderService(mesh)

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
		case <-ctx.Done():
			// Print final metrics
			metrics := orderSvc.service.GetMetrics()
			fmt.Printf("\nFinal Metrics:\n")
			fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
			fmt.Printf("Success Rate: %.2f%%\n", float64(metrics.SuccessfulCalls)/float64(metrics.TotalRequests)*100)
			fmt.Printf("Average Latency: %v\n", metrics.AverageLatency)
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
