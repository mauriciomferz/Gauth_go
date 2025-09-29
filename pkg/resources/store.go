package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
)

// RedisConfigStore implements ConfigStore using Redis
type RedisConfigStore struct {
	client    *redis.Client
	keyPrefix string
	watchers  []chan<- ServiceConfig
	mu        sync.RWMutex
}

// RedisConfigStoreConfig contains Redis configuration
type RedisConfigStoreConfig struct {
	Address   string
	Password  string
	DB        int
	KeyPrefix string
}

// NewRedisConfigStore creates a new Redis-based configuration store
func NewRedisConfigStore(cfg RedisConfigStoreConfig) (*RedisConfigStore, error) {
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "service:config:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &RedisConfigStore{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
	}, nil
}

func (s *RedisConfigStore) configKey(serviceType ServiceType) string {
	return s.keyPrefix + string(serviceType)
}

func (s *RedisConfigStore) Load(ctx context.Context, serviceType ServiceType) (*ServiceConfig, error) {
	data, err := s.client.Get(ctx, s.configKey(serviceType)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("configuration not found for service %s", serviceType)
		}
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	var config ServiceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return &config, nil
}

func (s *RedisConfigStore) Save(ctx context.Context, config ServiceConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	key := s.configKey(config.Type)
	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Notify watchers
	s.mu.RLock()
	for _, w := range s.watchers {
		select {
		case w <- config:
		default:
		}
	}
	s.mu.RUnlock()

	return nil
}

func (s *RedisConfigStore) List(ctx context.Context) ([]ServiceConfig, error) {
	pattern := s.keyPrefix + "*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list configurations: %w", err)
	}

	var configs []ServiceConfig
	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var config ServiceConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func (s *RedisConfigStore) Watch(ctx context.Context) (<-chan ServiceConfig, error) {
	ch := make(chan ServiceConfig, 100)

	s.mu.Lock()
	s.watchers = append(s.watchers, ch)
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		for i, w := range s.watchers {
			if w == ch {
				s.watchers = append(s.watchers[:i], s.watchers[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		close(ch)
	}()

	// Subscribe to Redis keyspace notifications
	pubsub := s.client.PSubscribe(ctx, "__keyspace@*__:"+s.keyPrefix+"*")
	defer pubsub.Close()

	go func() {
		for msg := range pubsub.Channel() {
			key := msg.Payload
			if msg.Pattern != "set" {
				continue
			}

			data, err := s.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			var config ServiceConfig
			if err := json.Unmarshal(data, &config); err != nil {
				continue
			}

			select {
			case ch <- config:
			default:
			}
		}
	}()

	return ch, nil
}

// Close closes the Redis connection
func (s *RedisConfigStore) Close() error {
	return s.client.Close()
}

// InMemoryConfigStore implements ConfigStore with in-memory storage
type InMemoryConfigStore struct {
	configs  sync.Map
	watchers []chan<- ServiceConfig
	mu       sync.RWMutex
}

// NewInMemoryConfigStore creates a new in-memory configuration store
func NewInMemoryConfigStore() *InMemoryConfigStore {
	return &InMemoryConfigStore{}
}

func (s *InMemoryConfigStore) Load(ctx context.Context, serviceType ServiceType) (*ServiceConfig, error) {
	if config, ok := s.configs.Load(serviceType); ok {
		cfg := config.(ServiceConfig)
		return &cfg, nil
	}
	return nil, fmt.Errorf("configuration not found for service %s", serviceType)
}

func (s *InMemoryConfigStore) Save(ctx context.Context, config ServiceConfig) error {
	s.configs.Store(config.Type, config)

	s.mu.RLock()
	for _, w := range s.watchers {
		select {
		case w <- config:
		default:
		}
	}
	s.mu.RUnlock()

	return nil
}

func (s *InMemoryConfigStore) List(ctx context.Context) ([]ServiceConfig, error) {
	var configs []ServiceConfig
	s.configs.Range(func(_, value interface{}) bool {
		configs = append(configs, value.(ServiceConfig))
		return true
	})
	return configs, nil
}

func (s *InMemoryConfigStore) Watch(ctx context.Context) (<-chan ServiceConfig, error) {
	ch := make(chan ServiceConfig, 100)

	s.mu.Lock()
	s.watchers = append(s.watchers, ch)
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		for i, w := range s.watchers {
			if w == ch {
				s.watchers = append(s.watchers[:i], s.watchers[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}
