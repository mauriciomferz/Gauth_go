# Resilience Patterns Guide

This guide explains the resilience patterns implemented in the library and when to use them.

## Overview

The library implements four main resilience patterns:
1. Circuit Breaker
2. Rate Limiting
3. Bulkhead
4. Retry

Each pattern addresses different types of failures and load scenarios.

## Circuit Breaker Pattern

### Purpose
Prevents system overload by stopping calls to failing services.

### When to Use
- Service has high failure rate
- Downstream service is unresponsive
- Need to prevent cascade failures

### Configuration Example
```go
service.Config = resources.ServiceConfig{
    CircuitBreaker: resources.CircuitBreakerConfig{
        FailureThreshold: 5,      // Open after 5 failures
        ResetTimeout:    10 * time.Second,  // Time before retry
        HalfOpenLimit:   2,       // Requests allowed in half-open
    },
}
```

### Best Practices
1. Set appropriate failure thresholds
2. Use reasonable reset timeouts
3. Monitor circuit state changes
4. Handle circuit open scenarios gracefully

## Rate Limiting Pattern

### Purpose
Prevents service overload by controlling request rates.

### When to Use
- Protect limited resources
- Prevent DoS scenarios
- Implement fair usage policies
- Control costs

### Configuration Example
```go
service.Config = resources.ServiceConfig{
    RequestsPerSecond: 100,  // Base rate
    BurstSize:        20,    // Burst allowance
    WindowSize:       1,      // Time window in seconds
}
```

### Best Practices
1. Set rates based on capacity
2. Allow reasonable bursts
3. Monitor rate limit events
4. Implement graceful degradation

## Bulkhead Pattern

### Purpose
Isolates failures by partitioning service resources.

### When to Use
- Protect shared resources
- Isolate critical functions
- Prevent resource exhaustion
- Maintain SLAs for critical paths

### Configuration Example
```go
service.Config = resources.ServiceConfig{
    MaxConcurrent:  10,  // Max concurrent requests
    QueueSize:      50,  // Request queue size
    Timeout:        30 * time.Second,  // Request timeout
}
```

### Best Practices
1. Size partitions appropriately
2. Set reasonable timeouts
3. Monitor resource usage
4. Handle rejection scenarios

## Retry Pattern

### Purpose
Handles transient failures through automatic retries.

### When to Use
- Network timeouts
- Temporary service unavailability
- Race conditions
- Eventual consistency issues

### Configuration Example
```go
service.Config = resources.ServiceConfig{
    RetryAttempts:     3,                    // Max attempts
    RetryBackoff:      50 * time.Millisecond, // Initial delay
    RetryMultiplier:   2.0,                  // Backoff multiplier
    MaxRetryInterval:  5 * time.Second,      // Max delay
}
```

### Best Practices
1. Use exponential backoff
2. Set maximum retry limits
3. Consider downstream impact
4. Handle permanent failures

## Pattern Combinations

### Circuit Breaker + Retry
Good for handling transient failures while preventing cascading failures:

```go
service.Config = resources.ServiceConfig{
    CircuitBreaker: resources.CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:    10 * time.Second,
    },
    RetryAttempts:    3,
    RetryBackoff:     50 * time.Millisecond,
}
```

### Rate Limit + Bulkhead
Good for resource protection and load management:

```go
service.Config = resources.ServiceConfig{
    RequestsPerSecond: 100,
    BurstSize:        20,
    MaxConcurrent:    10,
    QueueSize:        50,
}
```

## Monitoring and Metrics

### Health Metrics
Monitor pattern effectiveness:

```go
snapshot := service.Health.GetSnapshot()
fmt.Printf("Success Rate: %.2f%%\n", snapshot.SuccessRate)
fmt.Printf("Average Latency: %v\n", snapshot.AverageLatency)
```

### Event Handling
Track pattern behavior:

```go
service.Events().Subscribe(func(e events.Event) {
    switch e.Type {
    case events.CircuitOpened:
        // Handle circuit breaker opening
    case events.RateLimitExceeded:
        // Handle rate limit events
    case events.BulkheadRejected:
        // Handle bulkhead rejections
    }
})
```

## Troubleshooting

### High Failure Rates
1. Check circuit breaker configuration
2. Review retry policies
3. Monitor downstream services
4. Verify timeout settings

### Performance Issues
1. Adjust rate limits
2. Review bulkhead settings
3. Check resource usage
4. Monitor latency trends

### Resource Exhaustion
1. Verify bulkhead configuration
2. Check rate limit settings
3. Monitor queue sizes
4. Review concurrent requests

## Best Practices Summary

1. **Configuration**
   - Set appropriate thresholds
   - Use reasonable timeouts
   - Configure gradual degradation
   - Monitor pattern effectiveness

2. **Error Handling**
   - Handle all failure scenarios
   - Provide clear error messages
   - Implement fallback strategies
   - Log relevant context

3. **Monitoring**
   - Track pattern metrics
   - Monitor resource usage
   - Alert on degradation
   - Analyze trends

4. **Testing**
   - Test failure scenarios
   - Verify pattern behavior
   - Check combination effects
   - Monitor resource usage