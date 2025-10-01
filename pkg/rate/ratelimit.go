// Package rate provides rate limiting functionality for the GAuth protocol.
package rate

import (
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
)

// LimitEntry tracks per-client state.
type LimitEntry struct {
	Count       int
	WindowStart time.Time
	LastAccess  time.Time
	BurstTokens int
	WindowSize  time.Duration
	MaxRequests int
}

// TokenBucketLimiter provides rate limiting functionality
type TokenBucketLimiter struct {
	Config  common.RateLimitConfig
	entries map[string]*LimitEntry
	mutex   sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(cfg common.RateLimitConfig) *TokenBucketLimiter {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 60
	}
	if cfg.BurstSize <= 0 {
		cfg.BurstSize = cfg.RequestsPerSecond
	}
	if cfg.RequestsPerSecond <= 0 {
		cfg.RequestsPerSecond = 60
	}
	return &TokenBucketLimiter{
		Config:  cfg,
		entries: make(map[string]*LimitEntry),
	}
}

// IsAllowed checks if a client is allowed to proceed
func (rl *TokenBucketLimiter) IsAllowed(clientID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	now := time.Now()
	windowDuration := time.Duration(rl.Config.WindowSize) * time.Second
	entry, exists := rl.entries[clientID]
	if !exists {
		entry = &LimitEntry{
			Count:       1,
			WindowStart: now,
			LastAccess:  now,
			BurstTokens: rl.Config.BurstSize,
			WindowSize:  windowDuration,
			MaxRequests: rl.Config.RequestsPerSecond,
		}
		rl.entries[clientID] = entry
		return true
	}
	// Reset window if expired
	if now.Sub(entry.WindowStart) >= windowDuration {
		entry.Count = 1
		entry.WindowStart = now
		entry.LastAccess = now
		entry.BurstTokens = rl.Config.BurstSize
		return true
	}
	// Check if within rate limit
	if entry.Count < entry.MaxRequests {
		entry.Count++
		entry.LastAccess = now
		return true
	}
	// Try using burst token
	if entry.BurstTokens > 0 {
		entry.BurstTokens--
		entry.Count++
		entry.LastAccess = now
		return true
	}
	return false
}

// GetClientState returns the current state for a client
func (rl *TokenBucketLimiter) GetClientState(clientID string) *LimitEntry {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	return rl.entries[clientID]
}

// Cleanup removes expired entries from the rate limiter
func (rl *TokenBucketLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	now := time.Now()
	windowDuration := time.Duration(rl.Config.WindowSize) * time.Second
	for clientID, entry := range rl.entries {
		if now.Sub(entry.LastAccess) > windowDuration*2 {
			delete(rl.entries, clientID)
		}
	}
}

// GetStats returns statistics about the rate limiter
func (rl *TokenBucketLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	stats := make(map[string]interface{})
	stats["total_clients"] = len(rl.entries)
	stats["requests_per_second"] = rl.Config.RequestsPerSecond
	stats["burst_size"] = rl.Config.BurstSize
	stats["window_size"] = rl.Config.WindowSize
	active := 0
	blocked := 0
	now := time.Now()
	for _, entry := range rl.entries {
		if now.Sub(entry.LastAccess) < time.Duration(rl.Config.WindowSize)*time.Second {
			active++
			if entry.Count >= entry.MaxRequests && entry.BurstTokens == 0 {
				blocked++
			}
		}
	}
	stats["active_clients"] = active
	stats["blocked_clients"] = blocked
	return stats
}
