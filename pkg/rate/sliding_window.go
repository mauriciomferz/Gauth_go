package rate

import (
	"context"
	"sync"
	"time"
)

// SlidingWindow implements the sliding window rate limiting algorithm
type SlidingWindow struct {
	requests int64
	window   time.Duration
	counts   sync.Map
	mu       sync.RWMutex
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(cfg Config) *SlidingWindow {
	return &SlidingWindow{
		requests: cfg.Rate,
		window:   cfg.Window,
	}
}

// windowInfo holds the count and timestamps for a rate limit window
type windowInfo struct {
	count      int64
	timestamps []time.Time
}

// Allow implements the Limiter interface
func (sw *SlidingWindow) Allow(ctx context.Context, id string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// Get or initialize window info
	infoIface, _ := sw.counts.LoadOrStore(id, &windowInfo{})
	info := infoIface.(*windowInfo)

	// Remove timestamps outside the window
	validIdx := 0
	for i, ts := range info.timestamps {
		if ts.After(cutoff) {
			validIdx = i
			break
		}
	}
	info.timestamps = info.timestamps[validIdx:]
	info.count = int64(len(info.timestamps))

	// Check if we can allow the request
	if info.count >= sw.requests {
		return ErrRateLimitExceeded
	}

	// Add new timestamp
	info.timestamps = append(info.timestamps, now)
	info.count++

	sw.counts.Store(id, info)
	return nil
}

// GetRemainingRequests implements the Limiter interface
func (sw *SlidingWindow) GetRemainingRequests(id string) int64 {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	infoIface, ok := sw.counts.Load(id)
	if !ok {
		return sw.requests
	}

	info := infoIface.(*windowInfo)
	return sw.requests - info.count
}

// Reset implements the Limiter interface
func (sw *SlidingWindow) Reset(id string) {
	sw.counts.Store(id, &windowInfo{})
}

// cleanup periodically removes old entries
//nolint:unused // reserved for automatic cleanup
func (sw *SlidingWindow) cleanup() {
	ticker := time.NewTicker(sw.window)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		cutoff := now.Add(-sw.window)

		sw.counts.Range(func(key, value interface{}) bool {
			info := value.(*windowInfo)
			sw.mu.Lock()
			validIdx := 0
			for i, ts := range info.timestamps {
				if ts.After(cutoff) {
					validIdx = i
					break
				}
			}
			info.timestamps = info.timestamps[validIdx:]
			info.count = int64(len(info.timestamps))
			sw.mu.Unlock()
			return true
		})
	}
}
