package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/monitoring"
	"github.com/Gimel-Foundation/gauth/pkg/rate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create metrics collector
	metrics := monitoring.NewMetricsCollector()

	// Create rate limiter
	config := rate.Config{
		Algorithm:     rate.SlidingWindow,
		Limit:         100,
		Window:        time.Minute,
		EnableMetrics: true,
	}
	store := rate.NewMemoryStore()
	limiter, err := rate.NewLimiter(config.Algorithm, config, store)
	if err != nil {
		log.Fatal(err)
	}
	defer limiter.Close()

	// Create handler with rate limiting and metrics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.RemoteAddr
		start := time.Now()

		// Try rate limit
		quota, err := limiter.Allow(r.Context(), clientID)

		// Record request metric
		metrics.Counter(monitoring.MetricRateLimitHits, 1, map[string]string{
			"client_id": clientID,
			"allowed":   fmt.Sprintf("%t", err == nil),
		})

		// Record response time
		metrics.Gauge(monitoring.MetricResponseTime, time.Since(start).Seconds(), map[string]string{
			"client_id": clientID,
			"endpoint":  r.URL.Path,
		})

		if err == rate.ErrLimitExceeded {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", quota.ResetAt.Unix()))
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", quota.RetryAfter.Seconds()))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", quota.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", quota.ResetAt.Unix()))

		fmt.Fprintf(w, "Hello, your quota: %+v\n", quota)
	})

	// Create metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/api", handler)

	// Start monitoring printout
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			metrics := metrics.GetAllMetrics()
			log.Printf("Current Metrics:")
			for name, metric := range metrics {
				log.Printf("  %s: %.2f (labels: %v)", name, metric.Value, metric.Labels)
			}
		}
	}()

	// Start server
	log.Println("Starting server on :8080...")
	log.Println("Metrics available at /metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
