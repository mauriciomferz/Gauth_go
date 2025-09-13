package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStorage implements the Storage interface using Redis
type RedisStorage struct {
	client     *redis.Client
	keyPrefix  string
	expiration time.Duration
}

// RedisConfig holds configuration for Redis storage
type RedisConfig struct {
	// Addresses of Redis servers
	Addresses []string

	// Password for authentication
	Password string

	// DB number to use
	DB int

	// KeyPrefix for Redis keys
	KeyPrefix string

	// DefaultExpiration for entries
	DefaultExpiration time.Duration

	// MaxRetries for operations
	MaxRetries int

	// MinRetryBackoff for retry delays
	MinRetryBackoff time.Duration

	// MaxRetryBackoff for retry delays
	MaxRetryBackoff time.Duration
}

// NewRedisStorage creates a new Redis-backed storage
func NewRedisStorage(config RedisConfig) (*RedisStorage, error) {
	if len(config.Addresses) == 0 {
		return nil, fmt.Errorf("no Redis addresses provided")
	}

	client := redis.NewClient(&redis.Options{
		Addr:            config.Addresses[0], // TODO: Support cluster
		Password:        config.Password,
		DB:              config.DB,
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: config.MinRetryBackoff,
		MaxRetryBackoff: config.MaxRetryBackoff,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{
		client:     client,
		keyPrefix:  config.KeyPrefix,
		expiration: config.DefaultExpiration,
	}, nil
}

// Store implements the Storage interface
func (rs *RedisStorage) Store(ctx context.Context, entry *Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Store by ID
	key := rs.entryKey(entry.ID)
	if err := rs.client.Set(ctx, key, data, rs.expiration).Err(); err != nil {
		return fmt.Errorf("failed to store entry: %w", err)
	}

	// Add to type index
	typeKey := rs.typeKey(entry.Type)
	if err := rs.client.SAdd(ctx, typeKey, entry.ID).Err(); err != nil {
		return fmt.Errorf("failed to index by type: %w", err)
	}

	// Add to actor index
	if entry.ActorID != "" {
		actorKey := rs.actorKey(entry.ActorID)
		if err := rs.client.SAdd(ctx, actorKey, entry.ID).Err(); err != nil {
			return fmt.Errorf("failed to index by actor: %w", err)
		}
	}

	// Add to chain index
	if entry.ChainID != "" {
		chainKey := rs.chainKey(entry.ChainID)
		if err := rs.client.SAdd(ctx, chainKey, entry.ID).Err(); err != nil {
			return fmt.Errorf("failed to index by chain: %w", err)
		}
	}

	// Add to time index
	timeKey := rs.timeKey(entry.Timestamp)
	if err := rs.client.SAdd(ctx, timeKey, entry.ID).Err(); err != nil {
		return fmt.Errorf("failed to index by time: %w", err)
	}

	return nil
}

// Search implements the Storage interface
func (rs *RedisStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error) {
	// Start with all IDs or filtered by type
	var ids []string
	var err error

	if len(filter.Types) > 0 {
		// Union of all type sets
		typeKeys := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			typeKeys[i] = rs.typeKey(t)
		}
		ids, err = rs.client.SUnion(ctx, typeKeys...).Result()
	} else {
		// Get all entry IDs
		pattern := rs.entryKey("*")
		var cursor uint64
		var keys []string
		for {
			keys, cursor, err = rs.client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return nil, fmt.Errorf("failed to scan entries: %w", err)
			}
			for _, key := range keys {
				ids = append(ids, rs.extractID(key))
			}
			if cursor == 0 {
				break
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get entry IDs: %w", err)
	}

	// Filter by actor
	if len(filter.ActorIDs) > 0 {
		actorKeys := make([]string, len(filter.ActorIDs))
		for i, actor := range filter.ActorIDs {
			actorKeys[i] = rs.actorKey(actor)
		}
		actorIDs, err := rs.client.SUnion(ctx, actorKeys...).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get actor entries: %w", err)
		}
		ids = intersection(ids, actorIDs)
	}

	// Filter by time range
	if filter.TimeRange != nil {
		timeKeys := rs.timeKeysInRange(filter.TimeRange.Start, filter.TimeRange.End)
		timeIDs, err := rs.client.SUnion(ctx, timeKeys...).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get time range entries: %w", err)
		}
		ids = intersection(ids, timeIDs)
	}

	// Get entries
	var entries []*Entry
	pipe := rs.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(ids))
	for i, id := range ids {
		cmds[i] = pipe.Get(ctx, rs.entryKey(id))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get entries: %w", err)
	}

	for _, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get entry data: %w", err)
		}

		var entry Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entry: %w", err)
		}

		if rs.matchesFilter(&entry, filter) {
			entries = append(entries, &entry)
		}

		if filter.Limit > 0 && len(entries) >= filter.Limit {
			break
		}
	}

	return entries, nil
}

// GetByID implements the Storage interface
func (rs *RedisStorage) GetByID(ctx context.Context, id string) (*Entry, error) {
	data, err := rs.client.Get(ctx, rs.entryKey(id)).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	return &entry, nil
}

// GetChain implements the Storage interface
func (rs *RedisStorage) GetChain(ctx context.Context, chainID string) ([]*Entry, error) {
	return rs.Search(ctx, &Filter{
		ChainID: chainID,
	})
}

// Cleanup implements the Storage interface
func (rs *RedisStorage) Cleanup(ctx context.Context, before time.Time) error {
	pattern := rs.keyPrefix + "time:*"
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = rs.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan time keys: %w", err)
		}

		for _, key := range keys {
			t := rs.extractTime(key)
			if t.Before(before) {
				ids, err := rs.client.SMembers(ctx, key).Result()
				if err != nil {
					continue
				}

				// Delete entries and indices
				pipe := rs.client.Pipeline()
				for _, id := range ids {
					pipe.Del(ctx, rs.entryKey(id))
				}
				pipe.Del(ctx, key)
				if _, err := pipe.Exec(ctx); err != nil {
					return fmt.Errorf("failed to cleanup entries: %w", err)
				}
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// Close implements io.Closer
func (rs *RedisStorage) Close() error {
	return rs.client.Close()
}

// Helper methods

func (rs *RedisStorage) entryKey(id string) string {
	return fmt.Sprintf("%sentry:%s", rs.keyPrefix, id)
}

func (rs *RedisStorage) typeKey(typ Type) string {
	return fmt.Sprintf("%stype:%s", rs.keyPrefix, typ)
}

func (rs *RedisStorage) actorKey(actorID string) string {
	return fmt.Sprintf("%sactor:%s", rs.keyPrefix, actorID)
}

func (rs *RedisStorage) chainKey(chainID string) string {
	return fmt.Sprintf("%schain:%s", rs.keyPrefix, chainID)
}

func (rs *RedisStorage) timeKey(t time.Time) string {
	return fmt.Sprintf("%stime:%s", rs.keyPrefix, t.Format("2006-01-02"))
}

func (rs *RedisStorage) extractID(key string) string {
	return key[len(rs.entryKey("")):]
}

func (rs *RedisStorage) extractTime(key string) time.Time {
	timeStr := key[len(rs.keyPrefix+"time:"):]
	t, _ := time.Parse("2006-01-02", timeStr)
	return t
}

func (rs *RedisStorage) timeKeysInRange(start, end time.Time) []string {
	var keys []string
	for t := start; !t.After(end); t = t.AddDate(0, 0, 1) {
		keys = append(keys, rs.timeKey(t))
	}
	return keys
}

func (rs *RedisStorage) matchesFilter(entry *Entry, filter *Filter) bool {
	if filter == nil {
		return true
	}

	// Filter logic same as FileStorage
	return true // TODO: Implement filter matching
}

// Helper function to compute set intersection
func intersection(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range a {
		m[item] = true
	}

	var result []string
	for _, item := range b {
		if m[item] {
			result = append(result, item)
		}
	}
	return result
}
