package rate
package rate

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/rate"
)

// RateLimitMiddleware wraps an http.Handler with rate limiting
func RateLimitMiddleware(next http.Handler, limiter rate.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.RemoteAddr

		ctx := r.Context()
		if err := limiter.Allow(ctx, clientID); err != nil {
			if err == rate.ErrLimitExceeded {
				// Add standard rate limit headers
				resetTime := time.Now().Add(time.Hour)
				w.Header().Set("X-RateLimit-Limit", "100")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
				
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Create limiter
	limiter := rate.NewTokenBucket(rate.Config{
		Tokens:    100,       // Max tokens per hour
		Interval:  time.Hour, // Refill interval
		BurstSize: 10,       // Allow bursts
	})

	// Create a simple API handler
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from rate-limited API!")
	})

	// Wrap with rate limiting middleware
	http.Handle("/api", RateLimitMiddleware(apiHandler, limiter))

	// Start monitoring goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()
			metrics := limiter.GetMetrics(ctx)
			log.Printf(
				"Rate Limiter Stats - Requests: %d, Rejected: %d, Error Rate: %.2f%%",
				metrics.Requests,
				metrics.Rejected,
				metrics.ErrorRate*100,
			)
		}
	}()

	// Start server
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}