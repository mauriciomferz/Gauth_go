# Development Guide

This guide helps you understand the codebase and how to extend it for your needs. The project follows a clean architecture pattern with clear separation of concerns and strong type safety.

## Project Structure

```
cascade/
├── cmd/                    # Example applications
│   └── main.go            # Demo entry point
├── internal/              # Private implementation
│   └── resilience/        # Resilience pattern implementations
├── pkg/                   # Public API packages
│   └── mesh/             # Service mesh implementation
└── docs/                 # Documentation
```

## Type System

We prioritize type safety throughout the codebase:

1. **Events**: Use strongly typed event definitions instead of strings
```go
type ServiceEvent struct {
    Type    EventType
    Service ServiceType
    Data    any // Generic but with concrete types per event
}
```

2. **Configuration**: Type-safe configuration structures
```go
type ServiceConfig struct {
    Name           string
    MaxConcurrency int
    RetryPolicy    RetryConfig
}
```

3. **Metadata**: Structured metadata instead of map[string]interface{}
```go
type ServiceMetadata struct {
    Version     string
    Region      string
    Tags        []string
    Attributes  map[string]string // Only string values
}
```

## Core Concepts

### Service Mesh

The service mesh coordinates interactions between microservices. Key components:

1. **ServiceType**: Enumerated type for different services
```go
type ServiceType int

const (
    AuthService ServiceType = iota
    UserService
    // ...
)
```

2. **DependencyGraph**: Manages service dependencies
```go
type DependencyGraph struct {
    dependencies map[ServiceType][]ServiceType
}
```

3. **HealthMetrics**: Tracks service health
```go
type HealthMetrics struct {
    Successes      int
    Failures       int
    ResponseTimes  []time.Duration
}
```

### Resilience Patterns

#### Circuit Breaker
Prevents cascading failures by stopping calls to failing services:

```go
breaker := resilience.NewCircuitBreaker("payment", 5, 10*time.Second)
err := breaker.Execute(func() error {
    return callService()
})
```

#### Rate Limiter
Controls request rates to prevent overload:

```go
limiter := resilience.NewRateLimiter(100, 20)
if err := limiter.Allow(); err != nil {
    return err
}
```

#### Bulkhead
Isolates failures by partitioning service resources:

```go
bulkhead := resilience.NewBulkhead(10)
err := bulkhead.Execute(ctx, func() error {
    return processRequest()
})
```

## Extending the Code

### Adding a New Service

1. Add a new service type:
```go
const (
    ExistingService ServiceType = iota
    NewService      // Add your service here
)
```

2. Update the dependency graph:
```go
mesh.graph.dependencies[NewService] = []ServiceType{
    AuthService,
    UserService,
}
```

### Creating Custom Resilience Patterns

1. Define your pattern interface:
```go
type CustomPattern interface {
    Execute(context.Context, func() error) error
}
```

2. Implement the pattern:
```go
type MyPattern struct {
    // Your implementation
}

func (p *MyPattern) Execute(ctx context.Context, fn func() error) error {
    // Your logic here
}
```

### Monitoring and Metrics

The system provides several monitoring points:

1. Service Health:
```go
health, err := mesh.GetServiceHealth(ServiceType)
fmt.Printf("Success Rate: %.2f%%\n", health.SuccessRate())
```

2. Load Factors:
```go
load := service.GetLoadFactor()
fmt.Printf("Current Load: %.2f\n", load)
```

## Testing

### Unit Tests

Run unit tests for specific packages:
```bash
go test ./pkg/mesh/...
go test ./internal/resilience/...
```

### Integration Tests

Run the full suite of integration tests:
```bash
go test ./test/integration/...
```

### Load Testing

Use the provided load testing utilities:
```bash
go run cmd/loadtest/main.go -duration=5m -rps=100
```

## Common Patterns

### Error Handling

We follow these principles for error handling:

1. Use structured errors for better context:
```go
if err != nil {
    return fmt.Errorf("service %s failed: %w", name, err)
}
```

2. Wrap errors with context:
```go
type ServiceError struct {
    Service ServiceType
    Err     error
}
```

### Configuration

Services can be configured through a fluent interface:

```go
service := NewMicroservice(PaymentService).
    WithCircuitBreaker(5, 10*time.Second).
    WithRateLimit(100, 20).
    WithRetry(3, 50*time.Millisecond)
```

## Best Practices

1. Keep services focused and small
2. Use strong typing everywhere
3. Document public APIs thoroughly
4. Write tests for new functionality
5. Follow Go idioms and conventions

## Troubleshooting

Common issues and solutions:

1. Circuit breaker triggering too often
   - Check failure thresholds
   - Adjust reset timeout

2. High latency under load
   - Review bulkhead configuration
   - Check rate limits

3. Cascading failures
   - Verify dependency graph
   - Check circuit breaker settings