# Resilience Package Documentation

The resilience package provides a robust set of patterns for building reliable and fault-tolerant services. This package implements several well-known resilience patterns that can be used individually or combined for comprehensive failure handling.

## Key Features

- Circuit Breaker: Prevents cascading failures by failing fast when a service is unhealthy
- Retry with Backoff: Implements intelligent retry logic with configurable backoff strategies
- Timeout: Ensures operations complete within expected time bounds
- Bulkhead: Limits concurrent operations to prevent resource exhaustion
- Pattern Composition: Allows combining multiple patterns for sophisticated failure handling

## Core Patterns

### Circuit Breaker

The circuit breaker pattern prevents system overload by temporarily stopping operations when a failure threshold is reached.

```go
cb := resilience.NewCircuitBreaker(resilience.CircuitConfig{
    Name:        "auth-service",
    MaxFailures: 3,
    Timeout:     2 * time.Second,
    Interval:    5 * time.Second,
})

err := cb.Execute(ctx, func(ctx context.Context) error {
    return callService()
})
```

### Retry with Backoff

Implements exponential backoff retry logic for transient failures.

```go
retry := resilience.NewRetry(resilience.RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  100 * time.Millisecond,
    MaxDelay:      1 * time.Second,
    Multiplier:    2.0,
})

err := retry.Execute(ctx, func(ctx context.Context) error {
    return callService()
})
```

### Timeout

Ensures operations complete within a specified time limit.

```go
timeout := resilience.NewTimeout(resilience.TimeoutConfig{
    Timeout: 150 * time.Millisecond,
})

err := timeout.Execute(ctx, func(ctx context.Context) error {
    return callService()
})
```

### Bulkhead

Controls concurrent operations to prevent resource exhaustion.

```go
bulkhead := resilience.NewBulkhead(resilience.BulkheadConfig{
    MaxConcurrent: 10,
    MaxWaitTime:   100 * time.Millisecond,
})

err := bulkhead.Execute(ctx, func(ctx context.Context) error {
    return callService()
})
```

## Pattern Composition

Multiple patterns can be combined for comprehensive resilience:

```go
combined := resilience.Combine(
    resilience.NewCircuitBreaker(circuitConfig),
    resilience.NewRetry(retryConfig),
    resilience.NewTimeout(timeoutConfig),
    resilience.NewBulkhead(bulkheadConfig),
)

err := combined.Execute(ctx, func(ctx context.Context) error {
    return callService()
})
```

## Best Practices

1. **Circuit Breaker Configuration**
   - Set appropriate failure thresholds based on service characteristics
   - Configure reasonable timeout and reset intervals
   - Monitor circuit state changes

2. **Retry Strategy**
   - Use exponential backoff to prevent thundering herd
   - Set appropriate max attempts to prevent infinite retries
   - Consider operation idempotency

3. **Timeout Management**
   - Set timeouts based on SLAs and dependencies
   - Consider cascading timeouts in distributed systems
   - Always provide context with timeout

4. **Bulkhead Implementation**
   - Size concurrent operation limits based on resources
   - Set appropriate wait times for queued operations
   - Monitor rejection rates

## Error Handling

The package provides specific error types for different failure scenarios:

- `ErrCircuitOpen`: Circuit breaker is open
- `ErrBulkheadFull`: Bulkhead capacity exceeded
- `context.DeadlineExceeded`: Operation timed out
- `ErrMaxRetriesExceeded`: Maximum retry attempts reached

## Monitoring and Metrics

The resilience patterns expose metrics for monitoring:

- Circuit breaker state changes
- Retry attempts and success rates
- Timeout occurrences
- Bulkhead rejection rates

## Examples

See the `/examples/resilience` directory for comprehensive examples of:

- Basic pattern usage
- Pattern composition
- Integration with HTTP services
- Distributed system resilience
- Advanced configuration scenarios

## Contributing

When contributing to this package:

1. Add tests for new functionality
2. Follow existing pattern interfaces
3. Document configuration options
4. Include examples for new features
5. Consider backward compatibility

## See Also

- [Package Reference](https://pkg.go.dev/github.com/Gimel-Foundation/gauth/pkg/resilience)
- [Example Applications](../examples/resilience)
- [Design Patterns Guide](../docs/PATTERNS_GUIDE.md)
- [Integration Tests](../test/integration)