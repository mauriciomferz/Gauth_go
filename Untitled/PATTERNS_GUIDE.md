# Resilience Patterns Usage Guide

This guide demonstrates the usage of resilience patterns in different scenarios.

## Table of Contents
1. [Basic Rate Limiting](#basic-rate-limiting)
2. [Microservices Architecture](#microservices-architecture)
3. [Distributed Systems](#distributed-systems)
4. [High-Availability Applications](#high-availability-applications)

## Basic Rate Limiting

The simplest form of protection is rate limiting. Use this when you need to:
- Protect APIs from abuse
- Manage resource consumption
- Ensure fair usage across clients

```go
limiter := ratelimit.WrapTokenBucket(&ratelimit.Config{
    RequestsPerSecond: 10,
    WindowSize:       1,
    BurstSize:       5,
})

// Use in your handler
if err := limiter.Allow(ctx, clientID); err != nil {
    return fmt.Errorf("rate limit exceeded")
}
```

## Microservices Architecture

In a microservices architecture, you'll want to use multiple patterns together:
- Circuit breakers for each service
- Rate limiting for API endpoints
- Bulkhead for resource isolation
- Retry for transient failures

See `examples/microservices/chain.go` for a complete example of:
- Service discovery and health checks
- Cascading failure prevention
- Load balancing and failover
- Request tracing and monitoring

## Distributed Systems

For distributed systems, focus on:
- Consensus and leader election
- Distributed rate limiting
- Cross-service circuit breaking
- Global resource management

Key patterns:
1. **Distributed Rate Limiting**
   ```go
   // Use a distributed store (e.g., Redis)
   store := redis.NewStore(redisClient)
   limiter := ratelimit.NewDistributedLimiter(store, config)
   ```

2. **Cross-Service Circuit Breaking**
   ```go
   breaker := circuit.NewCircuitBreaker(circuit.Options{
       FailureThreshold: 5,
       ResetTimeout:     10 * time.Second,
       OnStateChange: func(name string, from, to circuit.State) {
           // Notify other services about state change
           notifyStateChange(name, from, to)
       },
   })
   ```

## High-Availability Applications

For high-availability applications:
1. **Use Multiple Layers of Protection**
   ```go
   // Combine patterns
   bulkhead.Execute(ctx, func() error {
       return retry.Execute(ctx, func() error {
           if err := limiter.Allow(ctx, "client"); err != nil {
               return err
           }
           return breaker.Execute(serviceCall)
       })
   })
   ```

2. **Monitor and Adapt**
   - Track success/failure rates
   - Adjust thresholds dynamically
   - Use sliding windows for more accurate rate limiting

## Best Practices

1. **Rate Limiting**
   - Use token bucket for API endpoints
   - Use sliding window for accurate traffic shaping
   - Consider client identity and request type

2. **Circuit Breaking**
   - Set appropriate thresholds based on traffic
   - Use half-open state to test recovery
   - Implement fallback mechanisms

3. **Retry Strategies**
   - Use exponential backoff
   - Set maximum retry attempts
   - Consider timeout budgets

4. **Bulkhead Pattern**
   - Isolate critical resources
   - Set appropriate concurrency limits
   - Monitor resource usage

## Example Configurations

1. **API Gateway**
   ```go
   config := &Config{
       RateLimit: &ratelimit.Config{
           RequestsPerSecond: 100,
           BurstSize:        20,
       },
       CircuitBreaker: &circuit.Options{
           FailureThreshold: 10,
           ResetTimeout:     5 * time.Second,
       },
       Retry: &resilience.RetryStrategy{
           MaxAttempts: 3,
           Multiplier:  2.0,
       },
   }
   ```

2. **Internal Service**
   ```go
   config := &Config{
       RateLimit: &ratelimit.Config{
           RequestsPerSecond: 50,
           BurstSize:        10,
       },
       CircuitBreaker: &circuit.Options{
           FailureThreshold: 5,
           ResetTimeout:     3 * time.Second,
       },
       Bulkhead: &resilience.BulkheadConfig{
           MaxConcurrent: 20,
       },
   }
   ```

## Monitoring and Observability

To effectively use these patterns, implement proper monitoring:

1. **Metrics to Track**
   - Request rates and latencies
   - Circuit breaker state changes
   - Retry attempts and success rates
   - Resource utilization

2. **Logging**
   ```go
   breaker := circuit.NewCircuitBreaker(circuit.Options{
       OnStateChange: func(name string, from, to circuit.State) {
           log.Printf("Circuit %s state change: %s -> %s", name, from, to)
           metrics.RecordStateChange(name, from, to)
       },
   })
   ```

3. **Alerts**
   - Set up alerts for:
     - High failure rates
     - Circuit breaker trips
     - Rate limit exceeded events
     - Resource exhaustion

## Advanced Topics

1. **Custom Rate Limiting Algorithms**
   - Implement the `ratelimit.Algorithm` interface
   - Consider factors like:
     - Request priority
     - Client SLAs
     - Resource costs

2. **Adaptive Circuit Breaking**
   - Adjust thresholds based on:
     - Traffic patterns
     - Error rates
     - System load

3. **Coordinated Rate Limiting**
   - Share limits across instances
   - Consider global and local limits
   - Implement fair queuing

## Testing

Always test your resilience patterns:

1. **Unit Tests**
   ```go
   func TestCircuitBreaker(t *testing.T) {
       breaker := circuit.NewCircuitBreaker(opts)
       // Test state transitions
       // Test failure counting
       // Test timeout behavior
   }
   ```

2. **Integration Tests**
   - Test pattern combinations
   - Verify timeout behavior
   - Check resource cleanup

3. **Chaos Testing**
   - Inject failures
   - Simulate network issues
   - Test recovery behavior