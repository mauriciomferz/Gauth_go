package ratelimit

import (
	"context"
	"sync"
	"time"
)

type SlidingWindow struct {
	mu      sync.RWMutex
	windows map[string]*slidingWindow
	config  *Config
}

type slidingWindow struct {
	requests  []time.Time
	config    *Config
	lastClean time.Time
}

func NewSlidingWindow(config *Config) *SlidingWindow {
	sw := &SlidingWindow{
		windows: make(map[string]*slidingWindow),
		config:  config,
	}
	go sw.startCleanup()
	return sw
}

func (sw *SlidingWindow) startCleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		sw.mu.Lock()
		now := time.Now()
		for id, window := range sw.windows {
			if now.Sub(window.lastClean) > time.Duration(sw.config.WindowSize*2)*time.Second {
				delete(sw.windows, id)
			}
		}
		sw.mu.Unlock()
	}
}

func (sw *SlidingWindow) Allow(ctx context.Context, id string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	window, exists := sw.windows[id]
	if !exists {
		window = &slidingWindow{
			requests:  make([]time.Time, 0, sw.config.RequestsPerSecond*sw.config.WindowSize),
			config:    sw.config,
			lastClean: now,
		}
		sw.windows[id] = window
	}

	windowStart := now.Add(-time.Duration(sw.config.WindowSize) * time.Second)
	validRequests := make([]time.Time, 0, len(window.requests))
	for _, t := range window.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	maxRequests := sw.config.BurstSize
	if maxRequests <= 0 {
		maxRequests = sw.config.RequestsPerSecond * sw.config.WindowSize
	}
	effectiveLimit := maxRequests
	if effectiveLimit <= 0 {
		effectiveLimit = sw.config.RequestsPerSecond * sw.config.WindowSize
	}
	if len(validRequests) >= effectiveLimit {
		return ErrRateLimitExceeded
	}
	if maxRequests == 0 || len(validRequests) >= sw.config.RequestsPerSecond {
		windowDuration := now.Sub(windowStart).Seconds()
		if windowDuration > 0 {
			currentRate := float64(len(validRequests)) / windowDuration
			if currentRate >= float64(sw.config.RequestsPerSecond) {
				return ErrRateLimitExceeded
			}
		}
	}
	validRequests = append(validRequests, now)
	window.requests = validRequests
	window.lastClean = now
	return nil
}

// GetQuota implements Algorithm.GetQuota
func (sw *SlidingWindow) GetQuota(id string) Quota {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	now := time.Now()
	window, exists := sw.windows[id]
	if !exists {
		maxRequests := sw.config.RequestsPerSecond * sw.config.WindowSize
		return Quota{
			Remaining: maxRequests,
			Total:     maxRequests,
			ResetAt:   now.Add(time.Duration(sw.config.WindowSize) * time.Second),
			Window: struct {
				Start    time.Time     `json:"start"`
				Duration time.Duration `json:"duration"`
				Requests int           `json:"requests"`
			}{
				Start:    now,
				Duration: time.Duration(sw.config.WindowSize) * time.Second,
				Requests: 0,
			},
		}
	}
	windowStart := now.Add(-time.Duration(sw.config.WindowSize) * time.Second)
	validCount := 0
	for _, t := range window.requests {
		if t.After(windowStart) {
			validCount++
		}
	}
	maxRequests := sw.config.RequestsPerSecond * sw.config.WindowSize
	return Quota{
		Remaining: maxRequests - validCount,
		Total:     maxRequests,
		ResetAt:   now.Add(time.Duration(sw.config.WindowSize) * time.Second),
		Window: struct {
			Start    time.Time     `json:"start"`
			Duration time.Duration `json:"duration"`
			Requests int           `json:"requests"`
		}{
			Start:    windowStart,
			Duration: time.Duration(sw.config.WindowSize) * time.Second,
			Requests: validCount,
		},
	}
}

// Reset implements Algorithm.Reset
func (sw *SlidingWindow) Reset(id string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	delete(sw.windows, id)
}
