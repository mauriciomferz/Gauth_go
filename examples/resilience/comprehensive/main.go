package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/resilience"
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

	// Create circuit breaker (using available config)
	cb := resilience.NewCircuitBreaker(resilience.WithCircuitBreaker(3, 2*time.Second, nil))

	// Create retry handler (using available config)
	retry := resilience.NewRetry(resilience.WithRetry(3, 100*time.Millisecond, 1*time.Second))

	// Create timeout handler (stub, as TimeoutConfig is not available)
	timeout := resilience.NewTimeout(nil)

	// Create bulkhead (using available config)
	bulkhead := resilience.NewBulkhead(2)

	// Combine only Patterns (not Bulkhead)
	combined := resilience.Combine(cb, retry, timeout)

	// HTTP handler using resilience patterns
	http.HandleFunc("/resilient", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := bulkhead.Execute(ctx, func() error {
			return combined.Execute(ctx, func() error {
				return service.SlowCall()
			})
		})
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				http.Error(w, "Request timed out", http.StatusGatewayTimeout)
				return
			}
			// Other error handling can be added here if needed
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Request successful! Service stats: %d successes, %d failures\n",
			service.successCount, service.failureCount)
	})

	// Start server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Starting server on :8080...")
	log.Fatal(server.ListenAndServe())
}
