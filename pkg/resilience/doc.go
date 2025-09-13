/*
Package resilience implements patterns for building reliable and fault-tolerant authentication services.

This package provides implementations of common resilience patterns that can be
used individually or combined for comprehensive failure handling in distributed authentication systems.

Key Patterns:

 1. Circuit Breaker:
    Prevents cascading failures by failing fast when an authentication service is unhealthy.

    cb := resilience.NewCircuitBreaker(resilience.CircuitConfig{
    MaxFailures: 3,
    Timeout:     2 * time.Second,
    })

    err := cb.Execute(ctx, func(ctx context.Context) error {
    return authService.Authenticate(credentials)
    })

 2. Retry with Backoff:
    Implements intelligent retry logic with configurable backoff strategies for authentication requests.

    retry := resilience.NewRetry(resilience.RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  100 * time.Millisecond,
    Multiplier:    2.0,
    JitterFactor:  0.1,
    })

 3. Timeout:
    Ensures operations complete within expected time bounds.

    timeout := resilience.NewTimeout(resilience.TimeoutConfig{
    Timeout: 150 * time.Millisecond,
    })

 4. Bulkhead:
    Controls concurrent operations to prevent resource exhaustion.

    bulkhead := resilience.NewBulkhead(resilience.BulkheadConfig{
    MaxConcurrent: 10,
    MaxWaitTime:   100 * time.Millisecond,
    })

Pattern Composition:

Patterns can be combined for comprehensive resilience:

	combined := resilience.Combine(
		resilience.NewCircuitBreaker(circuitConfig),
		resilience.NewRetry(retryConfig),
		resilience.NewTimeout(timeoutConfig),
		resilience.NewBulkhead(bulkheadConfig),
	)

	err := combined.Execute(ctx, func(ctx context.Context) error {
		return callService()
	})

Monitoring:

All patterns expose metrics for monitoring:
- Circuit breaker state changes
- Retry attempts and success rates
- Timeout occurrences
- Bulkhead rejection rates

Error Handling:

The package provides specific error types:
- ErrCircuitOpen: Circuit breaker is open
- ErrBulkheadFull: Bulkhead capacity exceeded
- ErrMaxRetriesExceeded: Maximum retry attempts reached

Thread Safety:

All types in this package are designed to be thread-safe
and can be used concurrently.

Usage Example:

	// Create a resilient HTTP client
	client := &http.Client{
		Transport: resilience.NewTransport(
			resilience.NewCircuitBreaker(resilience.CircuitConfig{
				MaxFailures: 3,
				Timeout:     2 * time.Second,
			}),
			resilience.NewRetry(resilience.RetryConfig{
				MaxAttempts: 3,
				InitialDelay: 100 * time.Millisecond,
			}),
		),
	}

	// Make resilient HTTP requests
	resp, err := client.Get("http://api.example.com/data")

Best Practices:

1. Circuit Breaker:
  - Set appropriate failure thresholds
  - Configure reasonable timeout and reset intervals
  - Monitor state changes

2. Retry Strategy:
  - Use exponential backoff
  - Set appropriate max attempts
  - Consider operation idempotency

3. Timeout Management:
  - Set timeouts based on SLAs
  - Consider cascading timeouts
  - Always provide context with timeout

4. Bulkhead Implementation:
  - Size concurrent limits appropriately
  - Set appropriate wait times
  - Monitor rejection rates

See Also:
- Package examples/resilience for comprehensive examples
- Package monitoring for metrics integration
- Package events for failure event handling
*/
package resilience
