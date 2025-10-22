// Package monitoring provides monitoring and metrics functionality
package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// NewMonitor creates a new monitoring service
func NewMonitor() *Monitor {
	return &Monitor{
		metrics: &Metrics{
			CustomMetrics: make(map[string]interface{}),
		},
	}
}

// NewMetrics provides a constructor for Metrics (compatibility for enhanced collector)
func NewMetrics() *Metrics {
	return &Metrics{CustomMetrics: make(map[string]interface{})}
}

// Start starts the monitoring service
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("monitor already started")
	}

	m.started = true
	fmt.Println("Monitoring service started")
	return nil
}

// Stop stops the monitoring service
func (m *Monitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("monitor not started")
	}

	m.started = false
	fmt.Println("Monitoring service stopped")
	return nil
}

// IncrementRequests increments the request counter
func (m *Monitor) IncrementRequests() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.RequestCount++
}

// IncrementErrors increments the error counter
func (m *Monitor) IncrementErrors() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.ErrorCount++
}

// RecordResponseTime records a response time
func (m *Monitor) RecordResponseTime(duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.ResponseTimes = append(m.metrics.ResponseTimes, duration)
}

// SetActiveSession sets the number of active sessions
func (m *Monitor) SetActiveSession(count int64) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.ActiveSessions = count
}

// GetMetrics returns current metrics (thread-safe copy)
func (m *Monitor) GetMetrics() Metrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// Create a copy to avoid race conditions
	return Metrics{
		RequestCount:   m.metrics.RequestCount,
		ErrorCount:     m.metrics.ErrorCount,
		ResponseTimes:  append([]time.Duration{}, m.metrics.ResponseTimes...),
		ActiveSessions: m.metrics.ActiveSessions,
		CustomMetrics:  make(map[string]interface{}), // Simplified copy
	}
}

// SetCustomMetric sets a custom metric
func (m *Monitor) SetCustomMetric(key string, value interface{}) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.CustomMetrics[key] = value
}

// GetCustomMetric gets a custom metric
func (m *Monitor) GetCustomMetric(key string) (interface{}, bool) {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()
	value, exists := m.metrics.CustomMetrics[key]
	return value, exists
}

// Demo demonstrates monitoring functionality
func Demo() error {
	fmt.Println("=== Monitoring Demo ===")

	monitor := NewMonitor()

	// Start monitoring
	if err := monitor.Start(); err != nil {
		return err
	}
	defer func() {
		if err := monitor.Stop(); err != nil {
			fmt.Printf("monitor stop error: %v\n", err)
		}
	}()

	// Simulate some activity
	for i := 0; i < 10; i++ {
		monitor.IncrementRequests()
		if i%3 == 0 {
			monitor.IncrementErrors()
		}
		monitor.RecordResponseTime(time.Millisecond * time.Duration(50+i*10))
	}

	monitor.SetActiveSession(5)
	monitor.SetCustomMetric("version", "1.0.0")
	monitor.SetCustomMetric("region", "us-west-2")

	// Get and display metrics
	metrics := monitor.GetMetrics()
	fmt.Printf("Requests: %d\n", metrics.RequestCount)
	fmt.Printf("Errors: %d\n", metrics.ErrorCount)
	fmt.Printf("Active Sessions: %d\n", metrics.ActiveSessions)
	fmt.Printf("Average Response Time: %v\n", averageResponseTime(metrics.ResponseTimes))

	return nil
}

// averageResponseTime calculates average response time
func averageResponseTime(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range times {
		total += t
	}

	return total / time.Duration(len(times))
}

// Metric names (basic set – extend as needed)
const (
	MetricRequestsTotal       = "requests_total"
	MetricErrorsTotal         = "errors_total"
	MetricActiveSessionsGauge = "active_sessions"
	MetricResponseTimeMs      = "response_time_ms"
)

// Metric represents a single collected metric
type Metric struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"` // counter, gauge, histogram
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// Metrics aggregates runtime metrics
type Metrics struct {
	mu             sync.RWMutex
	RequestCount   int64
	ErrorCount     int64
	ResponseTimes  []time.Duration
	ActiveSessions int64
	CustomMetrics  map[string]interface{}
}

// Monitor coordinates collection of high-level metrics
type Monitor struct {
	mu      sync.Mutex
	metrics *Metrics
	started bool
}
