package ratelimit

import (
	"sync"
	"time"
)

// RateLimitEntry tracks rate limit state for a client
type RateLimitEntry struct {
	Count       int           // Number of requests in current window
	MaxRequests int           // Maximum requests per window
	WindowStart time.Time     // Start time of current window
	WindowSize  time.Duration // Size of the time window
}

// ClientRateLimiter provides rate limiting for multiple clients
type ClientRateLimiter struct {
	mutex           sync.RWMutex
	clients         map[string]*RateLimitEntry
	windowSize      time.Duration
	maxReqPerWindow int
	lastCleanup     time.Time
	cleanupInterval time.Duration
}

// NewClientRateLimiter creates a new client-based rate limiter
func NewClientRateLimiter(windowSize time.Duration, maxReqPerWindow int) *ClientRateLimiter {
	return &ClientRateLimiter{
		clients:         make(map[string]*RateLimitEntry),
		windowSize:      windowSize,
		maxReqPerWindow: maxReqPerWindow,
		lastCleanup:     time.Now(),
		cleanupInterval: 10 * time.Minute, // Clean up expired entries every 10 minutes
	}
}

// IsAllowed checks if a request from a client is allowed
func (rl *ClientRateLimiter) IsAllowed(clientID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Lazy cleanup
	if time.Since(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanupLocked()
	}

	// Get or create client entry
	entry, exists := rl.clients[clientID]
	if !exists {
		entry = &RateLimitEntry{
			Count:       0,
			MaxRequests: rl.maxReqPerWindow,
			WindowStart: time.Now(),
			WindowSize:  rl.windowSize,
		}
		rl.clients[clientID] = entry
	}

	// Reset window if needed
	if time.Since(entry.WindowStart) > entry.WindowSize {
		entry.Count = 0
		entry.WindowStart = time.Now()
	}

	// Check if allowed
	if entry.Count >= entry.MaxRequests {
		return false
	}

	// Increment counter and allow
	entry.Count++
	return true
}

// GetClientState returns the rate limit state for a client
func (rl *ClientRateLimiter) GetClientState(clientID string) *RateLimitEntry {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	entry, exists := rl.clients[clientID]
	if !exists {
		return nil
	}

	// Return a copy to prevent concurrent modification
	return &RateLimitEntry{
		Count:       entry.Count,
		MaxRequests: entry.MaxRequests,
		WindowStart: entry.WindowStart,
		WindowSize:  entry.WindowSize,
	}
}

// Cleanup removes expired entries
func (rl *ClientRateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.cleanupLocked()
}

// cleanupLocked removes expired entries (must be called with lock held)
func (rl *ClientRateLimiter) cleanupLocked() {
	now := time.Now()
	rl.lastCleanup = now

	for clientID, entry := range rl.clients {
		if now.Sub(entry.WindowStart) > entry.WindowSize*2 {
			delete(rl.clients, clientID)
		}
	}
}

// GetStats returns statistics about the rate limiter
func (rl *ClientRateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["clientCount"] = len(rl.clients)
	stats["windowSize"] = rl.windowSize.String()
	stats["maxRequestsPerWindow"] = rl.maxReqPerWindow

	// Add client-specific stats if there aren't too many
	if len(rl.clients) <= 100 {
		clientStats := make(map[string]map[string]interface{})

		for clientID, entry := range rl.clients {
			clientStats[clientID] = map[string]interface{}{
				"count":       entry.Count,
				"maxRequests": entry.MaxRequests,
				"windowStart": entry.WindowStart.Format(time.RFC3339),
				"remaining":   entry.MaxRequests - entry.Count,
			}
		}

		stats["clients"] = clientStats
	}

	return stats
}
