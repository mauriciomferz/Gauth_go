// Package circuit provides monitoring types for circuit breakers.
package circuit

import (
	"sync"
	"time"
)

// Stats represents circuit breaker statistics
type Stats struct {
	// Core metrics
	Requests    uint64  `json:"requests"`     // Total requests
	Failures    uint64  `json:"failures"`     // Total failures
	FailureRate float64 `json:"failure_rate"` // Current failure rate

	// State information
	CurrentState       State     `json:"current_state"`     // Current breaker state
	LastStateChange    time.Time `json:"last_state_change"` // Last state transition
	TimeInCurrentState Duration  `json:"time_in_state"`     // Time in current state

	// Window metrics
	WindowSize        int     `json:"window_size"`         // Size of sliding window
	WindowFailures    int     `json:"window_failures"`     // Failures in window
	WindowSuccesses   int     `json:"window_successes"`    // Successes in window
	WindowFailureRate float64 `json:"window_failure_rate"` // Window failure rate

	// Timing metrics
	LastFailure time.Time `json:"last_failure"` // Last failure time
	LastSuccess time.Time `json:"last_success"` // Last success time
	MTBF        Duration  `json:"mtbf"`         // Mean time between failures

	// Half-open metrics
	HalfOpenAttempts  int `json:"half_open_attempts"`  // Test attempts
	HalfOpenSuccesses int `json:"half_open_successes"` // Successful tests
}

// Duration wraps time.Duration for JSON marshaling
type Duration time.Duration

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

// Monitor tracks circuit breaker statistics
type Monitor struct {
	mu    sync.RWMutex
	stats map[string]*Stats
}

// NewMonitor creates a new circuit breaker monitor
func NewMonitor() *Monitor {
	return &Monitor{
		stats: make(map[string]*Stats),
	}
}

// OnStateChange handles state transitions
func (m *Monitor) OnStateChange(name string, from, to State, _ time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.stats[name]
	if !ok {
		stats = &Stats{}
		m.stats[name] = stats
	}

	stats.CurrentState = to
	stats.LastStateChange = time.Now()
	stats.TimeInCurrentState = Duration(0)

	if from == StateHalfOpen && to == StateClosed {
		stats.HalfOpenSuccesses++
	}
}

// OnSuccess records a successful request
func (m *Monitor) OnSuccess(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.stats[name]
	if !ok {
		stats = &Stats{}
		m.stats[name] = stats
	}

	stats.Requests++
	stats.LastSuccess = time.Now()
	stats.WindowSuccesses = (stats.WindowSuccesses + 1) % stats.WindowSize

	if stats.Requests > 0 {
		stats.FailureRate = float64(stats.Failures) / float64(stats.Requests)
	}

	if stats.WindowSuccesses+stats.WindowFailures > 0 {
		stats.WindowFailureRate = float64(stats.WindowFailures) /
			float64(stats.WindowSuccesses+stats.WindowFailures)
	}
}

// OnFailure records a failed request
func (m *Monitor) OnFailure(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.stats[name]
	if !ok {
		stats = &Stats{}
		m.stats[name] = stats
	}

	stats.Requests++
	stats.Failures++
	stats.LastFailure = time.Now()
	stats.WindowFailures = (stats.WindowFailures + 1) % stats.WindowSize

	if stats.LastSuccess.IsZero() {
		stats.MTBF = 0
	} else {
		stats.MTBF = Duration(time.Since(stats.LastSuccess))
	}

	if stats.Requests > 0 {
		stats.FailureRate = float64(stats.Failures) / float64(stats.Requests)
	}

	if stats.WindowSuccesses+stats.WindowFailures > 0 {
		stats.WindowFailureRate = float64(stats.WindowFailures) /
			float64(stats.WindowSuccesses+stats.WindowFailures)
	}
}

// GetStats returns stats for a circuit breaker
func (m *Monitor) GetStats(name string) *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, ok := m.stats[name]; ok {
		return stats
	}
	return nil
}

// GetAllStats returns stats for all circuit breakers
func (m *Monitor) GetAllStats() map[string]*Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Stats, len(m.stats))
	for k, v := range m.stats {
		result[k] = v
	}
	return result
}

// Reset resets stats for a circuit breaker
func (m *Monitor) Reset(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats[name] = &Stats{
		CurrentState:    StateClosed,
		LastStateChange: time.Now(),
	}
}
