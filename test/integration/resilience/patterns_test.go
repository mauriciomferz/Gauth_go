package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/resilience"
)

// Test retry pattern
func TestRetryBehavior(t *testing.T) {
       retry := resilience.NewRetry(resilience.RetryStrategy{
	       MaxAttempts:     3,
	       InitialInterval: 50 * time.Millisecond,
	       MaxInterval:     200 * time.Millisecond,
	       Multiplier:      2.0,
       })

       attempts := 0
       start := time.Now()

       err := retry.Do(func() error {
	       attempts++
	       if attempts < 3 {
		       return errors.New("temporary error")
	       }
	       return nil
       })

	_ = time.Since(start)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestPatternsRateLimiter(t *testing.T) {
       patterns := resilience.NewPatterns("test-rate-limiter",
	       resilience.WithRateLimit(10, 3, nil),
       )

       // Test burst
       for i := 0; i < 3; i++ {
	       err := patterns.Execute(context.Background(), func() error { return nil })
	       if err != nil {
		       t.Errorf("Expected request %d to be allowed, got %v", i, err)
	       }
       }
       err := patterns.Execute(context.Background(), func() error { return nil })
       if err == nil {
	       t.Error("Should not allow requests after burst")
       }

       // Test recovery
       time.Sleep(200 * time.Millisecond)
       err = patterns.Execute(context.Background(), func() error { return nil })
       if err != nil {
	       t.Error("Should allow requests after recovery")
       }
}
