package pdp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/redis"
)

const (
	// Prefix for global cache version
	redisCacheVersionKey = "pdp:cache:version"
	// Prefix for cache entries: pdp:cache:v{version}:{hash}
	redisCacheKeyPrefix = "pdp:cache:v%d:%s"
	// TTL for cache version key (should be persistent, but safe fallback)
	redisVersionTTL = 365 * 24 * time.Hour

	// Secondary indices for granular invalidation
	redisIndexSubjectKey  = "pdp:index:subject:%s"
	redisIndexResourceKey = "pdp:index:resource:%s"
	redisIndexActionKey   = "pdp:index:action:%s"
)

// RedisDecisionCache implements DecisionCache using Redis for distributed caching.
//
// Key Features:
// - Global Invalidation: Uses a version counter. Incrementing version conceptually invalidates all old keys.
// - Granular Invalidation: Maintains Sets of cache keys per Subject/Resource/Action.
// - TTL: Entries expire automatically.
type RedisDecisionCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisDecisionCache creates a new Redis-backed decision cache.
func NewRedisDecisionCache(client *redis.Client, ttl time.Duration) *RedisDecisionCache {
	return &RedisDecisionCache{
		client: client,
		ttl:    ttl,
	}
}

// getCacheVersion retrieves the current global cache version.
// Defaults to 1 if not set.
func (r *RedisDecisionCache) getCacheVersion(ctx context.Context) (int64, error) {
	// Optimistic: assuming version doesn't change every millisecond.
	// In critical path, we could cache this locally for short duration (e.g. 1s)
	// but to ensure strict consistency on InvalidateAll, we read from Redis.
	val, err := r.client.Get(ctx, redisCacheVersionKey)
	if err != nil {
		// If key not found, initialize to 1
		if err.Error() == "key not found" {
			// Using SetNX effectively
			if setErr := r.client.Set(ctx, redisCacheVersionKey, 1, redisVersionTTL); setErr == nil {
				return 1, nil
			}
			// Add retry logic or simple fallback
		}
		return 1, nil // Fallback/Default
	}

	// Parse int
	var ver int64
	_, err = fmt.Sscanf(val, "%d", &ver)
	if err != nil {
		return 1, nil
	}
	return ver, nil
}

// makeRedisKey generates the versioned cache key.
func (r *RedisDecisionCache) makeRedisKey(ctx context.Context, req Request) (string, string, error) {
	ver, err := r.getCacheVersion(ctx)
	if err != nil {
		return "", "", err
	}

	// Make deterministic hash of the request (same logic as InMemory)
	// We reuse the logic but need to implement it here or export makeKey from cache.go
	// Since makeKey in cache.go is unexported, duplicating logic to avoid dependency issues for now.
	// In a real refactor, `makeKey` should be a shared utility.

	type keyStruct struct {
		Subject    string            `json:"subject"`
		Action     string            `json:"action"`
		Resource   string            `json:"resource"`
		Attributes map[string]string `json:"attributes"`
		TimeUnix   int64             `json:"time_unix"`
	}

	ks := keyStruct{
		Subject:    req.Subject,
		Action:     req.Action,
		Resource:   req.Resource,
		Attributes: req.Attributes,
		TimeUnix:   req.Time.Unix(),
	}

	data, _ := json.Marshal(ks)
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf(redisCacheKeyPrefix, ver, hashStr), hashStr, nil
}

// Get retrieves a cached decision.
func (r *RedisDecisionCache) Get(ctx context.Context, req Request) (Decision, bool, error) {
	key, _, err := r.makeRedisKey(ctx, req)
	if err != nil {
		return Decision{}, false, err
	}

	val, err := r.client.Get(ctx, key)
	if err != nil {
		if err.Error() == "key not found" {
			return Decision{}, false, nil
		}
		return Decision{}, false, err
	}

	var dec Decision
	if err := json.Unmarshal([]byte(val), &dec); err != nil {
		return Decision{}, false, err
	}

	if dec.Metadata == nil {
		dec.Metadata = make(map[string]string)
	}
	dec.Metadata["cache_hit"] = "true"
	dec.Metadata["cache_backend"] = "redis"

	return dec, true, nil
}

// Set stores a decision and updates indices.
func (r *RedisDecisionCache) Set(ctx context.Context, req Request, decision Decision) error {
	key, _, err := r.makeRedisKey(ctx, req)
	if err != nil {
		return err
	}

	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}

	// 1. Set the cache entry
	if setErr := r.client.Set(ctx, key, string(data), r.ttl); setErr != nil {
		return setErr
	}

	// 2. Add to indices (fire and forget / best effort? or robust?)
	// We'll treat this as best effort to avoid heavy latency punishment.
	// Index format: Set of Cache Keys.
	// Note: Indices also need to handle Global Versioning or expiration.
	// If Global Version increments, these indices point to stale keys (which won't be found or will be wrong version).
	// Actually, if version changes, the Key computed by makeRedisKey changes.
	// So indices will just point to dead keys. That's fine.

	// Add to indices asynchronously or pipeline? Pipeline is better.
	// We are limited by the client wrapper interface which doesn't expose Pipeline explicitly for arbitrary commands easily.
	// We will just do sequential calls for this MVP.

	// We need access to SAdd (Set Add). The wrapper pkg/redis/client.go doesn't expose SAdd directly in the interface visible.
	// Looking at client.go, it wraps redis.Client and exposes some helpers but also GetClient().
	// We can use r.client.GetClient() to access full API.

	rawClient := r.client.GetClient()
	pipe := rawClient.Pipeline()

	// Add key to Subject Index
	pipe.SAdd(ctx, fmt.Sprintf(redisIndexSubjectKey, req.Subject), key)
	pipe.Expire(ctx, fmt.Sprintf(redisIndexSubjectKey, req.Subject), r.ttl) // Refresh index TTL

	// Add key to Resource Index
	pipe.SAdd(ctx, fmt.Sprintf(redisIndexResourceKey, req.Resource), key)
	pipe.Expire(ctx, fmt.Sprintf(redisIndexResourceKey, req.Resource), r.ttl)

	// Add key to Action Index
	pipe.SAdd(ctx, fmt.Sprintf(redisIndexActionKey, req.Action), key)
	pipe.Expire(ctx, fmt.Sprintf(redisIndexActionKey, req.Action), r.ttl)

	_, err = pipe.Exec(ctx)
	return err
}

// InvalidateAll increments the global cache version.
func (r *RedisDecisionCache) InvalidateAll(ctx context.Context) error {
	rawClient := r.client.GetClient()
	// Increment version
	return rawClient.Incr(ctx, redisCacheVersionKey).Err()
}

// InvalidateSubject removes keys associated with subject.
func (r *RedisDecisionCache) InvalidateSubject(ctx context.Context, subject string) error {
	return r.invalidateByIndex(ctx, fmt.Sprintf(redisIndexSubjectKey, subject))
}

// InvalidateResource removes keys associated with resource.
func (r *RedisDecisionCache) InvalidateResource(ctx context.Context, resource string) error {
	return r.invalidateByIndex(ctx, fmt.Sprintf(redisIndexResourceKey, resource))
}

// InvalidateAction removes keys associated with action.
func (r *RedisDecisionCache) InvalidateAction(ctx context.Context, action string) error {
	return r.invalidateByIndex(ctx, fmt.Sprintf(redisIndexActionKey, action))
}

// invalidateByIndex reads keys from a Set and Deletes them.
func (r *RedisDecisionCache) invalidateByIndex(ctx context.Context, indexKey string) error {
	rawClient := r.client.GetClient()

	// Get all keys from index
	keys, err := rawClient.SMembers(ctx, indexKey).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all Keys
	if err := rawClient.Del(ctx, keys...).Err(); err != nil {
		return err
	}

	// Delete the index itself
	return rawClient.Del(ctx, indexKey).Err()
}

// GetMetrics returns basic metrics (stub for Redis as connection stats are on client).
func (r *RedisDecisionCache) GetMetrics() PDPCacheMetrics {
	// redis/client.go has GetStats()
	stats := r.client.GetStats()

	return PDPCacheMetrics{
		Capacity: -1, // Unlimited/Redis managed
		Size:     0,  // Expensive to count
		Lookups:  0,  // Not tracking locally
		Hits:     uint64(stats.Hits),
		Misses:   uint64(stats.Misses),
		TTL:      r.ttl.String(),
		Backend:  "redis",
	}
}

// Close is a no-op as Client lifecycle is managed externally usually,
// but if we owned it we would close it. Here we assume we share the client?
// Implementing interface strictness.
func (r *RedisDecisionCache) Close() error {
	return nil
}
