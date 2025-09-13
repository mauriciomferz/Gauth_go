package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Gimel-Foundation/gauth/pkg/resilience"
)

// SimulatedService represents a service that might fail
type SimulatedService struct {
	failureRate float64
	delay       time.Duration
	mu          sync.Mutex
	failures    int
}

func NewSimulatedService(failureRate float64, delay time.Duration) *SimulatedService {
	return &SimulatedService{
		failureRate: failureRate,
		delay:       delay,
	}
}

func (s *SimulatedService) Process(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate processing delay
	time.Sleep(s.delay)

	// Simulate random failures
	if rand.Float64() < s.failureRate {
		s.mu.Lock()
		s.failures++
		s.mu.Unlock()
		return fmt.Errorf("simulated service failure (total failures: %d)", s.failures)
	}
	return nil
}

// ResilientService wraps a service with resilience patterns
// ServiceMetrics holds Prometheus metrics for the service
type ServiceMetrics struct {
	requestTotal    prometheus.Counter
	requestDuration prometheus.Histogram
	circuitState    prometheus.Gauge
	retryAttempts   prometheus.Counter
	timeoutTotal    prometheus.Counter
	failureTotal    prometheus.Counter
}

// NewServiceMetrics creates and registers service metrics
func NewServiceMetrics() *ServiceMetrics {
	m := &ServiceMetrics{
		requestTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "resilient_service_requests_total",
			Help: "Total number of requests made to the resilient service",
		}),
		requestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "resilient_service_duration_seconds",
			Help:    "Duration of service calls in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		circuitState: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "resilient_service_circuit_state",
			Help: "Current state of the circuit breaker (0=closed, 1=open, 0.5=half-open)",
		}),
		retryAttempts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "resilient_service_retry_attempts_total",
			Help: "Total number of retry attempts",
		}),
		timeoutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "resilient_service_timeouts_total",
			Help: "Total number of timeouts",
		}),
		failureTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "resilient_service_failures_total",
			Help: "Total number of service failures",
		}),
	}

	// Register metrics
	prometheus.MustRegister(
		m.requestTotal,
		m.requestDuration,
		m.circuitState,
		m.retryAttempts,
		m.timeoutTotal,
		m.failureTotal,
	)

	return m
}

type ResilientService struct {
	service *SimulatedService
	breaker *resilience.CircuitBreaker
	retry   *resilience.Retry
	timeout *resilience.Timeout
	metrics *ServiceMetrics
}

func NewResilientService(service *SimulatedService) *ResilientService {
	metrics := NewServiceMetrics()

	breaker := resilience.NewCircuitBreaker(resilience.BreakerConfig{
		Name:             "example-service",
		FailureThreshold: 5,
		ResetTimeout:     10 * time.Second,
		OnStateChange: func(from, to string) {
			stateValue := 0.0
			switch to {
			case "CLOSED":
				stateValue = 0.0
			case "OPEN":
				stateValue = 1.0
			case "HALF_OPEN":
				stateValue = 0.5
			}
			metrics.circuitState.Set(stateValue)
		},
	})

	retry := resilience.NewRetry(resilience.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		OnRetry: func(attempt int, err error) {
			metrics.retryAttempts.Inc()
			log.Printf("Retry attempt %d: %v", attempt, err)
		},
	})

	timeout := resilience.NewTimeout(resilience.TimeoutConfig{
		Duration: 2 * time.Second,
		OnTimeout: func() {
			metrics.timeoutTotal.Inc()
		},
	})

	return &ResilientService{
		service: service,
		breaker: breaker,
		retry:   retry,
		timeout: timeout,
		metrics: metrics,
	}
}

func (rs *ResilientService) ProcessWithResilience(ctx context.Context) error {
	start := time.Now()
	defer func() {
		rs.metrics.requestDuration.Observe(time.Since(start).Seconds())
	}()

	rs.metrics.requestTotal.Inc()

	// Create timeout context
	ctx, cancel := rs.timeout.WithTimeout(ctx)
	defer cancel()

	// Execute with retry and circuit breaker
	err := rs.retry.Execute(ctx, func() error {
		return rs.breaker.Execute(func() error {
			if err := rs.service.Process(ctx); err != nil {
				rs.metrics.failureTotal.Inc()
				return err
			}
			return nil
		})
	})

	if err != nil {
		switch {
		case resilience.IsCircuitOpen(err):
			log.Printf("Circuit breaker is open: %v", err)
			return fmt.Errorf("service unavailable: circuit breaker is open: %w", err)
		case resilience.IsTimeout(err):
			log.Printf("Operation timed out: %v", err)
			return fmt.Errorf("service timed out: %w", err)
		case resilience.IsMaxRetriesExceeded(err):
			log.Printf("Max retries exceeded: %v", err)
			return fmt.Errorf("service failed after max retries: %w", err)
		default:
			log.Printf("Operation failed: %v", err)
			return fmt.Errorf("service error: %w", err)
		}
	}

	return nil
}

func setupHTTPServer(rs *ResilientService) *http.Server {
	mux := http.NewServeMux()

	// Expose Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Resilient endpoint
	mux.HandleFunc("/resilient", func(w http.ResponseWriter, r *http.Request) {
		err := rs.ProcessWithResilience(r.Context())
		if err != nil {
			status := http.StatusInternalServerError
			if resilience.IsCircuitOpen(err) {
				status = http.StatusServiceUnavailable
			}
			http.Error(w, err.Error(), status)
			return
		}
		fmt.Fprintln(w, "Operation completed successfully")
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Healthy")
	})

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Create simulated service with 30% failure rate and variable delay
	service := NewSimulatedService(0.3, 100*time.Millisecond)

	// Create resilient service wrapper
	rs := NewResilientService(service)

	// Set up HTTP server
	server := setupHTTPServer(rs)

	// Print usage instructions
	fmt.Println("Comprehensive Resilience Patterns Example")
	fmt.Println("======================================")
	fmt.Println("\nServer is running on http://localhost:8080")
	fmt.Println("\nEndpoints:")
	fmt.Println("  - /resilient : Test resilient operation")
	fmt.Println("  - /metrics   : View Prometheus metrics")
	fmt.Println("  - /health    : Health check endpoint")
	fmt.Println("\nTry these commands:")
	fmt.Println("  1. Normal operation:")
	fmt.Println("     curl http://localhost:8080/resilient")
	fmt.Println("\n  2. Trigger circuit breaker (run multiple times):")
	fmt.Println("     for i in {1..10}; do curl http://localhost:8080/resilient; done")
	fmt.Println("\n  3. View Prometheus metrics:")
	fmt.Println("     curl http://localhost:8080/metrics")
	fmt.Println("\n  4. Check health:")
	fmt.Println("     curl http://localhost:8080/health")
	fmt.Println("\nMetrics are available in Prometheus format at /metrics")
	fmt.Println("You can use this endpoint with any Prometheus compatible monitoring system")

	log.Printf("Starting server on :8080...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
