# API Documentation

## Core Types

### ServiceType

```go
type ServiceType int

const (
    AuthService ServiceType = iota
    UserService
    OrderService
    // ...
)
```

Represents different types of services in the mesh. Each service type has specific characteristics and dependencies.

### Microservice

```go
type Microservice struct {
    Type         ServiceType
    Name         string
    Dependencies []ServiceType
    Config       resources.ServiceConfig
    Status       resources.HealthStatus
    Usage        resources.ResourceUsage
}
```

The main service type that implements resilience patterns. Use `NewMicroservice` to create instances.

### ServiceConfig

```go
type ServiceConfig struct {
    MaxConcurrent     int
    RequestsPerSecond int
    TimeoutSeconds    int
    RetryAttempts     int
    RetryBackoff      time.Duration
}
```

Configuration options for service behavior and resilience patterns.

## Events

### EventType

```go
type EventType int

const (
    ServiceStarted EventType = iota
    ServiceStopped
    RequestReceived
    RequestCompleted
    RequestFailed
    CircuitOpened
    CircuitClosed
    CircuitHalfOpen
    RateLimitExceeded
    BulkheadRejected
)
```

Represents different types of events that can occur in the service mesh.

### Event

```go
type Event struct {
    Type      EventType
    ServiceID string
    Timestamp time.Time
    Duration  time.Duration
    Error     error
    Details   map[string]interface{}
}
```

Represents a service event with its context and metadata.

## Health Monitoring

### HealthMetrics

```go
type HealthMetrics struct {
    // private fields
}

func (h *HealthMetrics) SuccessRate() float64
func (h *HealthMetrics) LastFailureTime() time.Time
func (h *HealthMetrics) GetSnapshot() HealthSnapshot
```

Tracks service health and performance metrics.

### HealthSnapshot

```go
type HealthSnapshot struct {
    TotalRequests   int
    SuccessRate     float64
    LastFailureTime time.Time
    AverageLatency  time.Duration
}
```

Point-in-time snapshot of service health metrics.

## Resource Management

### ResourceUsage

```go
type ResourceUsage struct {
    CPUPercent    float64
    MemoryPercent float64
    Connections   int
    ThreadCount   int
    UpdatedAt     time.Time
}
```

Tracks resource utilization for a service.

### StatusType

```go
type StatusType int

const (
    StatusHealthy StatusType = iota
    StatusDegraded
    StatusUnhealthy
    StatusUnknown
)
```

Represents different health status values for services.

## Main APIs

### NewMicroservice

```go
func NewMicroservice(sType ServiceType, name string, deps []ServiceType) *Microservice
```

Creates a new microservice with the specified type, name, and dependencies.

### ProcessRequest

```go
func (s *Microservice) ProcessRequest(ctx context.Context, mesh *ServiceMesh) error
```

Processes a request through the service with all configured resilience patterns.

### SetServiceLoad

```go
func (s *Microservice) SetLoadFactor(factor float64)
```

Updates the service's load factor to simulate different load conditions.

### GetSnapshot

```go
func (h *HealthMetrics) GetSnapshot() HealthSnapshot
```

Returns a point-in-time snapshot of service health metrics.

## Best Practices

1. **Service Creation**
   ```go
   service := mesh.NewMicroservice(
       mesh.PaymentService,
       "Payment",
       []mesh.ServiceType{mesh.AuthService},
   )
   ```

2. **Configuration**
   ```go
   service.Config = resources.ServiceConfig{
       MaxConcurrent:     10,
       RequestsPerSecond: 100,
       TimeoutSeconds:    30,
       RetryAttempts:     3,
   }
   ```

3. **Event Handling**
   ```go
   service.Events().Subscribe(func(e events.Event) {
       // Handle events
   })
   ```

4. **Health Monitoring**
   ```go
   snapshot := service.Health.GetSnapshot()
   if snapshot.SuccessRate < 90.0 {
       // Handle degraded service
   }
   ```

## Error Handling

Services return structured errors that can be type-asserted:

```go
if err := service.ProcessRequest(ctx, mesh); err != nil {
    switch e := err.(type) {
    case *ServiceError:
        log.Printf("Service error: %v", e)
    case *CircuitBreakerError:
        log.Printf("Circuit breaker open: %v", e)
    case *RateLimitError:
        log.Printf("Rate limit exceeded: %v", e)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## Thread Safety

All public methods are thread-safe. Internal synchronization is handled automatically.