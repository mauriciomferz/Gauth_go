package comprehensive
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/resilience"
)

// SimulatedService represents an external service that might fail
type SimulatedService struct {
	failureCount int
	successCount int
}

func (s *SimulatedService) Call() error {
	// Simulate intermittent failures
	s.failureCount++
	if s.failureCount%3 == 0 {
		return errors.New("service temporarily unavailable")
	}
	s.successCount++
	return nil
}

func (s *SimulatedService) SlowCall() error {
	time.Sleep(200 * time.Millisecond)
	return s.Call()
}

func main() {
	// Create simulated service
	service := &SimulatedService{}

	// Create circuit breaker
	cb := resilience.NewCircuitBreaker(resilience.CircuitConfig{
		Name:        "example-circuit",
		MaxFailures: 3,
		Timeout:     2 * time.Second,
		Interval:    5 * time.Second,
	})

	// Create retry handler
	retry := resilience.NewRetry(resilience.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		Multiplier:    2.0,
	})

	// Create timeout handler
	timeout := resilience.NewTimeout(resilience.TimeoutConfig{
		Timeout: 150 * time.Millisecond,
	})

	// Create bulkhead
	bulkhead := resilience.NewBulkhead(resilience.BulkheadConfig{
		MaxConcurrent: 2,
		MaxWaitTime:   100 * time.Millisecond,
	})

	// Combine patterns
	combined := resilience.Combine(cb, retry, timeout, bulkhead)

	// HTTP handler using resilience patterns
	http.HandleFunc("/resilient", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		err := combined.Execute(ctx, func(ctx context.Context) error {
			return service.SlowCall()
		})

		if err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				http.Error(w, "Request timed out", http.StatusGatewayTimeout)
			case errors.Is(err, resilience.ErrCircuitOpen):
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			case errors.Is(err, resilience.ErrBulkheadFull):
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		fmt.Fprintf(w, "Request successful! Service stats: %d successes, %d failures\n",
			service.successCount, service.failureCount)
	})

	// Start server
	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}