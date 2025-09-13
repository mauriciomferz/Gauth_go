package resilient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
)

// ExampleService simulates a service that might fail
type ExampleService struct {
	failures int
	mu       sync.Mutex
}

func (s *ExampleService) Call() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failures++
	if s.failures%3 == 0 {
		return fmt.Errorf("service temporarily unavailable")
	}
	return nil
}

// RunExample demonstrates different resilience patterns in action
func RunExample() {
	// Set up rate limiters
	tokenConfig := &ratelimit.Config{
		RequestsPerSecond: 5,
		WindowSize:        2,
		BurstSize:         10,
	}
	slidingConfig := &ratelimit.Config{
		RequestsPerSecond: 5,
		WindowSize:        2,
		BurstSize:         8,
	}

	tokenBucket := ratelimit.WrapTokenBucket(tokenConfig)
	slidingWindow := ratelimit.WrapSlidingWindow(slidingConfig)

	// Set up circuit breaker
	cbOpts := circuit.Options{
		Name:             "example-service",
		FailureThreshold: 3,
		ResetTimeout:     5 * time.Second,
		HalfOpenLimit:    2,
		OnStateChange: func(name string, from, to circuit.State) {
			fmt.Printf("Circuit state changed from %s to %s\n", from, to)
		},
	}
	breaker := circuit.NewCircuitBreaker(cbOpts)

	// Set up retry strategy
	retryStrategy := resilience.RetryStrategy{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}
	retry := resilience.NewRetry(retryStrategy)

	// Set up bulkhead
	bulkhead := resilience.NewBulkhead(2)

	// Create example service
	service := &ExampleService{}

	// Test scenarios
	var wg sync.WaitGroup
	ctx := context.Background()

	// Scenario 1: Token Bucket Rate Limiting
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\nScenario 1: Token Bucket Rate Limiting")
		for i := 1; i <= 12; i++ {
			err := tokenBucket.Allow(ctx, "client1")
			quota := tokenBucket.GetQuota("client1")
			fmt.Printf("Request %d: %v (Remaining: %d/%d, Resets: %v)\n",
				i,
				errString(err),
				quota.Remaining,
				quota.Total,
				quota.ResetAt.Sub(time.Now()).Round(time.Second))
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Scenario 2: Sliding Window Rate Limiting
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\nScenario 2: Sliding Window Rate Limiting")
		for i := 1; i <= 12; i++ {
			err := slidingWindow.Allow(ctx, "client2")
			quota := slidingWindow.GetQuota("client2")
			fmt.Printf("Request %d: %v (Window Requests: %d, Duration: %v)\n",
				i,
				errString(err),
				quota.Window.Requests,
				quota.Window.Duration)
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Scenario 3: Circuit Breaker with Retry
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\nScenario 3: Circuit Breaker with Retry")

		for i := 1; i <= 8; i++ {
			err := retry.Execute(ctx, func() error {
				return breaker.Execute(service.Call)
			})

			fmt.Printf("Request %d: %v (Circuit State: %s)\n",
				i,
				errString(err),
				breaker.State())
			time.Sleep(time.Second)
		}
	}()

	// Scenario 4: Combined Patterns
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\nScenario 4: All Patterns Combined")

		for i := 1; i <= 5; i++ {
			start := time.Now()
			err := bulkhead.Execute(ctx, func() error {
				return retry.Execute(ctx, func() error {
					if err := tokenBucket.Allow(ctx, "client4"); err != nil {
						return err
					}
					return breaker.Execute(service.Call)
				})
			})

			duration := time.Since(start)
			fmt.Printf("Request %d: %v (Took: %v, Circuit: %s)\n",
				i,
				errString(err),
				duration.Round(time.Millisecond),
				breaker.State())
			time.Sleep(time.Second)
		}
	}()

	// Wait for all scenarios to complete
	wg.Wait()
}

func errString(err error) string {
	if err == nil {
		return "Success"
	}
	return err.Error()
}
