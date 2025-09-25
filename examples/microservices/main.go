package main

import (
	"context"
	"fmt"
	"crypto/rand"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
)

// MicroserviceExample demonstrates resilience patterns in a microservices architecture
type MicroserviceExample struct {
	name     string
	latency  time.Duration
	failRate float64
	mu       sync.Mutex
}

func NewMicroservice(name string, latency time.Duration, failRate float64) *MicroserviceExample {
	return &MicroserviceExample{
		name:     name,
		latency:  latency,
		failRate: failRate,
	}
}

func (s *MicroserviceExample) Call(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Simulate processing time
	time.Sleep(s.latency)

	// Simulate random failures using crypto/rand
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Errorf("failed to generate secure random: %w", err)
	}
	randomFloat := float64(randomBytes[0]) / 255.0 // Convert to 0-1 range
	
	if randomFloat < s.failRate {
		return fmt.Errorf("%s: service temporarily unavailable", s.name)
	}
	return nil
}

// ServiceChain represents a chain of dependent microservices
type ServiceChain struct {
	services []*MicroserviceExample
	breakers map[string]*circuit.CircuitBreaker
	limiter  ratelimit.Algorithm
	retry    *resilience.Retry
	bulkhead *resilience.Bulkhead
}

func NewServiceChain(services []*MicroserviceExample) *ServiceChain {
	// Initialize circuit breakers for each service
	breakers := make(map[string]*circuit.CircuitBreaker)
	for _, svc := range services {
		breakers[svc.name] = circuit.NewCircuitBreaker(circuit.Options{
			Name:             svc.name,
			FailureThreshold: 3,
			ResetTimeout:     5 * time.Second,
			HalfOpenLimit:    2,
			OnStateChange: func(name string, from, to circuit.State) {
				fmt.Printf("[%s] Circuit state changed: %s -> %s\n", name, from, to)
			},
		})
	}

	// Initialize rate limiter for the entire chain
	limiter := ratelimit.WrapTokenBucket(&ratelimit.Config{
		RequestsPerSecond: 10,
		WindowSize:        1,
		BurstSize:         5,
	})

	// Initialize retry strategy
	retry := resilience.NewRetry(resilience.RetryStrategy{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	})

	// Initialize bulkhead for concurrency control
	bulkhead := resilience.NewBulkhead(5)

	return &ServiceChain{
		services: services,
		breakers: breakers,
		limiter:  limiter,
		retry:    retry,
		bulkhead: bulkhead,
	}
}

func (sc *ServiceChain) ExecuteChain(ctx context.Context) error {
	// First apply rate limiting
	if err := sc.limiter.Allow(ctx, "chain"); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Execute through bulkhead
	return sc.bulkhead.Execute(ctx, func() error {
		// Call each service in the chain with retries and circuit breaking
		for _, svc := range sc.services {
			breaker := sc.breakers[svc.name]
			err := sc.retry.Execute(ctx, func() error {
				return breaker.Execute(func() error {
					return svc.Call(ctx)
				})
			})
			if err != nil {
				return fmt.Errorf("service %s failed: %w", svc.name, err)
			}
		}
		return nil
	})
}

func main() {
	// Create a chain of microservices with different characteristics
	services := []*MicroserviceExample{
		NewMicroservice("auth-service", 50*time.Millisecond, 0.1),   // 10% failure rate
		NewMicroservice("user-service", 100*time.Millisecond, 0.2),  // 20% failure rate
		NewMicroservice("order-service", 150*time.Millisecond, 0.3), // 30% failure rate
	}

	chain := NewServiceChain(services)
	ctx := context.Background()

	// Simulate multiple concurrent requests
	var wg sync.WaitGroup
	start := time.Now()

	fmt.Println("\nStarting Microservices Resilience Demo...")
	fmt.Println("----------------------------------------")
	fmt.Println("Services in chain:")
	for _, svc := range services {
		fmt.Printf("- %s (latency: %v, failure rate: %.1f%%)\n",
			svc.name, svc.latency, svc.failRate*100)
	}
	fmt.Println("----------------------------------------")

	for i := 1; i <= 20; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			fmt.Printf("\n[Request %d] Starting request...\n", requestID)
			err := chain.ExecuteChain(ctx)
			duration := time.Since(start)
			if err != nil {
				fmt.Printf("[Request %d] Failed after %v: %v\n", requestID, duration.Round(time.Millisecond), err)
			} else {
				fmt.Printf("[Request %d] Completed successfully after %v\n", requestID, duration.Round(time.Millisecond))
			}
		}(i)

		// Add some delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()
	fmt.Println("\nMicroservices demo completed!")
}
