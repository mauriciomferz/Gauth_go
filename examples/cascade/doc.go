// Package cascade demonstrates resilience patterns for preventing cascading failures
// in microservice architectures. This example shows how to implement and combine multiple
// resilience techniques to create robust authentication systems.
//
// The example demonstrates:
//
//  1. Circuit Breaking: Failing fast when dependencies are unhealthy
//  2. Retry with Backoff: Intelligently retrying failed operations
//  3. Rate Limiting: Protecting services from overload
//  4. Fallback Mechanisms: Providing degraded functionality when dependencies fail
//  5. Health Checking: Monitoring and responding to service health
//  6. Timeout Management: Preventing resource exhaustion from slow responses
//  7. Bulkhead Pattern: Isolating failures to prevent system-wide cascades
//  8. Graceful Degradation: Maintaining core functionality during partial outages
//
// # Core Components
//
// The framework consists of several key components:
//
//  1. Service Mesh Integration:
//     - Coordinates service interactions
//     - Manages dependencies
//     - Monitors service health
//     - Simulates varying load conditions
//
//  2. Resilience Patterns (/internal/resilience):
//     - Circuit Breaker: Prevents cascading failures
//     - Rate Limiter: Controls request rates
//     - Bulkhead: Isolates failures
//     - Retry: Handles transient failures
//
// # Basic Usage
//
// To use the basic service mesh:
//
//	import "cascade/pkg/mesh"
//
//	// Create a new service mesh
//	serviceMesh := mesh.NewServiceMesh()
//
//	// Configure service load factors
//	serviceMesh.SetServiceLoad(mesh.PaymentService, 0.5)
//
//	// Process requests through the mesh
//	err := serviceMesh.ProcessRequest(ctx, mesh.OrderService)
//
// # Advanced Usage
//
// For more complex scenarios, you can configure individual resilience patterns:
//
//	// Configure circuit breaker
//	breaker := resilience.NewCircuitBreaker(
//	    "payment",
//	    5,           // failure threshold
//	    10*time.Second, // reset timeout
//	)
//
//	// Configure rate limiter
//	limiter := resilience.NewRateLimiter(
//	    100, // requests per second
//	    20,  // burst size
//	)
//
//	// Configure bulkhead
//	bulkhead := resilience.NewBulkhead(10) // concurrent requests
//
// Design Principles
//
//  1. Clear Separation of Concerns:
//     - Each package has a focused responsibility
//     - Public APIs are well-documented
//     - Implementation details are hidden
//
//  2. Type Safety:
//     - Strong typing with concrete types instead of interface{}
//     - Event types for all service events
//     - Structured metadata instead of generic maps
//     - Clear error hierarchies and custom error types
//
//  3. Modularity:
//     - Independent, focused packages
//     - Clean interfaces for extensibility
//     - Pluggable implementations for each pattern
//     - Clear boundaries between public and internal APIs
//
//  4. Testability:
//     - Comprehensive unit tests
//     - Integration test suites for patterns
//     - Performance benchmarks
//     - Failure scenario simulations
//
// For more information, see the documentation in /docs and examples in /examples.
package main
