package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
)

// Rotator defines the contract for an entity that can rotate its active key.
type Rotator interface {
	// Rotate performs the key rotation.
	Rotate(ctx context.Context) error
}

// Scheduler manages periodic key rotation.
type Scheduler struct {
	rotator  Rotator
	interval time.Duration
	metrics  metrics.Metrics
	ticker   *time.Ticker
	stop     chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewScheduler creates a new rotation scheduler.
func NewScheduler(rotator Rotator, interval time.Duration, m metrics.Metrics) *Scheduler {
	if m == nil {
		m = metrics.Noop
	}
	return &Scheduler{
		rotator:  rotator,
		interval: interval,
		metrics:  m,
		stop:     make(chan struct{}),
	}
}

// Start begins the rotation schedule in a background goroutine.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("scheduler already running")
	}
	s.running = true
	s.ticker = time.NewTicker(s.interval)
	s.mu.Unlock()

	go s.run(ctx)
	return nil
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stop)
	s.running = false
}

func (s *Scheduler) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.Stop()
			return
		case <-s.stop:
			return
		case <-s.ticker.C:
			s.triggerRotation(ctx)
		}
	}
}

func (s *Scheduler) triggerRotation(ctx context.Context) {
	start := time.Now()
	err := s.rotator.Rotate(ctx)
	// TODO: Add metric for duration
	_ = start
	if err != nil {
		// Log error via metrics if possible, or print?
		// For now we assume metrics interface might get extended or we just use general failure counter
		// s.metrics.IncKeyRotationFailures() (hypothetical)
		// Since we don't have IncKeyRotationFailures in metrics.Metrics yet, we'll skip specific metric
		// or repurpose one like IncViolation("key_rotation_failed")
		s.metrics.IncViolation("key_rotation_failed")
	} else {
		// s.metrics.IncKeyRotationSuccess()
	}
}
