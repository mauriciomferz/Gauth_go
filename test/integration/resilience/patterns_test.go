package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
)

func TestResiliencePatterns(t *testing.T) {
	t.Run("CompositeResilience", testCompositeResilience)
	t.Run("RetryBehavior", testRetryBehavior)
	t.Run("CircuitBreakerTransitions", testCircuitBreakerTransitions)
	t.Run("BulkheadIsolation", testBulkheadIsolation)
}

func testCompositeResilience(t *testing.T) {
	composite := createCompositeResiliencePattern()
	
	testSuccessfulExecution(t, composite)
	testRateLimiting(t, composite)
	testCircuitBreakerFailures(t, composite)
	testConcurrentExecutionWithBulkhead(t, composite)
}

func createCompositeResiliencePattern() *resilience.Composite {
	return resilience.NewComposite(resilience.CompositeOptions{
		CircuitOptions: circuit.Options{
			Name:             "test-circuit",
			FailureThreshold: 3,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenLimit:    1,
		},
		MaxConcurrent: 5,
		RetryStrategy: resilience.RetryStrategy{
			MaxAttempts:     3,
			InitialInterval: 50 * time.Millisecond,
			MaxInterval:     200 * time.Millisecond,
			Multiplier:      2.0,
		},
		RateLimit: 10,
		BurstSize: 3,
	})
}

func testSuccessfulExecution(t *testing.T, composite *resilience.Composite) {
	err := composite.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func testRateLimiting(t *testing.T, composite *resilience.Composite) {
	for i := 0; i < 15; i++ {
		err := composite.Execute(context.Background(), func() error {
			return nil
		})
		if i >= 10 && err == nil {
			t.Errorf("Expected rate limit error after 10 requests on iteration %d", i)
		}
	}
}

func testCircuitBreakerFailures(t *testing.T, composite *resilience.Composite) {
	for i := 0; i < 5; i++ {
		err := composite.Execute(context.Background(), func() error {
			return errors.New("test error")
		})
		if i >= 3 && err == nil {
			t.Errorf("Expected circuit open after 3 failures on iteration %d", i)
		}
	}
}

func testConcurrentExecutionWithBulkhead(t *testing.T, composite *resilience.Composite) {
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := composite.Execute(context.Background(), func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for range errors {
		errorCount++
	}
	if errorCount == 0 {
		t.Error("Expected some concurrent requests to fail due to bulkhead")
	}
}

func testRetryBehavior(t *testing.T) {
	retry := resilience.NewRetry(resilience.RetryStrategy{
		MaxAttempts:     3,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     200 * time.Millisecond,
		Multiplier:      2.0,
	})

	attempts := 0
	start := time.Now()

	err := retry.Execute(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	duration := time.Since(start)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	if duration < 150*time.Millisecond {
		t.Error("Expected retry delays to be at least 150ms")
	}
}

func testCircuitBreakerTransitions(t *testing.T) {
	breaker := circuit.NewBreaker(circuit.Options{
		Name:             "test-transitions",
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenLimit:    1,
	})

	testInitialState(t, breaker)
	testStateTransitionToOpen(t, breaker)
	testStateTransitionToClosed(t, breaker)
}

func testInitialState(t *testing.T, breaker *circuit.Breaker) {
	if got := breaker.GetState(); got != circuit.StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", got)
	}
}

func testStateTransitionToOpen(t *testing.T, breaker *circuit.Breaker) {
	_ = breaker.Execute(func() error {
		return errors.New("error")
	})
	_ = breaker.Execute(func() error {
		return errors.New("error")
	})

	if got := breaker.GetState(); got != circuit.StateOpen {
		t.Errorf("Expected state to be Open after failures, got %v", got)
	}
}

func testStateTransitionToClosed(t *testing.T, breaker *circuit.Breaker) {
	time.Sleep(150 * time.Millisecond)

	err := breaker.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error during state transition, got %v", err)
	}
	if got := breaker.GetState(); got != circuit.StateClosed {
		t.Errorf("Expected state to be Closed after success, got %v", got)
	}
}

func testBulkheadIsolation(t *testing.T) {
	bulkhead := resilience.NewBulkhead(2)
	var wg sync.WaitGroup
	executing := make(chan struct{}, 3)
	completed := make(chan struct{}, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := bulkhead.Execute(context.Background(), func() error {
				executing <- struct{}{}
				time.Sleep(100 * time.Millisecond)
				completed <- struct{}{}
				return nil
			})
			if err != nil {
				t.Logf("Bulkhead execution error: %v", err)
			}
		}()
	}

	// Wait for executions to start
	time.Sleep(50 * time.Millisecond)

	// Should only have 2 executing at once
	if got := len(executing); got != 2 {
		t.Errorf("Expected 2 concurrent executions, got %d", got)
	}

	wg.Wait()
	if got := len(completed); got != 3 {
		t.Errorf("Expected 3 completed executions, got %d", got)
	}
}

func TestRateLimiterBehavior(t *testing.T) {
	limiter := resilience.NewRateLimiter(10, 3)

	// Test burst
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Expected request %d to be allowed", i)
		}
	}
	if limiter.Allow() {
		t.Error("Should not allow requests after burst")
	}

	// Test recovery
	time.Sleep(200 * time.Millisecond)
	if !limiter.Allow() {
		t.Error("Should allow requests after recovery")
	}
}
