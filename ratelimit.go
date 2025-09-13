
// Package gauth provides rate limiting functionality for the GAuth protocol.
package gauth

import (
    "context"
    "sync"
    "time"
)

// RateLimiter provides rate limiting functionality

	mu             sync.RWMutex

	requestCount   int    "sync"

	lastReset      time.Time

	resetInterval  time.Duration    "github.com/Gimel-Foundation/gauth/pkg/gauth"

	requestLimit   int

	onRateExceeded func())    "time"import (//

}



// NewRateLimiter creates a new rate limiter with the given configuration

func NewRateLimiter(requestLimit int, resetInterval time.Duration, onRateExceeded func()) *RateLimiter {    "github.com/Gimel-Foundation/gauth/pkg/gauth"    "sync"

	return &RateLimiter{

		requestLimit:   requestLimit,)

		resetInterval: resetInterval,

		lastReset:     time.Now(),    "time"// Rate limiting features include:////

		onRateExceeded: onRateExceeded,

	}// RateLimitEntry tracks rate limit state for a client

}

type RateLimitEntry struct {

// Allow checks if a request should be allowed under the current rate limit

func (r *RateLimiter) Allow(ctx context.Context) bool {    Count       int       // Number of requests in current window

	r.mu.Lock()

	defer r.mu.Unlock()    WindowStart time.Time // Start time of current window    "github.com/Gimel-Foundation/gauth/pkg/gauth"//   - Token bucket algorithm implementation



	// Check if we need to reset the counter    LastAccess  time.Time // Last request time

	now := time.Now()

	if now.Sub(r.lastReset) >= r.resetInterval {    BurstTokens int       // Available burst tokens)

		r.requestCount = 0

		r.lastReset = now    WindowSize  time.Duration // Duration of rate limit window

	}

    MaxRequests int          // Maximum requests per window//   - Configurable limits and burst sizes// Rate limiting features include:// Rate limiting features include:

	// Check if we're at the limit

	if r.requestCount >= r.requestLimit {}

		if r.onRateExceeded != nil {

			r.onRateExceeded()// RateLimitEntry tracks rate limit state for a client

		}

		return false// RateLimiter implements configurable rate limiting

	}

type RateLimiter struct {type RateLimitEntry struct {//   - Thread-safe operation

	// Increment the counter and allow the request

	r.requestCount++    config  gauth.RateLimitConfig

	return true

}    entries map[string]*RateLimitEntry    Count        int           // Number of requests in current window



// Reset resets the rate limiter state    mutex   sync.RWMutex

func (r *RateLimiter) Reset() {

	r.mu.Lock()}    WindowStart  time.Time     // Start time of current window//   - Per-client rate tracking//   - Token bucket algorithm implementation//   - Token bucket algorithm implementation

	defer r.mu.Unlock()

    LastAccess   time.Time     // Last request time

	r.requestCount = 0

	r.lastReset = time.Now()    BurstTokens  int           // Available burst tokens//   - Distributed rate limiting support

}

    WindowSize   time.Duration // Duration of rate limit window

// RemainingRequests returns the number of remaining requests allowed

func (r *RateLimiter) RemainingRequests() int {    MaxRequests  int          // Maximum requests per windowpackage gauth//   - Configurable limits and burst sizes//   - Configurable limits and burst sizes

	r.mu.RLock()

	defer r.mu.RUnlock()}



	// Check if we need to reset

	if time.Since(r.lastReset) >= r.resetInterval {

		return r.requestLimit// RateLimiter implements configurable rate limiting

	}

type RateLimiter struct {import (//   - Thread-safe operation//   - Thread-safe operation

	remaining := r.requestLimit - r.requestCount

	if remaining < 0 {    config  gauth.RateLimitConfig

		return 0

	}    entries map[string]*RateLimitEntry    "sync"

	return remaining

}    mutex   sync.RWMutex



// TimeUntilReset returns the duration until the rate limit resets}    "time"//   - Per-client rate tracking//   - Per-client rate tracking

func (r *RateLimiter) TimeUntilReset() time.Duration {

	r.mu.RLock()

	defer r.mu.RUnlock()

    "github.com/Gimel-Foundation/gauth/pkg/gauth"//   - Distributed rate limiting support//   - Distributed rate limiting support

	nextReset := r.lastReset.Add(r.resetInterval)

	return time.Until(nextReset))

}

package gauth

// SetRequestLimit updates the request limit

func (r *RateLimiter) SetRequestLimit(limit int) {// RateLimitEntry tracks rate limit state for a client

	r.mu.Lock()

	defer r.mu.Unlock()type RateLimitEntry struct {

	r.requestLimit = limit

}    Count        int           // Number of requests in current window



// SetResetInterval updates the reset interval    WindowStart  time.Time     // Start time of current windowimport (

func (r *RateLimiter) SetResetInterval(interval time.Duration) {

	r.mu.Lock()    LastAccess   time.Time     // Last request time

	defer r.mu.Unlock()

	r.resetInterval = interval    BurstTokens  int           // Available burst tokens    "sync"    "sync"

}

    WindowSize   time.Duration // Duration of rate limit window

// SetRateExceededCallback updates the callback for rate limit exceeded events

func (r *RateLimiter) SetRateExceededCallback(callback func()) {    MaxRequests  int          // Maximum requests per window    "time"    "time"

	r.mu.Lock()

	defer r.mu.Unlock()}

	r.onRateExceeded = callback

}

// RateLimiter implements configurable rate limiting

type RateLimiter struct {    "github.com/Gimel-Foundation/gauth/pkg/gauth"    "github.com/Gimel-Foundation/gauth/pkg/gauth"

    config  gauth.RateLimitConfig

    entries map[string]*RateLimitEntry))

    mutex   sync.RWMutex

}



// NewRateLimiter creates a rate limiter with the given configuration// RateLimitEntry tracks rate limit state for a client// RateLimitEntry tracks rate limit state for a client

func NewRateLimiter(cfg gauth.RateLimitConfig) *RateLimiter {

    if cfg.WindowSize <= 0 {type RateLimitEntry struct {type RateLimitEntry struct {

        cfg.WindowSize = 60 // Default 60-second window

    }    Count        int           // Number of requests in current window    Count        int           // Number of requests in current window

    if cfg.BurstSize <= 0 {

        cfg.BurstSize = cfg.RequestsPerSecond // Default burst = rate    WindowStart  time.Time     // Start time of current window    WindowStart  time.Time     // Start time of current window

    }

    if cfg.RequestsPerSecond <= 0 {    LastAccess   time.Time     // Last request time    LastAccess   time.Time     // Last request time

        cfg.RequestsPerSecond = 60 // Default 60 requests per minute

    }    BurstTokens  int           // Available burst tokens    BurstTokens  int           // Available burst tokens



    return &RateLimiter{    WindowSize   time.Duration // Duration of rate limit window    WindowSize   time.Duration // Duration of rate limit window

        config:  cfg,

        entries: make(map[string]*RateLimitEntry),    MaxRequests  int          // Maximum requests per window    MaxRequests  int          // Maximum requests per window

    }

}}}



// IsAllowed checks if a client can make another request

func (rl *RateLimiter) IsAllowed(clientID string) bool {

    rl.mutex.Lock()// RateLimiter implements configurable rate limiting// RateLimiter implements configurable rate limiting

    defer rl.mutex.Unlock()

type RateLimiter struct {type RateLimiter struct {

    now := time.Now()

    windowDuration := time.Duration(rl.config.WindowSize) * time.Second    config  gauth.RateLimitConfig    config  gauth.RateLimitConfig



    entry, exists := rl.entries[clientID]    entries map[string]*RateLimitEntry    entries map[string]*RateLimitEntry

    if !exists {

        // Initialize new client    mutex   sync.RWMutex    mutex   sync.RWMutex

        entry = &RateLimitEntry{

            Count:       1,}}

            WindowStart: now,

            LastAccess:  now,

            BurstTokens: rl.config.BurstSize,

            WindowSize:  windowDuration,// NewRateLimiter creates a rate limiter with the given configuration// NewRateLimiter creates a rate limiter with the given configuration

            MaxRequests: rl.config.RequestsPerSecond,

        }func NewRateLimiter(cfg gauth.RateLimitConfig) *RateLimiter {func NewRateLimiter(cfg gauth.RateLimitConfig) *RateLimiter {

        rl.entries[clientID] = entry

        return true    if cfg.WindowSize <= 0 {    if cfg.WindowSize <= 0 {

    }

        cfg.WindowSize = 60 // Default 60-second window        cfg.WindowSize = 60 // Default 60-second window

    // Reset window if expired

    if now.Sub(entry.WindowStart) >= windowDuration {    }    }

        entry.Count = 1

        entry.WindowStart = now    if cfg.BurstSize <= 0 {    if cfg.BurstSize <= 0 {

        entry.LastAccess = now

        entry.BurstTokens = rl.config.BurstSize        cfg.BurstSize = cfg.RequestsPerSecond // Default burst = rate        cfg.BurstSize = cfg.RequestsPerSecond // Default burst = rate

        return true

    }    }    }



    // Check if within rate limit    if cfg.RequestsPerSecond <= 0 {    if cfg.RequestsPerSecond <= 0 {

    if entry.Count < entry.MaxRequests {

        entry.Count++        cfg.RequestsPerSecond = 60 // Default 60 requests per minute        cfg.RequestsPerSecond = 60 // Default 60 requests per minute

        entry.LastAccess = now

        return true    }    }

    }



    // Try using burst token

    if entry.BurstTokens > 0 {    return &RateLimiter{    return &RateLimiter{

        entry.BurstTokens--

        entry.Count++        config:  cfg,        config:  cfg,

        entry.LastAccess = now

        return true        entries: make(map[string]*RateLimitEntry),        entries:     make(map[string]*RateLimitEntry),

    }

    }        maxRequests: maxRequests,

    return false

}}        windowSize:  windowSize,



// GetClientState returns rate limit state for a client    }

func (rl *RateLimiter) GetClientState(clientID string) *RateLimitEntry {

    rl.mutex.RLock()// IsAllowed checks if a client can make another request}

    defer rl.mutex.RUnlock()

    return rl.entries[clientID]func (rl *RateLimiter) IsAllowed(clientID string) bool {

}

    rl.mutex.Lock()func (rl *RateLimiter) IsAllowed(identifier string) bool {

// Cleanup removes expired client entries

func (rl *RateLimiter) Cleanup() {    defer rl.mutex.Unlock()    rl.mutex.Lock()

    rl.mutex.Lock()

    defer rl.mutex.Unlock()    defer rl.mutex.Unlock()



    now := time.Now()    now := time.Now()    now := time.Now()

    windowDuration := time.Duration(rl.config.WindowSize) * time.Second

    windowDuration := time.Duration(rl.config.WindowSize) * time.Second    entry, exists := rl.entries[identifier]

    for clientID, entry := range rl.entries {

        if now.Sub(entry.LastAccess) > windowDuration*2 {    if !exists {

            delete(rl.entries, clientID)

        }    entry, exists := rl.entries[clientID]        rl.entries[identifier] = &RateLimitEntry{

    }

}    if !exists {            Count:       1,



// GetStats returns rate limiting statistics        // Initialize new client            WindowStart: now,

func (rl *RateLimiter) GetStats() map[string]interface{} {

    rl.mutex.RLock()        entry = &RateLimitEntry{            LastAccess:  now,

    defer rl.mutex.RUnlock()

            Count:       1,        }

    stats := make(map[string]interface{})

    stats["total_clients"] = len(rl.entries)            WindowStart: now,        return true

    stats["requests_per_second"] = rl.config.RequestsPerSecond

    stats["burst_size"] = rl.config.BurstSize            LastAccess:  now,    }

    stats["window_size"] = rl.config.WindowSize

            BurstTokens: rl.config.BurstSize,    if now.Sub(entry.WindowStart) > rl.windowSize {

    active := 0

    blocked := 0            WindowSize:  windowDuration,        entry.Count = 1

    now := time.Now()

    for _, entry := range rl.entries {            MaxRequests: rl.config.RequestsPerSecond,        entry.WindowStart = now

        if now.Sub(entry.LastAccess) < time.Duration(rl.config.WindowSize)*time.Second {

            active++        }        entry.LastAccess = now

            if entry.Count >= entry.MaxRequests && entry.BurstTokens == 0 {

                blocked++        rl.entries[clientID] = entry        return true

            }

        }        return true    }

    }

    stats["active_clients"] = active    }    if entry.Count >= rl.maxRequests {

    stats["blocked_clients"] = blocked

        entry.LastAccess = now

    return stats

}    // Reset window if expired        return false

    if now.Sub(entry.WindowStart) >= windowDuration {    }

        entry.Count = 1    entry.Count++

        entry.WindowStart = now    entry.LastAccess = now

        entry.LastAccess = now    return true

        entry.BurstTokens = rl.config.BurstSize}

        return true

    }func (rl *RateLimiter) CleanupExpired() {

    rl.mutex.Lock()

    // Check if within rate limit    defer rl.mutex.Unlock()

    if entry.Count < entry.MaxRequests {    now := time.Now()

        entry.Count++    for identifier, entry := range rl.entries {

        entry.LastAccess = now        if now.Sub(entry.LastAccess) > rl.windowSize*2 {

        return true            delete(rl.entries, identifier)

    }        }

    }

    // Try using burst token}
    if entry.BurstTokens > 0 {
        entry.BurstTokens--
        entry.Count++
        entry.LastAccess = now
        return true
    }

    return false
}

// GetClientState returns rate limit state for a client
func (rl *RateLimiter) GetClientState(clientID string) *RateLimitEntry {
    rl.mutex.RLock()
    defer rl.mutex.RUnlock()
    return rl.entries[clientID]
}

// Cleanup removes expired client entries
func (rl *RateLimiter) Cleanup() {
    rl.mutex.Lock()
    defer rl.mutex.Unlock()

    now := time.Now()
    windowDuration := time.Duration(rl.config.WindowSize) * time.Second

    for clientID, entry := range rl.entries {
        if now.Sub(entry.LastAccess) > windowDuration*2 {
            delete(rl.entries, clientID)
        }
    }
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
    rl.mutex.RLock()
    defer rl.mutex.RUnlock()

    stats := make(map[string]interface{})
    stats["total_clients"] = len(rl.entries)
    stats["requests_per_second"] = rl.config.RequestsPerSecond
    stats["burst_size"] = rl.config.BurstSize
    stats["window_size"] = rl.config.WindowSize

    active := 0
    blocked := 0
    now := time.Now()
    for _, entry := range rl.entries {
        if now.Sub(entry.LastAccess) < time.Duration(rl.config.WindowSize)*time.Second {
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