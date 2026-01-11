package clock

import (
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/beevik/ntp"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
)

type MonitorStatus string

const (
	StatusHealthy  MonitorStatus = "Healthy"
	StatusWarning  MonitorStatus = "Warning"
	StatusCritical MonitorStatus = "Critical"
)

// SystemClockMonitor periodically checks local system time against an NTP server
// to detect dangerous clock skew.
type SystemClockMonitor struct {
	ntpServer   string
	maxSkew     time.Duration
	ntpInterval time.Duration

	currentSkew atomic.Int64 // nanoseconds
	lastCheck   time.Time
	lastError   error
	status      MonitorStatus

	mu      sync.RWMutex
	stopCh  chan struct{}
	metrics metrics.Metrics
	queryFn func(string) (*ntp.Response, error)
}

// NewSystemClockMonitor creates a new clock monitor.
// Default server: "pool.ntp.org"
// Default maxSkew: 5 minutes (AAP001 standard)
// Default interval: 1 hour
func NewSystemClockMonitor(server string, maxSkew time.Duration, interval time.Duration, m metrics.Metrics) *SystemClockMonitor {
	if server == "" {
		server = "pool.ntp.org"
	}
	if maxSkew == 0 {
		maxSkew = 5 * time.Minute
	}
	if interval == 0 {
		interval = 1 * time.Hour
	}

	return &SystemClockMonitor{
		ntpServer:   server,
		maxSkew:     maxSkew,
		ntpInterval: interval,
		status:      StatusHealthy,
		stopCh:      make(chan struct{}),
		metrics:     m,
		queryFn:     ntp.Query,
	}
}

// Start begins the background monitoring loop.
func (s *SystemClockMonitor) Start() {
	if os.Getenv("AGENTAUTH_DISABLE_BG_POLLS") != "1" {
		go func() {
			// Initial check immediately
			s.check()

			ticker := time.NewTicker(s.ntpInterval)
			defer ticker.Stop()

			for {
				select {
				case <-s.stopCh:
					return
				case <-ticker.C:
					s.check()
				}
			}
		}()
	}
}

// Stop halts the monitoring loop.
func (s *SystemClockMonitor) Stop() {
	close(s.stopCh)
}

// check performs the NTP query and updates internal state.
func (s *SystemClockMonitor) check() {
	if s.queryFn == nil {
		s.queryFn = ntp.Query
	}
	response, err := s.queryFn(s.ntpServer)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastCheck = time.Now()

	if err != nil {
		s.lastError = err
		log.Printf("[clock] Failed to query NTP server %s: %v", s.ntpServer, err)
		// If we fail repeatedly, we might want to downgrade status, but for now just log
		return
	}

	s.lastError = nil
	offset := response.ClockOffset
	s.currentSkew.Store(int64(offset))

	if s.metrics != nil {
		// Export skew in seconds for Prometheus
		s.metrics.SetSystemClockSkew(offset.Seconds())
	}

	// Determine status
	absSkew := offset
	if absSkew < 0 {
		absSkew = -absSkew
	}

	switch {
	case absSkew > s.maxSkew:
		s.status = StatusCritical
		log.Printf("[clock] CRITICAL: System clock skew %v exceeds threshold %v", offset, s.maxSkew)
	case absSkew > s.maxSkew/2:
		s.status = StatusWarning
		log.Printf("[clock] WARNING: System clock skew %v approaching threshold %v", offset, s.maxSkew)
	default:
		s.status = StatusHealthy
	}
}

// Skew returns the last measured clock skew (local - remote).
// Positive means local is ahead.
func (s *SystemClockMonitor) Skew() time.Duration {
	return time.Duration(s.currentSkew.Load())
}

// Status returns the current health status of the clock.
func (s *SystemClockMonitor) Status() (string, time.Duration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return string(s.status), time.Duration(s.currentSkew.Load()), s.lastError
}
