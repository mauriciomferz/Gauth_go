package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/cache"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	// DefaultChannel is the Redis Pub/Sub channel for cache invalidation
	DefaultChannel = "gauth:pdp:cache:invalidation"
)

// DistributedDecisionCache implements a hybrid L1/L2 cache with distributed invalidation.
type DistributedDecisionCache struct {
	l1      DecisionCache // Local LRU
	l2      cache.Cache   // Global Redis
	client  *redis.Client // Redis client for Pub/Sub
	channel string
	nodeID  string
	stopCh  chan struct{}
	mu      sync.Mutex
	metrics AuthorizationCacheMetrics
	logger  func(string, ...interface{})
	sf      singleflight.Group // RR-001: Request Coalescing
}

// InvalidationMessage is sent over Redis Pub/Sub
type InvalidationMessage struct {
	NodeID string `json:"node_id"`
	Key    string `json:"key"`    // Empty means InvalidateAll
	Action string `json:"action"` // "invalidate" or "clear"
}

// NewDistributedDecisionCache creates a new hybrid cache.
func NewDistributedDecisionCache(l1 DecisionCache, l2 cache.Cache, redisClient *redis.Client, nodeID string) *DistributedDecisionCache {
	c := &DistributedDecisionCache{
		l1:      l1,
		l2:      l2,
		client:  redisClient,
		channel: DefaultChannel,
		nodeID:  nodeID,
		stopCh:  make(chan struct{}),
		logger:  func(f string, a ...interface{}) { fmt.Printf("[dist-cache] "+f+"\n", a...) },
	}

	if redisClient != nil {
		go c.subscribe()
	}

	return c
}

func (c *DistributedDecisionCache) Get(key string) (AuthorizationCacheEntry, bool) {
	// 1. Check L1
	if entry, ok := c.l1.Get(key); ok {
		return entry, true
	}

	// 2. Check L2 with Singleflight (RR-001)
	if c.l2 != nil {
		// Do ensures that multiple concurrent calls for the same key result in only one execution
		v, err, _ := c.sf.Do(key, func() (interface{}, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			data, err := c.l2.Get(ctx, key)
			if err == nil && data != nil {
				var entry AuthorizationCacheEntry
				if err := json.Unmarshal(data, &entry); err == nil {
					return entry, nil
				}
			}
			return nil, err
		})

		if err == nil && v != nil {
			entry := v.(AuthorizationCacheEntry)
			// Backfill L1
			c.l1.Set(key, entry)
			return entry, true
		}
	}

	return AuthorizationCacheEntry{}, false
}

func (c *DistributedDecisionCache) Set(key string, entry AuthorizationCacheEntry) {
	// 1. Set L1
	c.l1.Set(key, entry)

	// 2. Set L2
	if c.l2 != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		data, _ := json.Marshal(entry)
		_ = c.l2.Set(ctx, key, data, 1*time.Hour) // Use a reasonable default TTL
	}
}

func (c *DistributedDecisionCache) Invalidate(key string) {
	// 1. Local L1
	c.l1.Invalidate(key)

	// 2. Clear L2
	if c.l2 != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_ = c.l2.Delete(ctx, key)
	}

	// 3. Notify Others
	c.publish(InvalidationMessage{NodeID: c.nodeID, Key: key, Action: "invalidate"})
}

func (c *DistributedDecisionCache) InvalidateAll() {
	// 1. Local L1
	c.l1.InvalidateAll()

	// 2. Clear L2 (Warning: potentially expensive if using pattern, but PDP cache keys should have a prefix)
	// For now we just clear L1. In a real system we'd use a prefix and DeletePattern.
	if c.l2 != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = c.l2.DeletePattern(ctx, "*") // PDP cache keys should probably have a prefix
	}

	// 3. Notify Others
	c.publish(InvalidationMessage{NodeID: c.nodeID, Action: "clear"})
}

func (c *DistributedDecisionCache) MarkStale(key string) {
	c.Invalidate(key)
}

func (c *DistributedDecisionCache) Size() int {
	return c.l1.Size()
}

func (c *DistributedDecisionCache) Snapshot() AuthorizationCacheMetrics {
	m := c.l1.Snapshot()
	// Merge L2 info if relevant? Snapshot is mostly for hits/misses which we track in L1 getters if we want,
	// but currently we just return L1's snapshot plus some indicators.
	return m
}

func (c *DistributedDecisionCache) Close() {
	close(c.stopCh)
}

func (c *DistributedDecisionCache) publish(msg InvalidationMessage) {
	if c.client == nil {
		return
	}
	data, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	c.client.Publish(ctx, c.channel, data)
}

func (c *DistributedDecisionCache) subscribe() {
	pubsub := c.client.Subscribe(context.Background(), c.channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			var inv InvalidationMessage
			if err := json.Unmarshal([]byte(msg.Payload), &inv); err != nil {
				continue
			}

			// Ignore our own messages
			if inv.NodeID == c.nodeID {
				continue
			}

			if inv.Action == "clear" {
				c.l1.InvalidateAll()
			} else if inv.Action == "invalidate" {
				c.l1.Invalidate(inv.Key)
			}
		case <-c.stopCh:
			return
		}
	}
}
