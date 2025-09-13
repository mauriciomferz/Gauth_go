package main

// Package main provides examples of using the cascade service mesh framework with
// type-safe resilience patterns.
//
// These examples demonstrate:
//
// 1. Service Configuration
//   - Type-safe configuration using concrete types
//   - Clear dependency specifications
//   - Proper resource limits and timeouts
//
// 2. Resilience Patterns
//   - Circuit Breaker pattern with proper state transitions
//   - Rate Limiting with precise control
//   - Bulkhead pattern for resource isolation
//   - Event-driven monitoring
//
// 3. Event Handling
//   - Type-safe event system
//   - Strongly typed event data
//   - Real-time monitoring
//   - Metrics collection
//
// Example Usage:
//
//	mesh := mesh.New(mesh.Config{
//	    Name: "demo-mesh",
//	    MetricsEnabled: true,
//	})
//
//	// Configure service with type-safe config
//	config := resources.ServiceConfig{
//	    Type: resources.OrderService,
//	    Name: "order-service",
//	    CircuitBreaker: resources.CircuitBreakerConfig{
//	        ErrorThreshold: 5,
//	        ResetTimeout:   time.Minute,
//	    },
//	}
//
//	// Add service to mesh
//	service := mesh.AddService(config)
//
//	// Process requests with resilience patterns
//	err := service.Process(ctx, func() error {
//	    return processOrder()
//	})
//
// The demo shows how to:
//   - Configure services with proper types
//   - Handle events type-safely
//   - Monitor service health
//   - Collect metrics
//   - Implement graceful shutdown
//
// Run the demo:
//
//	go run cmd/demo/main.go
