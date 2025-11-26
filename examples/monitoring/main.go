package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/monitoring"
	"github.com/mauriciomferz/Gauth_go/pkg/rate"
)

func main() {
	mon := monitoring.NewMonitor()
	if err := mon.Start(); err != nil {
		log.Printf("monitor start error: %v", err)
	}
	defer mon.Stop()

	// Use existing rate.Config fields
	cfg := rate.Config{RequestsPerSecond: 100, BurstSize: 200, WindowSize: time.Minute}
	limiter := rate.NewLimiter(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		clientID := r.RemoteAddr
		if err := limiter.AllowClient(context.Background(), clientID); err != nil {
			mon.IncrementErrors()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		mon.IncrementRequests()

		// simulate some work
		time.Sleep(25 * time.Millisecond)

		mon.RecordResponseTime(time.Since(start))

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerSecond))
		fmt.Fprintln(w, "Hello, your request is allowed!")
	})

	http.Handle("/api", handler)
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		m := mon.GetMetrics()
		// compute average response time
		var total time.Duration
		for _, d := range m.ResponseTimes {
			total += d
		}
		var avgMs float64
		if len(m.ResponseTimes) > 0 {
			avgMs = float64((total / time.Duration(len(m.ResponseTimes))).Milliseconds())
		}
		fmt.Fprintf(w, "requests_total %d\n", m.RequestCount)
		fmt.Fprintf(w, "errors_total %d\n", m.ErrorCount)
		fmt.Fprintf(w, "active_sessions %d\n", m.ActiveSessions)
		fmt.Fprintf(w, "response_time_avg_ms %.2f\n", avgMs)
	})

	srv := &http.Server{Addr: ":8080", ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Println("Server starting :8080 (metrics at /metrics)")
	log.Fatal(srv.ListenAndServe())
}
