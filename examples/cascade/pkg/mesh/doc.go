// Package mesh provides a service mesh implementation for demonstrating resilience patterns
// in microservices architectures. It includes implementations of several key resilience
// patterns including circuit breakers, rate limiting, bulkheads, and retry mechanisms.
//
// The package is structured around these key concepts:
//
// ServiceMesh: The top-level coordinator that manages service interactions and dependencies.
// It provides a framework for demonstrating how failures can cascade through a system and
// how different resilience patterns can work together to prevent system-wide failures.
//
// Microservice: Represents an individual service in the mesh, equipped with its own
// resilience patterns and health monitoring. Each service can be configured with different
// failure rates and load characteristics.
//
// Health Monitoring: Each service maintains its own health metrics, allowing for real-time
// monitoring of system behavior under different load conditions.
//
// Resilience Patterns:
//   - Circuit Breaker: Prevents cascading failures by stopping calls to failing services
//   - Rate Limiting: Controls the rate of requests to prevent overload
//   - Bulkhead: Isolates failures by partitioning service resources
//   - Retry: Handles transient failures through configurable retry strategies
//
// Example usage:
//
//	mesh := mesh.NewServiceMesh()
//	ctx := context.Background()
//
//	// Configure service load factors
//	mesh.SetServiceLoad(mesh.PaymentService, 0.5)
//
//	// Process a request through the mesh
//	err := mesh.ProcessRequest(ctx, mesh.OrderService)
//	if err != nil {
//	    log.Printf("Request failed: %v", err)
//	}
//
// The package is designed to be extensible, allowing for new services and resilience
// patterns to be added easily while maintaining a clear separation of concerns.
package mesh
