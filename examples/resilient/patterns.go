package resilient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/circuit"
	"github.com/mauriciomferz/AgentAuth/internal/ratelimit"
	"github.com/mauriciomferz/AgentAuth/internal/resilience"
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
	}
	breaker := circuit.NewBreaker(cbOpts)

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
			if err := tokenBucket.Allow(ctx, "token_bucket_example"); err != nil {
				fmt.Printf("Request %d: Rate limited (%v)\n", i, err)
			} else {
				fmt.Printf("Request %d: Success\n", i)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Scenario 2: Sliding Window Rate Limiting
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\nScenario 2: Sliding Window Rate Limiting")
		for i := 1; i <= 12; i++ {
			if err := slidingWindow.Allow(ctx, "sliding_window_example"); err != nil {
				fmt.Printf("Request %d: Rate limited (%v)\n", i, err)
			} else {
				fmt.Printf("Request %d: Success\n", i)
			}
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
				return breaker.Execute(ctx, service.Call)
			})

			fmt.Printf("Request %d: %v\n",
				i,
				errString(err))
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
					if err := tokenBucket.Allow(ctx, "combined_example"); err != nil {
						return fmt.Errorf("rate limit exceeded: %w", err)
					}
					return breaker.Execute(ctx, service.Call)
				})
			})

			duration := time.Since(start)
			fmt.Printf("Request %d: %v (Took: %v)\n",
				i,
				errString(err),
				duration.Round(time.Millisecond))
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
