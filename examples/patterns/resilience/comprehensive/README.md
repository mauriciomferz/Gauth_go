# Comprehensive Resilience Patterns Example

This example demonstrates the implementation of multiple resilience patterns working together to create a robust service.

## Features

1. **Circuit Breaker Pattern**
   - Failure threshold monitoring
   - Automatic circuit opening
   - Timed reset behavior
   - State monitoring

2. **Retry Pattern**
   - Exponential backoff
   - Maximum attempts limit
   - Configurable delays
   - Retry monitoring

3. **Timeout Pattern**
   - Context-based timeouts
   - Configurable durations
   - Clean cancellation
   - Timeout tracking

4. **Monitoring**
   - Real-time metrics
   - Pattern state tracking
   - Success/failure rates
   - Response times

## Running the Example

```bash
go run main.go
```

The server will start on http://localhost:8080 with two endpoints:
- `/resilient`: Test the resilient operation
- `/metrics`: View current metrics

## Testing Scenarios

1. **Normal Operation**
```bash
curl http://localhost:8080/resilient
```

2. **Trigger Circuit Breaker**
```bash
for i in {1..10}; do curl http://localhost:8080/resilient; done
```

3. **View Metrics**
```bash
curl http://localhost:8080/metrics
```

## Code Structure

### 1. Base Service
```go
type SimulatedService struct {
    failureRate float64
    delay       time.Duration
}
```

### 2. Resilience Wrapper
```go
type ResilientService struct {
    service  *SimulatedService
    breaker *resilience.CircuitBreaker
    retry   *resilience.Retry
    timeout *resilience.Timeout
    monitor *resilience.Monitor
}
```

### 3. Pattern Configuration
```go
breaker := resilience.NewCircuitBreaker(resilience.BreakerConfig{
    Name:            "example-service",
    FailureThreshold: 5,
    ResetTimeout:     10 * time.Second,
})

retry := resilience.NewRetry(resilience.RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  100 * time.Millisecond,
    BackoffFactor: 2.0,
})

timeout := resilience.NewTimeout(resilience.TimeoutConfig{
    Duration: 2 * time.Second,
})
```

## Pattern Composition

The example shows how to compose multiple resilience patterns:

1. **Outer Layer**: Timeout
   - Ensures overall operation completion time

2. **Middle Layer**: Retry
   - Handles transient failures
   - Implements backoff strategy

3. **Inner Layer**: Circuit Breaker
   - Prevents cascade failures
   - Allows system recovery

## Error Handling

The example demonstrates proper error handling with type-safe errors:

```go
switch {
case resilience.IsCircuitOpen(err):
    // Handle circuit breaker open
case resilience.IsTimeout(err):
    // Handle timeout
case resilience.IsMaxRetriesExceeded(err):
    // Handle retry exhaustion
default:
    // Handle other errors
}
```

## Metrics

The example provides real-time metrics:
- Circuit breaker state
- Request counts
- Success/failure rates
- Retry attempts
- Timeout occurrences

## Best Practices

1. **Pattern Configuration**
   - Use reasonable timeouts
   - Configure appropriate thresholds
   - Implement exponential backoff
   - Monitor pattern behavior

2. **Error Handling**
   - Use typed errors
   - Proper error categorization
   - Clear error messages
   - Context preservation

3. **Monitoring**
   - Track pattern states
   - Monitor success rates
   - Measure response times
   - Alert on pattern triggers

4. **Resource Management**
   - Clean timeout cancellation
   - Proper context handling
   - Resource cleanup
   - Graceful shutdown