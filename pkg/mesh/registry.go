package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisRegistry implements ServiceRegistry using Redis
type RedisRegistry struct {
	client     *redis.Client
	keyPrefix  string
	expiration time.Duration
	watchers   []chan<- ServiceInfo
	watcherMu  sync.RWMutex
}

// RedisRegistryConfig configures the Redis registry
type RedisRegistryConfig struct {
	Client     *redis.Client
	KeyPrefix  string
	Expiration time.Duration
}

// NewRedisRegistry creates a new Redis-based service registry
func NewRedisRegistry(config RedisRegistryConfig) (ServiceRegistry, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("Redis client is required")
	}

	if config.KeyPrefix == "" {
		config.KeyPrefix = "mesh:service:"
	}

	if config.Expiration == 0 {
		config.Expiration = 1 * time.Minute
	}

	return &RedisRegistry{
		client:     config.Client,
		keyPrefix:  config.KeyPrefix,
		expiration: config.Expiration,
	}, nil
}

func (r *RedisRegistry) serviceKey(id ServiceID) string {
	return r.keyPrefix + string(id)
}

func (r *RedisRegistry) Register(ctx context.Context, info ServiceInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	key := r.serviceKey(info.ID)
	if err := r.client.Set(ctx, key, data, r.expiration).Err(); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Notify watchers
	r.watcherMu.RLock()
	for _, w := range r.watchers {
		select {
		case w <- info:
		default:
		}
	}
	r.watcherMu.RUnlock()

	// Start refresh loop
	go r.refreshLoop(ctx, info)

	return nil
}

func (r *RedisRegistry) Unregister(ctx context.Context, id ServiceID) error {
	key := r.serviceKey(id)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to unregister service: %w", err)
	}
	return nil
}

func (r *RedisRegistry) GetService(ctx context.Context, id ServiceID) (*ServiceInfo, error) {
	key := r.serviceKey(id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("service not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	var info ServiceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service info: %w", err)
	}

	return &info, nil
}

func (r *RedisRegistry) ListServices(ctx context.Context) ([]ServiceInfo, error) {
	pattern := r.keyPrefix + "*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var services []ServiceInfo
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var info ServiceInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}

		services = append(services, info)
	}

	return services, nil
}

func (r *RedisRegistry) Watch(ctx context.Context) (<-chan ServiceInfo, error) {
	ch := make(chan ServiceInfo, 100)

	r.watcherMu.Lock()
	r.watchers = append(r.watchers, ch)
	r.watcherMu.Unlock()

	go func() {
		<-ctx.Done()
		r.watcherMu.Lock()
		for i, w := range r.watchers {
			if w == ch {
				r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
				break
			}
		}
		r.watcherMu.Unlock()
		close(ch)
	}()

	return ch, nil
}

func (r *RedisRegistry) refreshLoop(ctx context.Context, info ServiceInfo) {
	ticker := time.NewTicker(r.expiration / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Register(ctx, info); err != nil {
				// Log error but continue trying
				continue
			}
		}
	}
}

// InMemoryRegistry implements ServiceRegistry with in-memory storage
type InMemoryRegistry struct {
	services  sync.Map
	watchers  []chan<- ServiceInfo
	watcherMu sync.RWMutex
}

// NewInMemoryRegistry creates a new in-memory service registry
func NewInMemoryRegistry() ServiceRegistry {
	return &InMemoryRegistry{}
}

func (r *InMemoryRegistry) Register(ctx context.Context, info ServiceInfo) error {
	r.services.Store(info.ID, info)

	r.watcherMu.RLock()
	for _, w := range r.watchers {
		select {
		case w <- info:
		default:
		}
	}
	r.watcherMu.RUnlock()

	return nil
}

func (r *InMemoryRegistry) Unregister(ctx context.Context, id ServiceID) error {
	r.services.Delete(id)
	return nil
}

func (r *InMemoryRegistry) GetService(ctx context.Context, id ServiceID) (*ServiceInfo, error) {
	if info, ok := r.services.Load(id); ok {
		service := info.(ServiceInfo)
		return &service, nil
	}
	return nil, fmt.Errorf("service not found: %s", id)
}

func (r *InMemoryRegistry) ListServices(ctx context.Context) ([]ServiceInfo, error) {
	var services []ServiceInfo
	r.services.Range(func(key, value interface{}) bool {
		services = append(services, value.(ServiceInfo))
		return true
	})
	return services, nil
}

func (r *InMemoryRegistry) Watch(ctx context.Context) (<-chan ServiceInfo, error) {
	ch := make(chan ServiceInfo, 100)

	r.watcherMu.Lock()
	r.watchers = append(r.watchers, ch)
	r.watcherMu.Unlock()

	go func() {
		<-ctx.Done()
		r.watcherMu.Lock()
		for i, w := range r.watchers {
			if w == ch {
				r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
				break
			}
		}
		r.watcherMu.Unlock()
		close(ch)
	}()

	return ch, nil
}
