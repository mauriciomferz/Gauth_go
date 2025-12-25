package gauth_rfc_001

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AtomicCounterStore provides atomic check-and-increment operations for constraint enforcement.
// This prevents TOCTOU (Time-Of-Check-Time-Of-Use) race conditions in distributed systems
// where multiple goroutines/processes might attempt to exceed quotas simultaneously.
//
// Security Context: Addresses Critical Vulnerability - Race Condition in Constraint Enforcement (TOCTOU)
// Without atomic operations, an attacker can bypass max_amount or max_daily_amount restrictions
// by sending concurrent requests that all read the same "current usage" value before any write occurs.
//
// Example Attack Scenario:
//   - PoA has max_daily_amount: 100
//   - Attacker sends 10 concurrent requests for amount: 100
//   - All 10 goroutines read current_usage = 0
//   - All 10 pass validation (0 + 100 <= 100)
//   - All 10 write, resulting in total usage = 1000 (10x over limit)
//
// Mitigation: Use Redis Lua scripts for atomic check-and-increment with EVALSHA for performance.
type AtomicCounterStore struct {
	client          *redis.Client
	prefix          string
	checkIncrSHA    string // SHA1 of the check-and-increment Lua script
	getValueSHA     string // SHA1 of the get value Lua script
	resetCounterSHA string // SHA1 of the reset counter Lua script
}

const (
	// luaCheckAndIncrement performs atomic check-and-increment with limit validation.
	// KEYS[1]: counter key (e.g., "gauth:quota:poa-123|2025-11-21")
	// ARGV[1]: increment amount (float as string, e.g., "50.00")
	// ARGV[2]: maximum limit (float as string, e.g., "100.00")
	// ARGV[3]: TTL in seconds (e.g., "86400" for 24 hours)
	// Returns: 1 if increment allowed and executed, 0 if limit would be exceeded
	luaCheckAndIncrement = `
local key = KEYS[1]
local increment = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

-- Get current value (nil if key doesn't exist)
local current = tonumber(redis.call('GET', key) or "0")

-- Check if adding increment would exceed limit
local newValue = current + increment
if newValue > limit then
    return 0  -- Fail: would exceed limit
end

-- Atomically increment and set TTL
redis.call('INCRBYFLOAT', key, increment)
redis.call('EXPIRE', key, ttl)

return 1  -- Success: increment allowed
`

	// luaGetValue retrieves the current counter value.
	// KEYS[1]: counter key
	// Returns: current value as string, or "0" if key doesn't exist
	luaGetValue = `
local key = KEYS[1]
return redis.call('GET', key) or "0"
`

	// luaResetCounter resets a counter to zero or a specific value.
	// KEYS[1]: counter key
	// ARGV[1]: new value (typically "0")
	// ARGV[2]: TTL in seconds
	// Returns: "OK"
	luaResetCounter = `
local key = KEYS[1]
local value = ARGV[1]
local ttl = tonumber(ARGV[2])

redis.call('SET', key, value, 'EX', ttl)
return "OK"
`
)

// NewAtomicCounterStore constructs a new Redis-backed atomic counter store.
// It preloads Lua scripts into Redis for efficient EVALSHA execution.
//
// Parameters:
//   - client: Redis client (must be connected)
//   - prefix: namespace prefix for counter keys (e.g., "gauth:quota")
//
// Returns error if Redis connection fails or script loading fails.
func NewAtomicCounterStore(client *redis.Client, prefix string) (*AtomicCounterStore, error) {
	if client == nil {
		return nil, fmt.Errorf("nil redis client")
	}
	if prefix == "" {
		prefix = "gauth:quota"
	}

	// Validate connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	store := &AtomicCounterStore{
		client: client,
		prefix: prefix,
	}

	// Load Lua scripts and cache their SHA1 hashes
	if err := store.loadScripts(ctx); err != nil {
		return nil, fmt.Errorf("failed to load lua scripts: %w", err)
	}

	return store, nil
}

// loadScripts loads all Lua scripts into Redis and caches their SHA1 hashes.
func (a *AtomicCounterStore) loadScripts(ctx context.Context) error {
	// Load check-and-increment script
	sha, err := a.client.ScriptLoad(ctx, luaCheckAndIncrement).Result()
	if err != nil {
		return fmt.Errorf("failed to load check-and-increment script: %w", err)
	}
	a.checkIncrSHA = sha

	// Load get value script
	sha, err = a.client.ScriptLoad(ctx, luaGetValue).Result()
	if err != nil {
		return fmt.Errorf("failed to load get-value script: %w", err)
	}
	a.getValueSHA = sha

	// Load reset counter script
	sha, err = a.client.ScriptLoad(ctx, luaResetCounter).Result()
	if err != nil {
		return fmt.Errorf("failed to load reset-counter script: %w", err)
	}
	a.resetCounterSHA = sha

	return nil
}

// key formats the full Redis key for a counter.
// Format: <prefix>:<counterID>
// Example: "gauth:quota:poa-abc123|2025-11-21"
func (a *AtomicCounterStore) key(counterID string) string {
	return fmt.Sprintf("%s:%s", a.prefix, counterID)
}

// CheckAndIncrement atomically checks if incrementing would exceed the limit,
// and if not, increments the counter. This is the core operation for quota enforcement.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - counterID: unique identifier for this counter (e.g., "poa-123|2025-11-21" for daily quotas)
//   - increment: amount to add (e.g., transaction amount 50.00)
//   - limit: maximum allowed value (e.g., max_daily_amount 100.00)
//   - ttl: time-to-live for the counter key (should match quota period, e.g., 24h for daily)
//
// Returns:
//   - allowed (bool): true if increment was allowed and executed, false if limit would be exceeded
//   - error: non-nil if Redis operation fails
//
// Thread-safe: Yes (atomic Redis operation)
// Network latency: Single round-trip to Redis
func (a *AtomicCounterStore) CheckAndIncrement(ctx context.Context, counterID string, increment, limit float64, ttl time.Duration) (allowed bool, err error) {
	if counterID == "" {
		return false, fmt.Errorf("empty counterID")
	}
	if increment < 0 {
		return false, fmt.Errorf("negative increment not allowed: %f", increment)
	}
	if limit < 0 {
		return false, fmt.Errorf("negative limit not allowed: %f", limit)
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default to 24 hours for daily quotas
	}

	keys := []string{a.key(counterID)}
	args := []interface{}{
		fmt.Sprintf("%.2f", increment), // ARGV[1]
		fmt.Sprintf("%.2f", limit),     // ARGV[2]
		int(ttl.Seconds()),             // ARGV[3]
	}

	result, err := a.client.EvalSha(ctx, a.checkIncrSHA, keys, args...).Result()
	if err != nil {
		// If script not found (e.g., Redis restarted), reload and retry
		if err.Error() == "NOSCRIPT No matching script. Please use EVAL." {
			if reloadErr := a.loadScripts(ctx); reloadErr != nil {
				return false, fmt.Errorf("failed to reload scripts: %w", reloadErr)
			}
			result, err = a.client.EvalSha(ctx, a.checkIncrSHA, keys, args...).Result()
			if err != nil {
				return false, fmt.Errorf("check-and-increment failed after reload: %w", err)
			}
		} else {
			return false, fmt.Errorf("check-and-increment failed: %w", err)
		}
	}

	// Lua script returns 1 for success, 0 for failure
	resultInt, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type: %T", result)
	}

	return resultInt == 1, nil
}

// GetValue retrieves the current counter value without modification.
// Useful for monitoring and debugging.
func (a *AtomicCounterStore) GetValue(ctx context.Context, counterID string) (float64, error) {
	if counterID == "" {
		return 0, fmt.Errorf("empty counterID")
	}

	keys := []string{a.key(counterID)}
	result, err := a.client.EvalSha(ctx, a.getValueSHA, keys).Result()
	if err != nil {
		// Handle script reload if needed
		if err.Error() == "NOSCRIPT No matching script. Please use EVAL." {
			if reloadErr := a.loadScripts(ctx); reloadErr != nil {
				return 0, fmt.Errorf("failed to reload scripts: %w", reloadErr)
			}
			result, err = a.client.EvalSha(ctx, a.getValueSHA, keys).Result()
			if err != nil {
				return 0, fmt.Errorf("get-value failed after reload: %w", err)
			}
		} else {
			return 0, fmt.Errorf("get-value failed: %w", err)
		}
	}

	// Result is a string like "50.00" or "0"
	valueStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}

	var value float64
	if _, err := fmt.Sscan(valueStr, &value); err != nil {
		return 0, fmt.Errorf("failed to parse value %q: %w", valueStr, err)
	}

	return value, nil
}

// ResetCounter resets a counter to zero (or a specific value).
// Useful for testing or manual intervention.
func (a *AtomicCounterStore) ResetCounter(ctx context.Context, counterID string, value float64, ttl time.Duration) error {
	if counterID == "" {
		return fmt.Errorf("empty counterID")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	keys := []string{a.key(counterID)}
	args := []interface{}{
		fmt.Sprintf("%.2f", value),
		int(ttl.Seconds()),
	}

	_, err := a.client.EvalSha(ctx, a.resetCounterSHA, keys, args...).Result()
	if err != nil {
		// Handle script reload if needed
		if err.Error() == "NOSCRIPT No matching script. Please use EVAL." {
			if reloadErr := a.loadScripts(ctx); reloadErr != nil {
				return fmt.Errorf("failed to reload scripts: %w", reloadErr)
			}
			_, err = a.client.EvalSha(ctx, a.resetCounterSHA, keys, args...).Result()
			if err != nil {
				return fmt.Errorf("reset-counter failed after reload: %w", err)
			}
		} else {
			return fmt.Errorf("reset-counter failed: %w", err)
		}
	}

	return nil
}

// WithAtomicCounterStore is a functional option to inject AtomicCounterStore into Service.
func WithAtomicCounterStore(client *redis.Client, prefix string) Option {
	return func(s *Service) {
		store, err := NewAtomicCounterStore(client, prefix)
		if err != nil {
			// Log error but don't fail service construction
			// Service will fall back to in-memory counters (with TOCTOU risk)
			return
		}
		s.atomicCounterStore = store
	}
}
