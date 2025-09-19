// Package circuit provides circuit breaker functionality for GAuth
package circuit

import (
	"sync"
	"time"
)

// State represents the state of the circuit breaker
type State int

const (
	StateClosed   State = iota // Circuit is closed (allowing requests)
	StateOpen                  // Circuit is open (blocking requests)
	StateHalfOpen              // Circuit is half-open (testing if service is healthy)
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type CircuitBreaker struct {
	name             string
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenLimit    int
	failureCount     int
	state            State
	lastStateChange  time.Time
	halfOpenCount    int
	mu               sync.RWMutex
	onStateChange    func(name string, from, to State)
}

type Options struct {
	Name             string
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenLimit    int
	OnStateChange    func(name string, from, to State)
}

func NewCircuitBreaker(opts Options) *CircuitBreaker {
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = 5
	}
	if opts.ResetTimeout == 0 {
		opts.ResetTimeout = 10 * time.Second
	}
	if opts.HalfOpenLimit <= 0 {
		opts.HalfOpenLimit = 1
	}

	return &CircuitBreaker{
		name:             opts.Name,
		failureThreshold: opts.FailureThreshold,
		resetTimeout:     opts.ResetTimeout,
		halfOpenLimit:    opts.HalfOpenLimit,
		state:            StateClosed,
		lastStateChange:  time.Now(),
		onStateChange:    opts.OnStateChange,
	}
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.halfOpenCount = 0
	cb.lastStateChange = time.Now()

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, StateClosed)
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return nil // replace with errors.ErrCircuitOpen if needed
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.toHalfOpen()
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenCount < cb.halfOpenLimit
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case StateHalfOpen:
		cb.toClosed()
	case StateClosed:
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.toOpen()
		}
	case StateHalfOpen:
		cb.toOpen()
	}
}

func (cb *CircuitBreaker) toOpen() {
	if cb.state != StateOpen {
		oldState := cb.state
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, StateOpen)
		}
	}
}

func (cb *CircuitBreaker) toHalfOpen() {
	if cb.state != StateHalfOpen {
		oldState := cb.state
		cb.state = StateHalfOpen
		cb.halfOpenCount = 0
		cb.lastStateChange = time.Now()
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, StateHalfOpen)
		}
	}
}

func (cb *CircuitBreaker) toClosed() {
	if cb.state != StateClosed {
		oldState := cb.state
		cb.state = StateClosed
		cb.failureCount = 0
		cb.halfOpenCount = 0
		cb.lastStateChange = time.Now()
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, StateClosed)
		}
	}
}

func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Monitor tracks circuit breaker statistics
type Monitor struct {
	mu    sync.RWMutex
	stats map[string]*Stats
}

func NewMonitor() *Monitor {
	return &Monitor{
		stats: make(map[string]*Stats),
	}
}

func (m *Monitor) OnStateChange(name string, from, to State, lastFailure time.Time) {
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

func (m *Monitor) GetStats(name string) *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, ok := m.stats[name]; ok {
		return stats
	}
	return nil
}

func (m *Monitor) GetAllStats() map[string]*Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Stats, len(m.stats))
	for k, v := range m.stats {
		result[k] = v
	}
	return result
}

func (m *Monitor) Reset(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats[name] = &Stats{
		CurrentState:    StateClosed,
		LastStateChange: time.Now(),
	}
}

// Stats represents circuit breaker statistics
type Stats struct {
	Requests    uint64  `json:"requests"`
	Failures    uint64  `json:"failures"`
	FailureRate float64 `json:"failure_rate"`

	CurrentState       State     `json:"current_state"`
	LastStateChange    time.Time `json:"last_state_change"`
	TimeInCurrentState Duration  `json:"time_in_state"`

	WindowSize        int     `json:"window_size"`
	WindowFailures    int     `json:"window_failures"`
	WindowSuccesses   int     `json:"window_successes"`
	WindowFailureRate float64 `json:"window_failure_rate"`

	LastFailure time.Time `json:"last_failure"`
	LastSuccess time.Time `json:"last_success"`
	MTBF        Duration  `json:"mtbf"`

	HalfOpenAttempts  int `json:"half_open_attempts"`
	HalfOpenSuccesses int `json:"half_open_successes"`
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}
