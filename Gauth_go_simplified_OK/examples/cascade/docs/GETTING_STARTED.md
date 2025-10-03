# Getting Started Guide

This guide will help you get started with using the service mesh library to implement resilience patterns in your own services.

## Basic Usage

### 1. Create a Simple Service

```go
package main

import (
    "context"
    "log"
    "time"

    "cascade/pkg/mesh"
    "cascade/pkg/events"
    "cascade/pkg/resilience"
)

// Define service-specific types for type safety
type OrderDetails struct {
    ID          string
    CustomerID  string
    Items       []OrderItem
    TotalAmount float64
}

type OrderItem struct {
    ProductID   string
    Quantity    int
    UnitPrice   float64
}

func main() {
    // Create a service mesh with typed configuration
    serviceMesh := mesh.New(mesh.Config{
        Name: "retail-mesh",
        Resilience: resilience.Config{
            DefaultTimeout: 30 * time.Second,
            RetryPolicy: resilience.RetryPolicy{
                MaxAttempts: 3,
                BackoffBase: 100 * time.Millisecond,
            },
        },
    })

    // Configure a service with strongly-typed events and configs
    orderService := serviceMesh.AddService(mesh.ServiceConfig{
        Type: mesh.OrderService,
        Dependencies: []mesh.Dependency{
            {Service: mesh.PaymentService, Required: true},
            {Service: mesh.InventoryService, Required: true},
        },
        CircuitBreaker: resilience.CircuitBreakerConfig{
            ErrorThreshold: 5,
            ResetTimeout:   time.Minute,
        },
        RateLimit: resilience.RateLimitConfig{
            RequestsPerSecond: 100,
            BurstSize:        20,
        },
    })

    // Use strongly-typed event handlers
    orderService.OnEvent(events.ServiceEvent{
        Type: events.CircuitOpened,
        Handler: func(e events.Event) {
            details := e.Data.(events.CircuitBreakerData)
            log.Printf("Circuit opened due to %d failures", details.FailureCount)
        },
    })

    // Process requests with type safety
    ctx := context.Background()
    order := OrderDetails{
        ID:         "order-123",
        CustomerID: "cust-456",
        Items: []OrderItem{
            {ProductID: "prod-789", Quantity: 2, UnitPrice: 29.99},
        },
    }
    
    err := orderService.Process(ctx, order)
    if err != nil {
        if resilience.IsCircuitOpenError(err) {
            log.Printf("Service unavailable: circuit open")
        } else {
            log.Printf("Request failed: %v", err)
        }
    }
}
```

### 2. Monitor Service Health

```go
// Get health metrics
snapshot := orderService.Health.GetSnapshot()
log.Printf("Success Rate: %.2f%%", snapshot.SuccessRate)
log.Printf("Average Latency: %v", snapshot.AverageLatency)

// Check service status
if snapshot.SuccessRate < 90.0 {
    log.Printf("Service degraded: %s", orderService.Name)
}
```

### 3. Handle Events

```go
// Subscribe to service events
orderService.Events().Subscribe(func(e events.Event) {
    switch e.Type {
    case events.CircuitOpened:
        log.Printf("Circuit breaker opened for %s", e.ServiceID)
    case events.RateLimitExceeded:
        log.Printf("Rate limit exceeded for %s", e.ServiceID)
    }
})
```

## Common Patterns

### Circuit Breaker Pattern

The circuit breaker prevents cascading failures by stopping calls to failing services:

```go
service := mesh.NewMicroservice(mesh.PaymentService, "Payment", nil)

// Circuit will open after 5 failures within 10 seconds
service.Config.CircuitBreaker = resources.CircuitBreakerConfig{
    FailureThreshold: 5,
    ResetTimeout:    10 * time.Second,
}
```

### Rate Limiting

Control the rate of requests to prevent overload:

```go
service.Config = resources.ServiceConfig{
    RequestsPerSecond: 100,  // Base rate
    BurstSize:        20,    // Allow bursts
    TimeoutSeconds:   30,    // Request timeout
}
```

### Bulkhead Pattern

Isolate failures by partitioning service resources:

```go
service.Config = resources.ServiceConfig{
    MaxConcurrent:  10,  // Maximum concurrent requests
    QueueSize:      50,  // Request queue size
}
```

## Best Practices

1. **Service Configuration**
   - Set reasonable timeouts for all services
   - Configure retry policies based on service characteristics
   - Use appropriate rate limits based on capacity

2. **Health Monitoring**
   - Monitor service health metrics regularly
   - Set up alerts for degraded services
   - Track dependencies' health status

3. **Error Handling**
   - Use structured error types
   - Include context in error messages
   - Handle timeouts appropriately

4. **Testing**
   - Test failure scenarios
   - Verify resilience patterns work as expected
   - Check dependency handling

## Advanced Topics

### Custom Event Handlers

Create custom event handlers for specific scenarios:

```go
type MetricsHandler struct {
    metrics map[string]*ServiceMetrics
}

func (h *MetricsHandler) Handle(e events.Event) {
    switch e.Type {
    case events.RequestCompleted:
        h.recordSuccess(e.ServiceID, e.Duration)
    case events.RequestFailed:
        h.recordFailure(e.ServiceID, e.Error)
    }
}
```

### Custom Resource Types

Define custom resource types for specific needs:

```go
type CustomServiceConfig struct {
    resources.ServiceConfig
    CustomField1 string
    CustomField2 int
}
```

### Advanced Dependency Management

Handle complex dependency chains:

```go
// Create dependency groups
group := mesh.NewServiceGroup("payment-group")
group.AddService(paymentService)
group.AddService(authService)

// Set group-level policies
group.SetCircuitBreaker(resources.CircuitBreakerConfig{
    FailureThreshold: 10,
    ResetTimeout:    30 * time.Second,
})
```

## Troubleshooting

### Common Issues

1. **High Failure Rates**
   - Check circuit breaker configuration
   - Verify rate limits are appropriate
   - Review dependency health

2. **Performance Issues**
   - Monitor resource usage
   - Check concurrent request limits
   - Verify timeout settings

3. **Cascading Failures**
   - Review dependency chain
   - Check circuit breaker settings
   - Monitor service health metrics

## Next Steps

- Review the [Examples](../examples/) directory for more scenarios
- Check out the [API Documentation](API.md)
- Learn about [Contributing](CONTRIBUTING.md)