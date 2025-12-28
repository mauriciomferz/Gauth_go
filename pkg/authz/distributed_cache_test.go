package authz

import (
	"context"
	"testing"
	"time"

	"sync"
	"sync/atomic"

	"github.com/alicebob/miniredis/v2"
	"github.com/mauriciomferz/Gauth_go/pkg/cache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedDecisionCache_Basic(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	l2, err := cache.NewRedisCache(&cache.Config{
		RedisURL: "redis://" + mr.Addr(),
		Type:     "redis",
	})
	require.NoError(t, err)

	l1 := NewLRUDecisionCache(10)
	dc := NewDistributedDecisionCache(l1, l2, client, "node-1")

	key := "test-key"
	entry := AuthorizationCacheEntry{
		Decision: Decision{Allow: true, Reason: "ok"},
	}

	// 1. Set
	dc.Set(key, entry)

	// 2. Clear L1 manually to force L2 hit
	l1.Invalidate(key)

	// 3. Get (should hit L2 and backfill L1)
	got, found := dc.Get(key)
	assert.True(t, found)
	assert.Equal(t, entry.Decision.Allow, got.Decision.Allow)

	// 4. Verify L1 backfilled
	_, foundL1 := l1.Get(key)
	assert.True(t, foundL1)

	// 5. Invalidate
	dc.Invalidate(key)
	_, foundGet := dc.Get(key)
	assert.False(t, foundGet)

	// Verify L2 cleared
	exists, _ := l2.Exists(context.Background(), key)
	assert.False(t, exists)
}

func TestDistributedDecisionCache_DistributedInvalidation(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client1 := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	client2 := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	l2_1, _ := cache.NewRedisCache(&cache.Config{RedisURL: "redis://" + mr.Addr(), Type: "redis"})
	l2_2, _ := cache.NewRedisCache(&cache.Config{RedisURL: "redis://" + mr.Addr(), Type: "redis"})

	l1_1 := NewLRUDecisionCache(10)
	l1_2 := NewLRUDecisionCache(10)

	node1 := NewDistributedDecisionCache(l1_1, l2_1, client1, "node-1")
	node2 := NewDistributedDecisionCache(l1_2, l2_2, client2, "node-2")

	key := "shared-key"
	entry := AuthorizationCacheEntry{Decision: Decision{Allow: true}}

	// 1. Node 1 sets the key (Node 2 won't have it in L1 yet)
	node1.Set(key, entry)

	// 2. Node 2 gets the key (Hits L2, fills its L1)
	_, found := node2.Get(key)
	assert.True(t, found)

	_, inL1_2 := l1_2.Get(key)
	assert.True(t, inL1_2)

	// 3. Node 1 invalidates the key
	node1.Invalidate(key)

	// Wait for Pub/Sub (usually very fast but let's give it a moment)
	time.Sleep(100 * time.Millisecond)

	// 4. Verify Node 2 L1 is invalidated via Pub/Sub
	_, foundNode2 := l1_2.Get(key)
	assert.False(t, foundNode2, "Node 2 L1 should be invalidated via Pub/Sub")
}

func TestDistributedDecisionCache_InvalidateAll(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client1 := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	client2 := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	l2_1, _ := cache.NewRedisCache(&cache.Config{RedisURL: "redis://" + mr.Addr(), Type: "redis"})
	l2_2, _ := cache.NewRedisCache(&cache.Config{RedisURL: "redis://" + mr.Addr(), Type: "redis"})

	l1_1 := NewLRUDecisionCache(10)
	l1_2 := NewLRUDecisionCache(10)

	node1 := NewDistributedDecisionCache(l1_1, l2_1, client1, "node-1")
	node2 := NewDistributedDecisionCache(l1_2, l2_2, client2, "node-2")

	// Set multiple keys
	node1.Set("k1", AuthorizationCacheEntry{Decision: Decision{Allow: true}})
	node1.Set("k2", AuthorizationCacheEntry{Decision: Decision{Allow: true}})

	// Fill node2 L1
	node2.Get("k1")
	node2.Get("k2")

	// Node 1 InvalidateAll
	node1.InvalidateAll()

	time.Sleep(100 * time.Millisecond)

	// Verify Node 2 is cleared
	assert.Equal(t, 0, node2.Size())

	// Verify Node 1 is cleared
	assert.Equal(t, 0, node1.Size())
}

func TestDistributedDecisionCache_Singleflight(t *testing.T) {
	// We need a custom mock or a way to count calls to L2.
	// Since RedisCache uses a redis.Client, we can use a wrapper or middleware,
	// BUT for this test, let's use a simpler approach: use a Slow L2 simulation.
	// Since DistributedDecisionCache takes an interface l2 cache.Cache, lets mock it.

	mockL2 := &mockCache{
		data: map[string][]byte{
			"key": []byte(`{"decision":{"allow":true}}`),
		},
		delay: 100 * time.Millisecond,
	}

	dc := NewDistributedDecisionCache(NewLRUDecisionCache(10), mockL2, nil, "node-1")

	// Simulate concurrent stampede
	var wg sync.WaitGroup
	concurrency := 10
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_, found := dc.Get("key")
			assert.True(t, found)
		}()
	}
	wg.Wait()

	// All 10 requests should succeed, but Get should be called only once
	assert.Equal(t, int32(1), mockL2.calls, "Expected L2 Get to be called exactly once due to singleflight")
}

type mockCache struct {
	data  map[string][]byte
	calls int32
	delay time.Duration
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	atomic.AddInt32(&m.calls, 1)
	time.Sleep(m.delay)
	val, ok := m.data[key]
	if !ok {
		return nil, nil // nil, nil for cache miss (redis style) or error?
		// RedisCache returns ErrNil (backend dependent) but wrapper returns nil, nil or nil, err depending on imp
		// Looking at distributed_cache.go: data, err := c.l2.Get(ctx, key) ... if err == nil && data != nil
	}
	return val, nil
}
func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (m *mockCache) Delete(ctx context.Context, key string) error            { return nil }
func (m *mockCache) DeletePattern(ctx context.Context, pattern string) error { return nil }
func (m *mockCache) Exists(ctx context.Context, key string) (bool, error)    { return true, nil }
func (m *mockCache) Close() error                                            { return nil }
func (m *mockCache) GetStats(ctx context.Context) (*cache.Stats, error) {
	return &cache.Stats{}, nil
}
func (m *mockCache) Ping(ctx context.Context) error { return nil }
