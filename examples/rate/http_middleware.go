package main

import (
	"fmt"
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

// func main() {
//    // Example main for running the rate-limited API server
//    // This is commented out to avoid duplicate main redeclaration errors.
// }
//           ctx := context.Background()
//           metrics := limiter.GetMetrics(ctx)
//           log.Printf(
//               "Rate Limiter Stats - Requests: %d, Rejected: %d, Error Rate: %.2f%%",
//               metrics.Requests,
//               metrics.Rejected,
//               metrics.ErrorRate*100,
//           )
//       }
//   }()
//
//   // Start server
//   log.Println("Starting server on :8080...")
//   if err := http.ListenAndServe(":8080", nil); err != nil {
//       log.Fatal(err)
//   }
//}
