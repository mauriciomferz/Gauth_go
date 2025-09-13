package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	LastAccess time.Time
}

// DistributedCache demonstrates resilience patterns in a distributed caching system
type DistributedCache struct {
	nodes         map[string]*CacheNode
	partitioner   *ConsistentHash
	circuitConfig circuit.Options
	retryConfig   resilience.RetryStrategy
	limiterConfig *ratelimit.Config
}

// CacheNode represents a single node in the distributed cache
type CacheNode struct {
	ID       string
	data     map[string]CacheEntry
	breaker  *circuit.CircuitBreaker
	limiter  ratelimit.Algorithm
	retry    *resilience.Retry
	bulkhead *resilience.Bulkhead
	mu       sync.RWMutex
}

// ConsistentHash implements consistent hashing for cache key distribution
type ConsistentHash struct {
	hashRing []string
	mu       sync.RWMutex
}

func NewDistributedCache(nodeIDs []string) *DistributedCache {
	cache := &DistributedCache{
		nodes:       make(map[string]*CacheNode),
		partitioner: NewConsistentHash(nodeIDs),
		circuitConfig: circuit.Options{
			FailureThreshold: 3,
			ResetTimeout:     5 * time.Second,
			HalfOpenLimit:    2,
		},
		retryConfig: resilience.RetryStrategy{
			MaxAttempts:     3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		},
		limiterConfig: &ratelimit.Config{
			RequestsPerSecond: 100,
			WindowSize:        1,
			BurstSize:         20,
		},
	}

	// Initialize cache nodes
	for _, id := range nodeIDs {
		cache.nodes[id] = &CacheNode{
			ID:       id,
			data:     make(map[string]CacheEntry),
			breaker:  circuit.NewCircuitBreaker(cache.circuitConfig),
			limiter:  ratelimit.WrapTokenBucket(cache.limiterConfig),
			retry:    resilience.NewRetry(cache.retryConfig),
			bulkhead: resilience.NewBulkhead(10),
		}
	}

	return cache
}

func NewConsistentHash(nodes []string) *ConsistentHash {
	ch := &ConsistentHash{
		hashRing: make([]string, 0, len(nodes)),
	}
	ch.hashRing = append(ch.hashRing, nodes...)
	return ch
}

func (ch *ConsistentHash) GetNode(key string) string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.hashRing) == 0 {
		return ""
	}

	// Simple hash function for demonstration
	hash := 0
	for i := 0; i < len(key); i++ {
		hash = 31*hash + int(key[i])
	}
	index := hash % len(ch.hashRing)
	if index < 0 {
		index += len(ch.hashRing)
	}
	return ch.hashRing[index]
}

func (dc *DistributedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	nodeID := dc.partitioner.GetNode(key)
	node, exists := dc.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Execute with bulkhead pattern
	return node.bulkhead.Execute(ctx, func() error {
		// First check rate limiter
		if err := node.limiter.Allow(ctx, "write"); err != nil {
			return fmt.Errorf("rate limit exceeded on node %s: %w", nodeID, err)
		}

		// Use retry with circuit breaker
		return node.retry.Execute(ctx, func() error {
			return node.breaker.Execute(func() error {
				node.mu.Lock()
				defer node.mu.Unlock()

				node.data[key] = CacheEntry{
					Value:      value,
					ExpiresAt:  time.Now().Add(ttl),
					LastAccess: time.Now(),
				}
				return nil
			})
		})
	})
}

func (dc *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
	nodeID := dc.partitioner.GetNode(key)
	node, exists := dc.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	var result interface{}
	err := node.bulkhead.Execute(ctx, func() error {
		if err := node.limiter.Allow(ctx, "read"); err != nil {
			return fmt.Errorf("rate limit exceeded on node %s: %w", nodeID, err)
		}

		return node.retry.Execute(ctx, func() error {
			return node.breaker.Execute(func() error {
				node.mu.RLock()
				defer node.mu.RUnlock()

				entry, exists := node.data[key]
				if !exists {
					return fmt.Errorf("key %s not found", key)
				}

				if time.Now().After(entry.ExpiresAt) {
					delete(node.data, key)
					return fmt.Errorf("key %s expired", key)
				}

				result = entry.Value
				return nil
			})
		})
	})

	return result, err
}

func main() {
	// Initialize cache with three nodes
	cache := NewDistributedCache([]string{"node1", "node2", "node3"})
	ctx := context.Background()

	// Simulate multiple concurrent operations
	var wg sync.WaitGroup
	operations := 50

	fmt.Println("\nStarting Distributed Cache Demo...")
	fmt.Println("----------------------------------------")
	fmt.Println("Configuration:")
	fmt.Println("- Nodes: node1, node2, node3")
	fmt.Println("- Rate Limit: 100 req/s, burst: 20")
	fmt.Println("- Circuit Breaker: 3 failures, 5s reset")
	fmt.Println("- Retry: 3 attempts, exponential backoff")
	fmt.Println("----------------------------------------")

	// Writer goroutines
	for i := 1; i <= operations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			value := fmt.Sprintf("value%d", id)

			start := time.Now()
			err := cache.Set(ctx, key, value, 1*time.Minute)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("[Write %d] Failed after %v: %v\n", id, duration.Round(time.Millisecond), err)
			} else {
				fmt.Printf("[Write %d] Completed in %v\n", id, duration.Round(time.Millisecond))
			}
		}(i)
		time.Sleep(10 * time.Millisecond) // Slight delay between operations
	}

	// Reader goroutines
	for i := 1; i <= operations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)

			start := time.Now()
			value, err := cache.Get(ctx, key)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("[Read %d] Failed after %v: %v\n", id, duration.Round(time.Millisecond), err)
			} else {
				fmt.Printf("[Read %d] Got %v in %v\n", id, value, duration.Round(time.Millisecond))
			}
		}(i)
		time.Sleep(5 * time.Millisecond) // Slight delay between operations
	}

	wg.Wait()
	fmt.Println("\nDistributed Cache demo completed!")
}
