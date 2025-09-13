package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauth_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"handler", "method", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // from 1ms to ~1s
		},
		[]string{"handler", "method"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauth_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), // from 100B to ~1GB
		},
		[]string{"handler"},
	)

	activeRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gauth_http_active_requests",
			Help: "Number of currently active HTTP requests",
		},
		[]string{"handler"},
	)
)

func init() {
	// Register HTTP metrics
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpResponseSize,
		activeRequests,
	)
}

// responseWriter wraps http.ResponseWriter to capture metrics
type responseWriter struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // Default status
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// MetricsMiddleware wraps an http.Handler to collect metrics
func MetricsMiddleware(handler string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		// Track active requests
		activeRequests.WithLabelValues(handler).Inc()
		defer activeRequests.WithLabelValues(handler).Dec()

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		status := strconv.Itoa(rw.status)

		httpRequestsTotal.WithLabelValues(handler, r.Method, status).Inc()
		httpRequestDuration.WithLabelValues(handler, r.Method).Observe(duration.Seconds())
		httpResponseSize.WithLabelValues(handler).Observe(float64(rw.written))
	})
}

// AuthMetricsMiddleware wraps authentication handlers to collect detailed auth metrics
func AuthMetricsMiddleware(handler string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		// Track active requests
		activeRequests.WithLabelValues(handler).Inc()
		defer activeRequests.WithLabelValues(handler).Dec()

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		status := strconv.Itoa(rw.status)

		// Record general HTTP metrics
		httpRequestsTotal.WithLabelValues(handler, r.Method, status).Inc()
		httpRequestDuration.WithLabelValues(handler, r.Method).Observe(duration.Seconds())
		httpResponseSize.WithLabelValues(handler).Observe(float64(rw.written))

		// Record auth-specific metrics
		method := r.Header.Get("X-Auth-Method")
		if method == "" {
			method = "unknown"
		}

		authAttempts.WithLabelValues(method, status).Inc()
		authLatency.WithLabelValues(method).Observe(duration.Seconds())
	})
}

// AuthzMetricsMiddleware wraps authorization handlers to collect detailed authz metrics
func AuthzMetricsMiddleware(handler string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		// Track active requests
		activeRequests.WithLabelValues(handler).Inc()
		defer activeRequests.WithLabelValues(handler).Dec()

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		status := strconv.Itoa(rw.status)

		// Record general HTTP metrics
		httpRequestsTotal.WithLabelValues(handler, r.Method, status).Inc()
		httpRequestDuration.WithLabelValues(handler, r.Method).Observe(duration.Seconds())
		httpResponseSize.WithLabelValues(handler).Observe(float64(rw.written))

		// Record authz-specific metrics
		policy := r.Header.Get("X-Policy")
		if policy == "" {
			policy = "unknown"
		}

		allowed := rw.status == http.StatusOK
		authzDecisions.WithLabelValues(boolToString(allowed), policy).Inc()
		authzLatency.WithLabelValues(policy).Observe(duration.Seconds())
	})
}
