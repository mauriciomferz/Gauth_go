package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RedisStore implements token storage using Redis
type RedisStore struct {
	client    redis.UniversalClient
	metrics   *RedisMetrics
	expiry    time.Duration
	keyPrefix string
}

// RedisMetrics tracks Redis operations
type RedisMetrics struct {
	storeOps      *prometheus.CounterVec
	storeErrors   *prometheus.CounterVec
	storeDuration *prometheus.HistogramVec
	activeTokens  prometheus.Gauge
	keySize       prometheus.Gauge
}

func newRedisMetrics(namespace string) *RedisMetrics {
	return &RedisMetrics{
		storeOps: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "token_store_operations_total",
				Help:      "Total number of token store operations",
			},
			[]string{"operation", "status"},
		),
		storeErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "token_store_errors_total",
				Help:      "Total number of token store errors",
			},
			[]string{"operation", "error_type"},
		),
		storeDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "token_store_operation_duration_seconds",
				Help:      "Duration of token store operations",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"operation"},
		),
		activeTokens: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "active_tokens",
			Help:      "Number of active tokens",
		}),
		keySize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "token_store_size_bytes",
			Help:      "Size of token store in bytes",
		}),
	}
}

// NewRedisStore creates a new Redis-backed token store
func NewRedisStore(opts *redis.UniversalOptions, expiry time.Duration) (*RedisStore, error) {
	client := redis.NewUniversalClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	store := &RedisStore{
		client:    client,
		metrics:   newRedisMetrics("token_store"),
		expiry:    expiry,
		keyPrefix: "token:",
	}

	// Start metrics collector
	go store.collectMetrics(context.Background())

	return store, nil
}

func (s *RedisStore) key(id string) string {
	return s.keyPrefix + id
}

func (s *RedisStore) Save(ctx context.Context, t *token.Token) error {
	timer := prometheus.NewTimer(s.metrics.storeDuration.WithLabelValues("save"))
	defer timer.ObserveDuration()

	data, err := json.Marshal(t)
	if err != nil {
		s.metrics.storeErrors.WithLabelValues("save", "marshal").Inc()
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	err = s.client.Set(ctx, s.key(t.ID), data, s.expiry).Err()
	if err != nil {
		s.metrics.storeErrors.WithLabelValues("save", "redis").Inc()
		return fmt.Errorf("failed to save token: %w", err)
	}

	s.metrics.storeOps.WithLabelValues("save", "success").Inc()
	s.metrics.activeTokens.Inc()
	return nil
}

func (s *RedisStore) Get(ctx context.Context, id string) (*token.Token, error) {
	timer := prometheus.NewTimer(s.metrics.storeDuration.WithLabelValues("get"))
	defer timer.ObserveDuration()

	data, err := s.client.Get(ctx, s.key(id)).Bytes()
	if err == redis.Nil {
		s.metrics.storeOps.WithLabelValues("get", "not_found").Inc()
		return nil, token.ErrTokenNotFound
	}
	if err != nil {
		s.metrics.storeErrors.WithLabelValues("get", "redis").Inc()
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var t token.Token
	if err := json.Unmarshal(data, &t); err != nil {
		s.metrics.storeErrors.WithLabelValues("get", "unmarshal").Inc()
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	s.metrics.storeOps.WithLabelValues("get", "success").Inc()
	return &t, nil
}

func (s *RedisStore) Delete(ctx context.Context, id string) error {
	timer := prometheus.NewTimer(s.metrics.storeDuration.WithLabelValues("delete"))
	defer timer.ObserveDuration()

	err := s.client.Del(ctx, s.key(id)).Err()
	if err != nil {
		s.metrics.storeErrors.WithLabelValues("delete", "redis").Inc()
		return fmt.Errorf("failed to delete token: %w", err)
	}

	s.metrics.storeOps.WithLabelValues("delete", "success").Inc()
	s.metrics.activeTokens.Dec()
	return nil
}

// collectMetrics periodically updates store metrics
func (s *RedisStore) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.updateMetrics(ctx); err != nil {
				log.Printf("Failed to update metrics: %v", err)
			}
		}
	}
}

func (s *RedisStore) updateMetrics(ctx context.Context) error {
	// Get total number of keys
	keys, err := s.client.Keys(ctx, s.keyPrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to count keys: %w", err)
	}
	s.metrics.activeTokens.Set(float64(len(keys)))

	// Get memory usage (not used, just for demonstration)
	_, err = s.client.Info(ctx, "memory").Result()
	if err != nil {
		return fmt.Errorf("failed to get memory info: %w", err)
	}
	// Parse memory info and update metrics (simplified)
	s.metrics.keySize.Set(float64(len(keys) * 100)) // Estimate

	return nil
}

func main() {
	ctx := context.Background()

	// Example: Create Redis Cluster store
	redisOpts := &redis.UniversalOptions{
		Addrs: []string{
			"localhost:6379",
			"localhost:6380",
			"localhost:6381",
		},
		Password:    "secret",
		DB:          0,
		MaxRetries:  3,
		PoolSize:    10,
		DialTimeout: 5 * time.Second,
	}

	store, err := NewRedisStore(redisOpts, 24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create Redis store: %v", err)
	}

	// Example token
	t := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   "user123",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Save token
	if err := store.Save(ctx, t); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}
	fmt.Printf("Saved token: %s\n", t.ID)

	// Retrieve token
	retrieved, err := store.Get(ctx, t.ID)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	fmt.Printf("Retrieved token: %s\n", retrieved.ID)

	// Delete token
	if err := store.Delete(ctx, t.ID); err != nil {
		log.Fatalf("Failed to delete token: %v", err)
	}
	fmt.Println("Deleted token")
}
